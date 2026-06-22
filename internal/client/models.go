package client

// This file models the subset of the SonicOS 7+ REST API that the provider
// manages. The shapes mirror the JSON the appliance returns from its built-in
// Swagger/OpenAPI document (Home | API on the device). SonicOS wraps every
// object collection in a named array, e.g. {"address_objects": [ {"ipv4": {...}} ]},
// and selects single objects with a /name/<name> path segment.

// ----------------------------------------------------------------------------
// Address objects (IPv4)
// Endpoint: /api/sonicos/address-objects/ipv4
// ----------------------------------------------------------------------------

// AddressObjectHost is the value for a single-host address object.
type AddressObjectHost struct {
	IP string `json:"ip"`
}

// AddressObjectNetwork is the value for a subnet address object.
type AddressObjectNetwork struct {
	Subnet string `json:"subnet"`
	Mask   string `json:"mask"`
}

// AddressObjectRange is the value for an address range object.
type AddressObjectRange struct {
	Begin string `json:"begin"`
	End   string `json:"end"`
}

// AddressObjectFQDN is the value for an FQDN address object.
type AddressObjectFQDN struct {
	Domain string `json:"domain"`
}

// AddressObjectIPv4 is a single IPv4 address object. Exactly one of Host,
// Network, Range, or FQDN is populated, which the API uses as the type
// discriminator.
type AddressObjectIPv4 struct {
	Name    string                `json:"name"`
	Zone    string                `json:"zone,omitempty"`
	Host    *AddressObjectHost    `json:"host,omitempty"`
	Network *AddressObjectNetwork `json:"network,omitempty"`
	Range   *AddressObjectRange   `json:"range,omitempty"`
	FQDN    *AddressObjectFQDN    `json:"fqdn,omitempty"`
}

type addressObjectWrapper struct {
	IPv4 *AddressObjectIPv4 `json:"ipv4,omitempty"`
}

type addressObjectsBody struct {
	AddressObjects []addressObjectWrapper `json:"address_objects"`
}

// ----------------------------------------------------------------------------
// Service objects
// Endpoint: /api/sonicos/service-objects
// ----------------------------------------------------------------------------

// ServicePort is the port range for a TCP/UDP service object. For a single
// port, Begin and End are equal.
type ServicePort struct {
	Begin int64 `json:"begin"`
	End   int64 `json:"end"`
}

// ServiceObject is a single custom service object.
type ServiceObject struct {
	Name     string       `json:"name"`
	Protocol string       `json:"protocol"`
	Port     *ServicePort `json:"port,omitempty"`
}

type serviceObjectsBody struct {
	ServiceObjects []ServiceObject `json:"service_objects"`
}

// ----------------------------------------------------------------------------
// Zones
// Endpoint: /api/sonicos/zones
// ----------------------------------------------------------------------------

// Zone is a security zone.
type Zone struct {
	Name           string `json:"name"`
	SecurityType   string `json:"security_type,omitempty"`
	InterfaceTrust bool   `json:"interface_trust"`
	AutoGenerate   bool   `json:"auto_generate_access_rules,omitempty"`
}

type zonesBody struct {
	Zones []Zone `json:"zones"`
}

// ----------------------------------------------------------------------------
// Access rules (IPv4)
// Endpoint: /api/sonicos/access-rules/ipv4
// ----------------------------------------------------------------------------

// AccessRuleName references an address object or service object by name, or the
// built-in "any". Exactly one field is set.
type AccessRuleName struct {
	Name string `json:"name,omitempty"`
	Any  bool   `json:"any,omitempty"`
}

// AccessRule is a single IPv4 access (firewall) rule. SonicOS assigns each rule
// a stable UUID on creation which the provider uses as the resource ID.
type AccessRule struct {
	UUID        string          `json:"uuid,omitempty"`
	Name        string          `json:"name,omitempty"`
	From        string          `json:"from"`
	To          string          `json:"to"`
	Action      string          `json:"action"`
	Enable      bool            `json:"enable"`
	Source      *AccessRuleName `json:"source,omitempty"`
	Destination *AccessRuleName `json:"destination,omitempty"`
	Service     *AccessRuleName `json:"service,omitempty"`
	Comment     string          `json:"comment,omitempty"`
}

type accessRuleWrapper struct {
	IPv4 *AccessRule `json:"ipv4,omitempty"`
}

type accessRulesBody struct {
	AccessRules []accessRuleWrapper `json:"access_rules"`
}

// statusResponse is the standard SonicOS API status envelope returned on
// errors and on some successful mutating calls.
type statusResponse struct {
	Status struct {
		Success bool `json:"success"`
		Info    []struct {
			Level   string `json:"level"`
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"info"`
	} `json:"status"`
}
