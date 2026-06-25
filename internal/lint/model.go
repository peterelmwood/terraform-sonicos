// Package lint provides a static analyzer ("linter") for SonicOS configuration
// expressed as Terraform resources. It reads the JSON emitted by
// `terraform show -json` (of either a plan file or current state), reconstructs
// the set of SonicOS objects and rules, and applies a set of rules that the
// firewall must or should obey — referential integrity, zone consistency,
// duplicate detection, rule shadowing, and overly-permissive rules.
//
// The goal is to explain, before or alongside `terraform apply`, why a
// configuration may be rejected by the appliance or may not behave as intended —
// findings the per-resource Terraform validators cannot produce because they
// require a whole-configuration view.
package lint

// Severity classifies a finding.
type Severity string

const (
	// SeverityError marks a configuration that the appliance will reject or that
	// is almost certainly a mistake (e.g. a rule referencing a missing object).
	SeverityError Severity = "error"
	// SeverityWarning marks a configuration that is valid but likely unintended
	// or risky (e.g. an any→any allow rule).
	SeverityWarning Severity = "warning"
)

// Finding is a single issue reported by a rule.
type Finding struct {
	Severity Severity `json:"severity"`
	// Rule is the stable identifier of the rule that produced the finding, e.g.
	// "referential.missing_address_object".
	Rule string `json:"rule"`
	// Address is the Terraform resource address the finding concerns, e.g.
	// "sonicos_access_rule.allow_web". It may be empty for config-wide findings.
	Address string `json:"address,omitempty"`
	// Message is a human-readable explanation.
	Message string `json:"message"`
}

// AddressObject is a normalized address object (IPv4 or IPv6).
type AddressObject struct {
	Address string // Terraform resource address
	Name    string
	Zone    string
	Type    string // host | network | range | fqdn
	IPv6    bool
	// Value fields, populated per Type.
	Host          string
	NetworkSubnet string
	NetworkMask   string // IPv4 only
	NetworkPrefix int64  // IPv6 only
	RangeBegin    string
	RangeEnd      string
}

// ServiceObject is a normalized service object.
type ServiceObject struct {
	Address   string
	Name      string
	Protocol  string
	PortBegin int64
	PortEnd   int64
	HasPort   bool
}

// Zone is a normalized security zone.
type Zone struct {
	Address      string
	Name         string
	SecurityType string
}

// AccessRule is a normalized access rule (IPv4 or IPv6).
type AccessRule struct {
	Address     string
	Name        string
	IPv6        bool
	From        string
	To          string
	Action      string
	Enable      bool
	SourceName  string // "" means any
	DestName    string // "" means any
	ServiceName string // "" means any
}

// NATPolicy is a normalized NAT policy.
type NATPolicy struct {
	Address               string
	Name                  string
	OriginalSource        string
	TranslatedSource      string
	OriginalDestination   string
	TranslatedDestination string
	OriginalService       string
	TranslatedService     string
	InboundInterface      string
	OutboundInterface     string
}

// Interface is a normalized interface.
type Interface struct {
	Address string
	Name    string
	Zone    string
}

// Config is the whole-configuration view the rules operate on.
type Config struct {
	AddressObjects []AddressObject
	ServiceObjects []ServiceObject
	Zones          []Zone
	AccessRules    []AccessRule
	NATPolicies    []NATPolicy
	Interfaces     []Interface
}

// addressObjectNames returns the set of defined address object names.
func (c *Config) addressObjectNames() map[string]AddressObject {
	m := make(map[string]AddressObject, len(c.AddressObjects))
	for _, o := range c.AddressObjects {
		m[o.Name] = o
	}
	return m
}

func (c *Config) serviceObjectNames() map[string]struct{} {
	m := make(map[string]struct{}, len(c.ServiceObjects))
	for _, o := range c.ServiceObjects {
		m[o.Name] = struct{}{}
	}
	return m
}

func (c *Config) zoneNames() map[string]struct{} {
	m := make(map[string]struct{}, len(c.Zones))
	for _, z := range c.Zones {
		m[z.Name] = struct{}{}
	}
	return m
}

func (c *Config) interfaceNames() map[string]struct{} {
	m := make(map[string]struct{}, len(c.Interfaces))
	for _, i := range c.Interfaces {
		m[i.Name] = struct{}{}
	}
	return m
}
