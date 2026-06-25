package lint

import (
	"strings"
	"testing"
)

// findingsByRule indexes findings by their rule identifier for assertions.
func findingsByRule(findings []Finding) map[string][]Finding {
	m := map[string][]Finding{}
	for _, f := range findings {
		m[f.Rule] = append(m[f.Rule], f)
	}
	return m
}

func TestReferentialMissingObjects(t *testing.T) {
	cfg := &Config{
		AccessRules: []AccessRule{{
			Address: "sonicos_access_rule.r", Name: "r", From: "LAN", To: "WAN", Action: "allow", Enable: true,
			SourceName: "does-not-exist", ServiceName: "ghost-svc",
		}},
	}
	byRule := findingsByRule(Lint(cfg))
	if len(byRule["referential.missing_address_object"]) != 1 {
		t.Errorf("expected missing address object finding, got %v", byRule)
	}
	if len(byRule["referential.missing_service_object"]) != 1 {
		t.Errorf("expected missing service object finding, got %v", byRule)
	}
}

func TestReferentialBuiltinZonesNotFlagged(t *testing.T) {
	cfg := &Config{
		AccessRules: []AccessRule{{
			Address: "sonicos_access_rule.r", Name: "r", From: "LAN", To: "WAN", Action: "deny", Enable: true,
		}},
	}
	for _, f := range Lint(cfg) {
		if f.Rule == "referential.unknown_zone" {
			t.Errorf("built-in zones should not be flagged: %+v", f)
		}
	}
}

func TestUnknownZoneWarning(t *testing.T) {
	cfg := &Config{
		AccessRules: []AccessRule{{
			Address: "sonicos_access_rule.r", Name: "r", From: "CUSTOMZONE", To: "WAN", Action: "deny", Enable: true,
		}},
	}
	if len(findingsByRule(Lint(cfg))["referential.unknown_zone"]) != 1 {
		t.Errorf("expected unknown_zone warning")
	}
}

func TestDuplicateNames(t *testing.T) {
	cfg := &Config{
		AddressObjects: []AddressObject{
			{Address: "sonicos_address_object.a", Name: "dup", Zone: "LAN", Type: "host", Host: "192.0.2.1"},
			{Address: "sonicos_address_object.b", Name: "dup", Zone: "LAN", Type: "host", Host: "192.0.2.2"},
		},
	}
	f := findingsByRule(Lint(cfg))["duplicate.address_object_name"]
	if len(f) != 1 || f[0].Severity != SeverityError {
		t.Errorf("expected one duplicate error, got %v", f)
	}
}

func TestFieldFormat(t *testing.T) {
	cfg := &Config{
		AddressObjects: []AddressObject{
			{Address: "sonicos_address_object.bad_host", Name: "bad", Zone: "LAN", Type: "host", Host: "not-an-ip"},
			{Address: "sonicos_address_object.bad_mask", Name: "net", Zone: "LAN", Type: "network", NetworkSubnet: "10.0.0.0", NetworkMask: "255.0.255.0"},
			{Address: "sonicos_address_object.bad_range", Name: "rng", Zone: "LAN", Type: "range", RangeBegin: "10.0.0.9", RangeEnd: "10.0.0.1"},
			{Address: "sonicos_address_object.v6", Name: "v6", Zone: "LAN", Type: "host", IPv6: true, Host: "2001:db8::1"},
		},
		ServiceObjects: []ServiceObject{
			{Address: "sonicos_service_object.s", Name: "s", Protocol: "tcp", PortBegin: 70000, PortEnd: 70000, HasPort: true},
		},
	}
	byRule := findingsByRule(Lint(cfg))
	if len(byRule["format.invalid_ip"]) != 1 {
		t.Errorf("expected one invalid_ip (host), got %v", byRule["format.invalid_ip"])
	}
	if len(byRule["format.invalid_mask"]) != 1 {
		t.Errorf("expected one invalid_mask, got %v", byRule["format.invalid_mask"])
	}
	if len(byRule["format.range_order"]) != 1 {
		t.Errorf("expected one range_order, got %v", byRule["format.range_order"])
	}
	if len(byRule["format.port_range"]) != 1 {
		t.Errorf("expected one port_range, got %v", byRule["format.port_range"])
	}
}

