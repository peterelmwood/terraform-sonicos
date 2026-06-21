package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestClient builds a Client pointed at a test server. The server URL
// already includes the scheme, so New leaves it intact.
func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c, err := New(Config{Host: srv.URL, Username: "admin", Password: "secret"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestLoginSendsBasicAuthAndOverride(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/sonicos/auth" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "secret" {
			t.Errorf("missing or wrong basic auth: user=%q pass=%q ok=%v", user, pass, ok)
		}
		var body map[string]bool
		_ = json.NewDecoder(r.Body).Decode(&body)
		if !body["override"] {
			t.Errorf("expected override=true, got %v", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := newTestClient(t, srv).Login(context.Background()); err != nil {
		t.Fatalf("Login: %v", err)
	}
}

func TestCommitPending(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/sonicos/config/pending" {
			called = true
			w.WriteHeader(http.StatusOK)
			return
		}
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	if err := newTestClient(t, srv).CommitPending(context.Background()); err != nil {
		t.Fatalf("CommitPending: %v", err)
	}
	if !called {
		t.Fatal("commit endpoint was not called")
	}
}

func TestCreateAddressObjectIPv4Payload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sonicos/address-objects/ipv4" || r.Method != http.MethodPost {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		data, _ := io.ReadAll(r.Body)
		var body addressObjectsBody
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if len(body.AddressObjects) != 1 || body.AddressObjects[0].IPv4 == nil {
			t.Fatalf("unexpected wrapper shape: %s", data)
		}
		got := body.AddressObjects[0].IPv4
		if got.Name != "web1" || got.Zone != "LAN" || got.Host == nil || got.Host.IP != "192.0.2.10" {
			t.Errorf("unexpected object: %+v", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).CreateAddressObjectIPv4(context.Background(), AddressObjectIPv4{
		Name: "web1", Zone: "LAN", Host: &AddressObjectHost{IP: "192.0.2.10"},
	})
	if err != nil {
		t.Fatalf("CreateAddressObjectIPv4: %v", err)
	}
}

func TestGetAddressObjectIPv4NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":{"success":false,"info":[{"level":"error","code":"E_NOT_FOUND","message":"no such object"}]}}`))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).GetAddressObjectIPv4(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsNotFound(err) {
		t.Fatalf("expected IsNotFound, got %v", err)
	}
}

func TestGetAddressObjectIPv4Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sonicos/address-objects/ipv4/name/web1" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(addressObjectsBody{AddressObjects: []addressObjectWrapper{
			{IPv4: &AddressObjectIPv4{Name: "web1", Zone: "LAN", Host: &AddressObjectHost{IP: "192.0.2.10"}}},
		}})
	}))
	defer srv.Close()

	obj, err := newTestClient(t, srv).GetAddressObjectIPv4(context.Background(), "web1")
	if err != nil {
		t.Fatalf("GetAddressObjectIPv4: %v", err)
	}
	if obj.Host == nil || obj.Host.IP != "192.0.2.10" {
		t.Errorf("unexpected object: %+v", obj)
	}
}

func TestCreateAccessRuleRecoversUUID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/sonicos/access-rules/ipv4":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/api/sonicos/access-rules/ipv4":
			_ = json.NewEncoder(w).Encode(accessRulesBody{AccessRules: []accessRuleWrapper{
				{IPv4: &AccessRule{UUID: "abc-123", Name: "allow-web", From: "LAN", To: "WAN", Action: "allow"}},
			}})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	uuid, err := newTestClient(t, srv).CreateAccessRule(context.Background(), AccessRule{Name: "allow-web", From: "LAN", To: "WAN", Action: "allow"})
	if err != nil {
		t.Fatalf("CreateAccessRule: %v", err)
	}
	if uuid != "abc-123" {
		t.Fatalf("expected uuid abc-123, got %q", uuid)
	}
}

func TestAPIErrorMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status":{"success":false,"info":[{"level":"error","code":"E_BAD","message":"invalid zone"}]}}`))
	}))
	defer srv.Close()

	err := newTestClient(t, srv).CreateZone(context.Background(), Zone{Name: "X"})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if len(apiErr.Messages) != 1 || apiErr.Messages[0] != "invalid zone" {
		t.Fatalf("unexpected messages: %v", apiErr.Messages)
	}
}
