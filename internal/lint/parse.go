package lint

import (
	"encoding/json"
	"fmt"
)

// The JSON shapes below are the subset of `terraform show -json` output we need.
// That command emits the same envelope for both plan files and state: a state
// representation under "values", and (for plans) the projected result under
// "planned_values". Each carries a module tree whose resources have a type,
// address, and attribute "values" map. We deliberately model only what we use
// rather than depend on the full terraform-json schema.

type tfResource struct {
	Address string         `json:"address"`
	Type    string         `json:"type"`
	Name    string         `json:"name"`
	Values  map[string]any `json:"values"`
}

type tfModule struct {
	Resources    []tfResource `json:"resources"`
	ChildModules []tfModule   `json:"child_modules"`
}

type tfStateValues struct {
	RootModule tfModule `json:"root_module"`
}

type tfShow struct {
	// Values is present for `terraform show -json` of state.
	Values *tfStateValues `json:"values"`
	// PlannedValues is present for `terraform show -json <planfile>`.
	PlannedValues *tfStateValues `json:"planned_values"`
}

// Parse reads `terraform show -json` output (of a plan or state) and builds the
// normalized Config of SonicOS resources. Non-SonicOS resources are ignored.
func Parse(data []byte) (*Config, error) {
	var show tfShow
	if err := json.Unmarshal(data, &show); err != nil {
		return nil, fmt.Errorf("parsing terraform JSON: %w", err)
	}

	root := show.PlannedValues
	if root == nil {
		root = show.Values
	}
	if root == nil {
		// Not a recognized show payload, or an empty plan/state.
		return &Config{}, nil
	}

	cfg := &Config{}
	var walk func(m tfModule)
	walk = func(m tfModule) {
		for _, r := range m.Resources {
			cfg.add(r)
		}
		for _, child := range m.ChildModules {
			walk(child)
		}
	}
	walk(root.RootModule)
	return cfg, nil
}

// add appends a single resource to the Config if it is a SonicOS type we model.
func (c *Config) add(r tfResource) {
	switch r.Type {
	case "sonicos_address_object":
		c.AddressObjects = append(c.AddressObjects, addressObjectFrom(r, false))
	case "sonicos_address_object_ipv6":
		c.AddressObjects = append(c.AddressObjects, addressObjectFrom(r, true))
	case "sonicos_service_object":
		c.ServiceObjects = append(c.ServiceObjects, ServiceObject{
			Address:   r.Address,
			Name:      str(r.Values, "name"),
			Protocol:  str(r.Values, "protocol"),
			PortBegin: integer(r.Values, "port_begin"),
			PortEnd:   integer(r.Values, "port_end"),
			HasPort:   r.Values["port_begin"] != nil,
		})
	case "sonicos_zone":
		c.Zones = append(c.Zones, Zone{
			Address:      r.Address,
			Name:         str(r.Values, "name"),
			SecurityType: str(r.Values, "security_type"),
		})
	case "sonicos_access_rule":
		c.AccessRules = append(c.AccessRules, accessRuleFrom(r, false))
	case "sonicos_access_rule_ipv6":
		c.AccessRules = append(c.AccessRules, accessRuleFrom(r, true))
	case "sonicos_nat_policy":
		c.NATPolicies = append(c.NATPolicies, NATPolicy{
			Address:               r.Address,
			Name:                  str(r.Values, "name"),
			OriginalSource:        str(r.Values, "original_source"),
			TranslatedSource:      str(r.Values, "translated_source"),
			OriginalDestination:   str(r.Values, "original_destination"),
			TranslatedDestination: str(r.Values, "translated_destination"),
			OriginalService:       str(r.Values, "original_service"),
			TranslatedService:     str(r.Values, "translated_service"),
			InboundInterface:      str(r.Values, "inbound_interface"),
			OutboundInterface:     str(r.Values, "outbound_interface"),
		})
	case "sonicos_interface":
		c.Interfaces = append(c.Interfaces, Interface{
			Address: r.Address,
			Name:    str(r.Values, "name"),
			Zone:    str(r.Values, "zone"),
		})
	}
}

func addressObjectFrom(r tfResource, ipv6 bool) AddressObject {
	o := AddressObject{
		Address:       r.Address,
		Name:          str(r.Values, "name"),
		Zone:          str(r.Values, "zone"),
		Type:          str(r.Values, "type"),
		IPv6:          ipv6,
		Host:          str(r.Values, "host"),
		NetworkSubnet: str(r.Values, "network_subnet"),
		RangeBegin:    str(r.Values, "range_begin"),
		RangeEnd:      str(r.Values, "range_end"),
	}
	if ipv6 {
		o.NetworkPrefix = integer(r.Values, "network_prefix")
	} else {
		o.NetworkMask = str(r.Values, "network_mask")
	}
	return o
}

func accessRuleFrom(r tfResource, ipv6 bool) AccessRule {
	return AccessRule{
		Address:     r.Address,
		Name:        str(r.Values, "name"),
		IPv6:        ipv6,
		From:        str(r.Values, "from"),
		To:          str(r.Values, "to"),
		Action:      str(r.Values, "action"),
		Enable:      boolean(r.Values, "enable", true),
		SourceName:  str(r.Values, "source_name"),
		DestName:    str(r.Values, "destination_name"),
		ServiceName: str(r.Values, "service_name"),
	}
}

// str reads a string attribute, returning "" when absent or null (e.g. an
// unknown value in a plan).
func str(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// integer reads a numeric attribute (JSON numbers decode to float64).
func integer(m map[string]any, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	}
	return 0
}

// boolean reads a bool attribute, falling back to def when absent or null.
func boolean(m map[string]any, key string, def bool) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return def
}
