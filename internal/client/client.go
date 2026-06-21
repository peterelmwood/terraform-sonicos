// Package client is a small typed HTTP client for the SonicOS 7+ REST API.
//
// It handles the three SonicOS-specific concerns that any automation against
// the appliance must deal with:
//
//   - Session auth. The API takes HTTP Basic credentials at POST
//     /api/sonicos/auth and then maintains a session cookie. Only one API
//     session is allowed at a time and it requires a full-admin identity, so
//     callers should use a dedicated automation account and serialize access.
//   - The pending/commit model. Mutating calls (POST/PUT/DELETE) do not take
//     effect immediately; they accumulate in a pending configuration that is
//     activated with POST /api/sonicos/config/pending. CommitPending performs
//     that commit; the provider decides when to call it.
//   - The named-collection JSON shape. Every object lives under a named array
//     and single objects are addressed with a /name/<name> path segment.
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// Client is a SonicOS API client. It is safe for sequential use; because the
// appliance permits only a single concurrent API session, callers must not
// share one Client across parallel writers without external locking.
type Client struct {
	httpClient *http.Client
	baseURL    string
	username   string
	password   string
}

// Config configures a Client.
type Config struct {
	// Host is the appliance management host, optionally with a scheme and
	// port, e.g. "https://192.0.2.1:8443" or "fw.example.com".
	Host string
	// Username and Password are full-admin API credentials.
	Username string
	Password string
	// Insecure disables TLS certificate verification, which is common for
	// appliances using the default self-signed certificate.
	Insecure bool
	// Timeout bounds each HTTP request. Zero applies a sane default.
	Timeout time.Duration
}

// New builds a Client from cfg. It does not contact the appliance; call Login
// to establish a session.
func New(cfg Config) (*Client, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("host is required")
	}

	host := cfg.Host
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	u, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("invalid host %q: %w", cfg.Host, err)
	}
	base := strings.TrimRight(u.String(), "/") + "/api/sonicos"

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("creating cookie jar: %w", err)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.Insecure}, //nolint:gosec // user-controlled for self-signed appliance certs
	}

	return &Client{
		httpClient: &http.Client{Jar: jar, Timeout: timeout, Transport: transport},
		baseURL:    base,
		username:   cfg.Username,
		password:   cfg.Password,
	}, nil
}

// APIError is returned when the appliance responds with a non-2xx status. It
// surfaces the SonicOS status envelope when one is present.
type APIError struct {
	StatusCode int
	Method     string
	Path       string
	Messages   []string
	Body       string
}

func (e *APIError) Error() string {
	if len(e.Messages) > 0 {
		return fmt.Sprintf("sonicos API %s %s: %d: %s", e.Method, e.Path, e.StatusCode, strings.Join(e.Messages, "; "))
	}
	body := e.Body
	if len(body) > 300 {
		body = body[:300]
	}
	return fmt.Sprintf("sonicos API %s %s: %d: %s", e.Method, e.Path, e.StatusCode, body)
}

// Login establishes an API session using Basic auth. The override flag bumps an
// existing session if one is held by the same account, which avoids "session
// already in use" failures when a previous run did not log out cleanly.
func (c *Client) Login(ctx context.Context) error {
	body := map[string]bool{"override": true}
	req, err := c.newRequest(ctx, http.MethodPost, "/auth", body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.username, c.password)
	return c.do(req, nil)
}

// Logout ends the API session. Errors are returned but are typically safe to
// ignore on cleanup paths.
func (c *Client) Logout(ctx context.Context) error {
	req, err := c.newRequest(ctx, http.MethodDelete, "/auth", nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// CommitPending activates the pending configuration. This is the SonicOS
// transaction boundary: nothing written since the last commit is live until
// this succeeds.
func (c *Client) CommitPending(ctx context.Context) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/config/pending", nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// newRequest builds a JSON request against the API base URL. A nil body sends
// no payload.
func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encoding request body: %w", err)
		}
		reader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// do executes req and, on a 2xx response, decodes the body into out when out is
// non-nil. Non-2xx responses are turned into an *APIError.
func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Method:     req.Method,
			Path:       req.URL.Path,
			Body:       string(data),
		}
		var status statusResponse
		if json.Unmarshal(data, &status) == nil {
			for _, info := range status.Status.Info {
				apiErr.Messages = append(apiErr.Messages, info.Message)
			}
		}
		return apiErr
	}

	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

// IsNotFound reports whether err is an *APIError with a 404 status, used by
// resources to detect drift (an object deleted out-of-band).
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}
