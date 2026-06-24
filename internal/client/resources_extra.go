package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// CRUD methods for the NAT policy, interface, and IPv6 object types. These
// follow the same conventions as resources.go: POST to create, PUT to a
// selector path to update, GET to read, DELETE to remove, and none of them
// commit (the caller drives CommitPending).

// ----------------------------------------------------------------------------
// Address objects (IPv6)
// ----------------------------------------------------------------------------

const addressObjectsV6Path = "/address-objects/ipv6"

// CreateAddressObjectIPv6 stages a new IPv6 address object.
func (c *Client) CreateAddressObjectIPv6(ctx context.Context, obj AddressObjectIPv6) error {
	body := addressObjectsV6Body{AddressObjects: []addressObjectV6Wrapper{{IPv6: &obj}}}
	req, err := c.newRequest(ctx, http.MethodPost, addressObjectsV6Path, body)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// UpdateAddressObjectIPv6 stages a change to an existing IPv6 address object.
func (c *Client) UpdateAddressObjectIPv6(ctx context.Context, name string, obj AddressObjectIPv6) error {
	body := addressObjectsV6Body{AddressObjects: []addressObjectV6Wrapper{{IPv6: &obj}}}
	req, err := c.newRequest(ctx, http.MethodPut, addressObjectsV6Path+"/name/"+url.PathEscape(name), body)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// GetAddressObjectIPv6 fetches a single IPv6 address object by name.
func (c *Client) GetAddressObjectIPv6(ctx context.Context, name string) (*AddressObjectIPv6, error) {
	path := addressObjectsV6Path + "/name/" + url.PathEscape(name)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var body addressObjectsV6Body
	if err := c.do(req, &body); err != nil {
		return nil, err
	}
	for _, w := range body.AddressObjects {
		if w.IPv6 != nil && w.IPv6.Name == name {
			return w.IPv6, nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Method: http.MethodGet, Path: path, Messages: []string{fmt.Sprintf("ipv6 address object %q not found", name)}}
}

// DeleteAddressObjectIPv6 stages deletion of an IPv6 address object.
func (c *Client) DeleteAddressObjectIPv6(ctx context.Context, name string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, addressObjectsV6Path+"/name/"+url.PathEscape(name), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// ----------------------------------------------------------------------------
// Access rules (IPv6)
// ----------------------------------------------------------------------------

const accessRulesV6Path = "/access-rules/ipv6"

// CreateAccessRuleIPv6 stages a new IPv6 access rule and returns the assigned UUID.
func (c *Client) CreateAccessRuleIPv6(ctx context.Context, rule AccessRule) (string, error) {
	body := accessRulesV6Body{AccessRules: []accessRuleV6Wrapper{{IPv6: &rule}}}
	req, err := c.newRequest(ctx, http.MethodPost, accessRulesV6Path, body)
	if err != nil {
		return "", err
	}
	if err := c.do(req, nil); err != nil {
		return "", err
	}
	created, err := c.findAccessRuleV6ByName(ctx, rule.Name)
	if err != nil {
		return "", err
	}
	return created.UUID, nil
}

// UpdateAccessRuleIPv6 stages a change to an existing IPv6 access rule by UUID.
func (c *Client) UpdateAccessRuleIPv6(ctx context.Context, uuid string, rule AccessRule) error {
	body := accessRulesV6Body{AccessRules: []accessRuleV6Wrapper{{IPv6: &rule}}}
	req, err := c.newRequest(ctx, http.MethodPut, accessRulesV6Path+"/uuid/"+url.PathEscape(uuid), body)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// GetAccessRuleIPv6 fetches a single IPv6 access rule by UUID.
func (c *Client) GetAccessRuleIPv6(ctx context.Context, uuid string) (*AccessRule, error) {
	path := accessRulesV6Path + "/uuid/" + url.PathEscape(uuid)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var body accessRulesV6Body
	if err := c.do(req, &body); err != nil {
		return nil, err
	}
	for _, w := range body.AccessRules {
		if w.IPv6 != nil && w.IPv6.UUID == uuid {
			return w.IPv6, nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Method: http.MethodGet, Path: path, Messages: []string{fmt.Sprintf("ipv6 access rule %q not found", uuid)}}
}

// DeleteAccessRuleIPv6 stages deletion of an IPv6 access rule by UUID.
func (c *Client) DeleteAccessRuleIPv6(ctx context.Context, uuid string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, accessRulesV6Path+"/uuid/"+url.PathEscape(uuid), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) findAccessRuleV6ByName(ctx context.Context, name string) (*AccessRule, error) {
	req, err := c.newRequest(ctx, http.MethodGet, accessRulesV6Path, nil)
	if err != nil {
		return nil, err
	}
	var body accessRulesV6Body
	if err := c.do(req, &body); err != nil {
		return nil, err
	}
	for _, w := range body.AccessRules {
		if w.IPv6 != nil && w.IPv6.Name == name {
			return w.IPv6, nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Method: http.MethodGet, Path: accessRulesV6Path, Messages: []string{fmt.Sprintf("ipv6 access rule named %q not found after creation", name)}}
}

// ----------------------------------------------------------------------------
// NAT policies (IPv4)
// ----------------------------------------------------------------------------

const natPoliciesPath = "/nat-policies/ipv4"

// CreateNATPolicy stages a new NAT policy and returns the assigned UUID,
// matched back by policy name.
func (c *Client) CreateNATPolicy(ctx context.Context, policy NATPolicy) (string, error) {
	body := natPoliciesBody{NATPolicies: []natPolicyWrapper{{IPv4: &policy}}}
	req, err := c.newRequest(ctx, http.MethodPost, natPoliciesPath, body)
	if err != nil {
		return "", err
	}
	if err := c.do(req, nil); err != nil {
		return "", err
	}
	created, err := c.findNATPolicyByName(ctx, policy.Name)
	if err != nil {
		return "", err
	}
	return created.UUID, nil
}

// UpdateNATPolicy stages a change to an existing NAT policy by UUID.
func (c *Client) UpdateNATPolicy(ctx context.Context, uuid string, policy NATPolicy) error {
	body := natPoliciesBody{NATPolicies: []natPolicyWrapper{{IPv4: &policy}}}
	req, err := c.newRequest(ctx, http.MethodPut, natPoliciesPath+"/uuid/"+url.PathEscape(uuid), body)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// GetNATPolicy fetches a single NAT policy by UUID.
func (c *Client) GetNATPolicy(ctx context.Context, uuid string) (*NATPolicy, error) {
	path := natPoliciesPath + "/uuid/" + url.PathEscape(uuid)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var body natPoliciesBody
	if err := c.do(req, &body); err != nil {
		return nil, err
	}
	for _, w := range body.NATPolicies {
		if w.IPv4 != nil && w.IPv4.UUID == uuid {
			return w.IPv4, nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Method: http.MethodGet, Path: path, Messages: []string{fmt.Sprintf("nat policy %q not found", uuid)}}
}

// DeleteNATPolicy stages deletion of a NAT policy by UUID.
func (c *Client) DeleteNATPolicy(ctx context.Context, uuid string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, natPoliciesPath+"/uuid/"+url.PathEscape(uuid), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) findNATPolicyByName(ctx context.Context, name string) (*NATPolicy, error) {
	req, err := c.newRequest(ctx, http.MethodGet, natPoliciesPath, nil)
	if err != nil {
		return nil, err
	}
	var body natPoliciesBody
	if err := c.do(req, &body); err != nil {
		return nil, err
	}
	for _, w := range body.NATPolicies {
		if w.IPv4 != nil && w.IPv4.Name == name {
			return w.IPv4, nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Method: http.MethodGet, Path: natPoliciesPath, Messages: []string{fmt.Sprintf("nat policy named %q not found after creation", name)}}
}

// ----------------------------------------------------------------------------
// Interfaces (IPv4)
// ----------------------------------------------------------------------------

const interfacesPath = "/interfaces/ipv4"

// ConfigureInterface stages configuration of an existing interface. Interfaces
// are physical ports, so this is modeled as a PUT to the named selector rather
// than a create.
func (c *Client) ConfigureInterface(ctx context.Context, iface Interface) error {
	body := interfacesBody{Interfaces: []interfaceWrapper{{IPv4: &iface}}}
	req, err := c.newRequest(ctx, http.MethodPut, interfacesPath+"/name/"+url.PathEscape(iface.Name), body)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// GetInterface fetches a single interface by name.
func (c *Client) GetInterface(ctx context.Context, name string) (*Interface, error) {
	path := interfacesPath + "/name/" + url.PathEscape(name)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var body interfacesBody
	if err := c.do(req, &body); err != nil {
		return nil, err
	}
	for _, w := range body.Interfaces {
		if w.IPv4 != nil && w.IPv4.Name == name {
			return w.IPv4, nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Method: http.MethodGet, Path: path, Messages: []string{fmt.Sprintf("interface %q not found", name)}}
}
