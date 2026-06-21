package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// This file holds the per-object CRUD methods. Each maps directly onto the
// SonicOS HTTP verbs: POST to create, PUT to a /name/<name> selector to update,
// GET to read, and DELETE to remove. None of these commit; the caller activates
// the pending configuration via Client.CommitPending.

// ----------------------------------------------------------------------------
// Address objects (IPv4)
// ----------------------------------------------------------------------------

const addressObjectsPath = "/address-objects/ipv4"

// CreateAddressObjectIPv4 stages a new IPv4 address object.
func (c *Client) CreateAddressObjectIPv4(ctx context.Context, obj AddressObjectIPv4) error {
	body := addressObjectsBody{AddressObjects: []addressObjectWrapper{{IPv4: &obj}}}
	req, err := c.newRequest(ctx, http.MethodPost, addressObjectsPath, body)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// UpdateAddressObjectIPv4 stages a change to an existing IPv4 address object.
func (c *Client) UpdateAddressObjectIPv4(ctx context.Context, name string, obj AddressObjectIPv4) error {
	body := addressObjectsBody{AddressObjects: []addressObjectWrapper{{IPv4: &obj}}}
	req, err := c.newRequest(ctx, http.MethodPut, addressObjectsPath+"/name/"+url.PathEscape(name), body)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// GetAddressObjectIPv4 fetches a single IPv4 address object by name. It returns
// an *APIError with a 404 status (see IsNotFound) when the object is absent.
func (c *Client) GetAddressObjectIPv4(ctx context.Context, name string) (*AddressObjectIPv4, error) {
	req, err := c.newRequest(ctx, http.MethodGet, addressObjectsPath+"/name/"+url.PathEscape(name), nil)
	if err != nil {
		return nil, err
	}
	var body addressObjectsBody
	if err := c.do(req, &body); err != nil {
		return nil, err
	}
	for _, w := range body.AddressObjects {
		if w.IPv4 != nil && w.IPv4.Name == name {
			return w.IPv4, nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Method: http.MethodGet, Path: addressObjectsPath, Messages: []string{fmt.Sprintf("address object %q not found", name)}}
}

// DeleteAddressObjectIPv4 stages deletion of an IPv4 address object.
func (c *Client) DeleteAddressObjectIPv4(ctx context.Context, name string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, addressObjectsPath+"/name/"+url.PathEscape(name), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// ----------------------------------------------------------------------------
// Service objects
// ----------------------------------------------------------------------------

const serviceObjectsPath = "/service-objects"

// CreateServiceObject stages a new service object.
func (c *Client) CreateServiceObject(ctx context.Context, obj ServiceObject) error {
	body := serviceObjectsBody{ServiceObjects: []ServiceObject{obj}}
	req, err := c.newRequest(ctx, http.MethodPost, serviceObjectsPath, body)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// UpdateServiceObject stages a change to an existing service object.
func (c *Client) UpdateServiceObject(ctx context.Context, name string, obj ServiceObject) error {
	body := serviceObjectsBody{ServiceObjects: []ServiceObject{obj}}
	req, err := c.newRequest(ctx, http.MethodPut, serviceObjectsPath+"/name/"+url.PathEscape(name), body)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// GetServiceObject fetches a single service object by name.
func (c *Client) GetServiceObject(ctx context.Context, name string) (*ServiceObject, error) {
	req, err := c.newRequest(ctx, http.MethodGet, serviceObjectsPath+"/name/"+url.PathEscape(name), nil)
	if err != nil {
		return nil, err
	}
	var body serviceObjectsBody
	if err := c.do(req, &body); err != nil {
		return nil, err
	}
	for i := range body.ServiceObjects {
		if body.ServiceObjects[i].Name == name {
			return &body.ServiceObjects[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Method: http.MethodGet, Path: serviceObjectsPath, Messages: []string{fmt.Sprintf("service object %q not found", name)}}
}

// DeleteServiceObject stages deletion of a service object.
func (c *Client) DeleteServiceObject(ctx context.Context, name string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, serviceObjectsPath+"/name/"+url.PathEscape(name), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// ----------------------------------------------------------------------------
// Zones
// ----------------------------------------------------------------------------

const zonesPath = "/zones"

// CreateZone stages a new security zone.
func (c *Client) CreateZone(ctx context.Context, obj Zone) error {
	body := zonesBody{Zones: []Zone{obj}}
	req, err := c.newRequest(ctx, http.MethodPost, zonesPath, body)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// UpdateZone stages a change to an existing zone.
func (c *Client) UpdateZone(ctx context.Context, name string, obj Zone) error {
	body := zonesBody{Zones: []Zone{obj}}
	req, err := c.newRequest(ctx, http.MethodPut, zonesPath+"/name/"+url.PathEscape(name), body)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// GetZone fetches a single zone by name.
func (c *Client) GetZone(ctx context.Context, name string) (*Zone, error) {
	req, err := c.newRequest(ctx, http.MethodGet, zonesPath+"/name/"+url.PathEscape(name), nil)
	if err != nil {
		return nil, err
	}
	var body zonesBody
	if err := c.do(req, &body); err != nil {
		return nil, err
	}
	for i := range body.Zones {
		if body.Zones[i].Name == name {
			return &body.Zones[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Method: http.MethodGet, Path: zonesPath, Messages: []string{fmt.Sprintf("zone %q not found", name)}}
}

// DeleteZone stages deletion of a zone.
func (c *Client) DeleteZone(ctx context.Context, name string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, zonesPath+"/name/"+url.PathEscape(name), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// ----------------------------------------------------------------------------
// Access rules (IPv4)
//
// Unlike the object types above, access rules are identified by a UUID that the
// appliance assigns on creation, so CreateAccessRule reads the rule back to
// recover it.
// ----------------------------------------------------------------------------

const accessRulesPath = "/access-rules/ipv4"

// CreateAccessRule stages a new IPv4 access rule and returns the UUID the
// appliance assigned to it, matched back by rule name.
func (c *Client) CreateAccessRule(ctx context.Context, rule AccessRule) (string, error) {
	body := accessRulesBody{AccessRules: []accessRuleWrapper{{IPv4: &rule}}}
	req, err := c.newRequest(ctx, http.MethodPost, accessRulesPath, body)
	if err != nil {
		return "", err
	}
	if err := c.do(req, nil); err != nil {
		return "", err
	}
	created, err := c.findAccessRuleByName(ctx, rule.Name)
	if err != nil {
		return "", err
	}
	return created.UUID, nil
}

// UpdateAccessRule stages a change to an existing access rule, addressed by
// UUID.
func (c *Client) UpdateAccessRule(ctx context.Context, uuid string, rule AccessRule) error {
	body := accessRulesBody{AccessRules: []accessRuleWrapper{{IPv4: &rule}}}
	req, err := c.newRequest(ctx, http.MethodPut, accessRulesPath+"/uuid/"+url.PathEscape(uuid), body)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// GetAccessRule fetches a single access rule by UUID.
func (c *Client) GetAccessRule(ctx context.Context, uuid string) (*AccessRule, error) {
	req, err := c.newRequest(ctx, http.MethodGet, accessRulesPath+"/uuid/"+url.PathEscape(uuid), nil)
	if err != nil {
		return nil, err
	}
	var body accessRulesBody
	if err := c.do(req, &body); err != nil {
		return nil, err
	}
	for _, w := range body.AccessRules {
		if w.IPv4 != nil && w.IPv4.UUID == uuid {
			return w.IPv4, nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Method: http.MethodGet, Path: accessRulesPath, Messages: []string{fmt.Sprintf("access rule %q not found", uuid)}}
}

// DeleteAccessRule stages deletion of an access rule by UUID.
func (c *Client) DeleteAccessRule(ctx context.Context, uuid string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, accessRulesPath+"/uuid/"+url.PathEscape(uuid), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// findAccessRuleByName scans the full rule set for a rule with the given name.
// Used only to recover the appliance-assigned UUID immediately after creation.
func (c *Client) findAccessRuleByName(ctx context.Context, name string) (*AccessRule, error) {
	req, err := c.newRequest(ctx, http.MethodGet, accessRulesPath, nil)
	if err != nil {
		return nil, err
	}
	var body accessRulesBody
	if err := c.do(req, &body); err != nil {
		return nil, err
	}
	for _, w := range body.AccessRules {
		if w.IPv4 != nil && w.IPv4.Name == name {
			return w.IPv4, nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Method: http.MethodGet, Path: accessRulesPath, Messages: []string{fmt.Sprintf("access rule named %q not found after creation", name)}}
}
