package client

// This file extends the API models with the object types added alongside the
// initial set: NAT policies, interfaces, and the IPv6 variants of address
// objects and access rules. As with models.go, the shapes mirror the JSON the
// appliance exposes in its built-in Swagger document and should be checked
// against your firmware version before production use.

// ----------------------------------------------------------------------------
// Address objects (IPv6)
// Endpoint: /api/sonicos/address-objects/ipv6
//
// IPv6 networks are expressed with a prefix length rather than a dotted mask;
// otherwise the type discriminator works exactly like the IPv4 variant.
// ----------------------------------------------------------------------------

// AddressObjectV6Network is the value for an IPv6 subnet address object.
type AddressObjectV6Network struct {
	Subnet string `json:"subnet"`
	Prefix int64  `json:"prefix_size"`
}

// AddressObjectIPv6 is a single IPv6 address object. Exactly one of Host,
// Network, or Range is populated.
type AddressObjectIPv6 struct {
	Name    string                  `json:"name"`
	Zone    string                  `json:"zone,omitempty"`
	Host    *AddressObjectHost      `json:"host,omitempty"`
	Network *AddressObjectV6Network `json:"network,omitempty"`
	Range   *AddressObjectRange     `json:"range,omitempty"`
}

type addressObjectV6Wrapper struct {
	IPv6 *AddressObjectIPv6 `json:"ipv6,omitempty"`
}

type addressObjectsV6Body struct {
	AddressObjects []addressObjectV6Wrapper `json:"address_objects"`
}

// ----------------------------------------------------------------------------
// Access rules (IPv6)
// Endpoint: /api/sonicos/access-rules/ipv6
//
// The rule body is identical to the IPv4 variant; only the wrapper key differs.
// ----------------------------------------------------------------------------

type accessRuleV6Wrapper struct {
	IPv6 *AccessRule `json:"ipv6,omitempty"`
}

type accessRulesV6Body struct {
	AccessRules []accessRuleV6Wrapper `json:"access_rules"`
}

// ----------------------------------------------------------------------------
// NAT policies (IPv4)
// Endpoint: /api/sonicos/nat-policies/ipv4
//
// NAT policies are UUID-identified like access rules. Each of the six
// object slots references an object by name, or uses the built-in "any"
// (original side) / "original" (translated side, meaning "no translation").
// ----------------------------------------------------------------------------

// NATRef references an object by name, or selects a built-in. Exactly one field
// is set: Name for a concrete object, Any for the original side's wildcard, or
// Original for a translated side that performs no translation.
type NATRef struct {
	Name     string `json:"name,omitempty"`
	Any      bool   `json:"any,omitempty"`
	Original bool   `json:"original,omitempty"`
}

// NATPolicy is a single IPv4 NAT policy.
type NATPolicy struct {
	UUID                  string  `json:"uuid,omitempty"`
	Name                  string  `json:"name,omitempty"`
	Enable                bool    `json:"enable"`
	Comment               string  `json:"comment,omitempty"`
	OriginalSource        *NATRef `json:"original_source,omitempty"`
	TranslatedSource      *NATRef `json:"translated_source,omitempty"`
	OriginalDestination   *NATRef `json:"original_destination,omitempty"`
	TranslatedDestination *NATRef `json:"translated_destination,omitempty"`
	OriginalService       *NATRef `json:"original_service,omitempty"`
	TranslatedService     *NATRef `json:"translated_service,omitempty"`
	Inbound               *NATRef `json:"inbound,omitempty"`
	Outbound              *NATRef `json:"outbound,omitempty"`
}

type natPolicyWrapper struct {
	IPv4 *NATPolicy `json:"ipv4,omitempty"`
}

type natPoliciesBody struct {
	NATPolicies []natPolicyWrapper `json:"nat_policies"`
}

// ----------------------------------------------------------------------------
// Interfaces (IPv4)
// Endpoint: /api/sonicos/interfaces/ipv4
//
// Interfaces are physical (or virtual) ports addressed by name, e.g. "X2".
// They are not created or destroyed like objects; the resource configures an
// existing interface and Delete returns it to an unassigned state.
// ----------------------------------------------------------------------------

// InterfaceStatic holds the addressing for a statically configured interface.
type InterfaceStatic struct {
	IP      string `json:"ip"`
	Netmask string `json:"netmask"`
	Gateway string `json:"gateway,omitempty"`
}

// InterfaceManagement controls which management services are reachable on the
// interface.
type InterfaceManagement struct {
	HTTPS bool `json:"https"`
	Ping  bool `json:"ping"`
	SSH   bool `json:"ssh"`
	SNMP  bool `json:"snmp"`
}

// Interface is a single IPv4 interface configuration. Exactly one of Static or
// DHCP describes the IP assignment.
type Interface struct {
	Name       string               `json:"name"`
	Zone       string               `json:"zone,omitempty"`
	Comment    string               `json:"comment,omitempty"`
	MTU        int64                `json:"mtu,omitempty"`
	Static     *InterfaceStatic     `json:"static,omitempty"`
	DHCP       bool                 `json:"dhcp,omitempty"`
	Management *InterfaceManagement `json:"management,omitempty"`
}

type interfaceWrapper struct {
	IPv4 *Interface `json:"ipv4,omitempty"`
}

type interfacesBody struct {
	Interfaces []interfaceWrapper `json:"interfaces"`
}