func TestShadowedRules(t *testing.T) {
	cfg := &Config{
		AccessRules: []AccessRule{
			{Address: "sonicos_access_rule.broad", Name: "broad", From: "LAN", To: "WAN", Action: "allow", Enable: true},
			{Address: "sonicos_access_rule.narrow", Name: "narrow", From: "LAN", To: "WAN", Action: "allow", Enable: true, ServiceName: "https"},
		},
		ServiceObjects: []ServiceObject{{Address: "sonicos_service_object.https", Name: "https", Protocol: "tcp", PortBegin: 443, PortEnd: 443, HasPort: true}},
	}
	f := findingsByRule(Lint(cfg))["shadow.shadowed_rule"]
	if len(f) != 1 || !strings.Contains(f[0].Message, "narrow") {
		t.Errorf("expected narrow rule to be shadowed, got %v", f)
	}
}

func TestShadowedRespectsZonePairAndDisabled(t *testing.T) {
	cfg := &Config{
		AccessRules: []AccessRule{
			{Name: "a", From: "LAN", To: "WAN", Action: "allow", Enable: false}, // disabled, can't shadow
			{Name: "b", From: "LAN", To: "WAN", Action: "allow", Enable: true, ServiceName: "https"},
			{Name: "c", From: "DMZ", To: "WAN", Action: "allow", Enable: true}, // different zone pair
		},
		ServiceObjects: []ServiceObject{{Name: "https", Protocol: "tcp", PortBegin: 443, PortEnd: 443, HasPort: true}},
	}
	if f := findingsByRule(Lint(cfg))["shadow.shadowed_rule"]; len(f) != 0 {
		t.Errorf("expected no shadow findings, got %v", f)
	}
}

func TestOverlyPermissive(t *testing.T) {
	cfg := &Config{
		AccessRules: []AccessRule{
			{Address: "sonicos_access_rule.anyany", Name: "anyany", From: "LAN", To: "WAN", Action: "allow", Enable: true},
			{Address: "sonicos_access_rule.wanin", Name: "wanin", From: "WAN", To: "LAN", Action: "allow", Enable: true, DestName: "web"},
		},
		AddressObjects: []AddressObject{{Name: "web", Zone: "LAN", Type: "host", Host: "192.0.2.5"}},
	}
	byRule := findingsByRule(Lint(cfg))
	if len(byRule["permissive.any_any_allow"]) != 1 {
		t.Errorf("expected any_any_allow, got %v", byRule["permissive.any_any_allow"])
	}
	if len(byRule["permissive.untrusted_inbound_any_service"]) != 1 {
		t.Errorf("expected untrusted_inbound_any_service, got %v", byRule["permissive.untrusted_inbound_any_service"])
	}
}

func TestZoneMismatch(t *testing.T) {
	cfg := &Config{
		AddressObjects: []AddressObject{{Name: "dmz-host", Zone: "DMZ", Type: "host", Host: "10.1.1.5"}},
		AccessRules: []AccessRule{
			{Address: "sonicos_access_rule.r", Name: "r", From: "LAN", To: "WAN", Action: "allow", Enable: true, SourceName: "dmz-host"},
		},
	}
	if len(findingsByRule(Lint(cfg))["zone.source_zone_mismatch"]) != 1 {
		t.Errorf("expected source_zone_mismatch")
	}
}

func TestCleanConfigNoFindings(t *testing.T) {
	cfg := &Config{
		Zones:          []Zone{{Name: "DMZ", SecurityType: "public"}},
		AddressObjects: []AddressObject{{Name: "web", Zone: "DMZ", Type: "host", Host: "10.1.1.5"}},
		ServiceObjects: []ServiceObject{{Name: "https", Protocol: "tcp", PortBegin: 443, PortEnd: 443, HasPort: true}},
		AccessRules: []AccessRule{
			{Name: "allow-web", From: "WAN", To: "DMZ", Action: "allow", Enable: true, DestName: "web", ServiceName: "https"},
		},
	}
	if f := Lint(cfg); len(f) != 0 {
		t.Errorf("expected clean config, got %v", f)
	}
}
