package lint

import "fmt"

// rule is a single check over the whole configuration.
type rule func(c *Config) []Finding

// allRules is the ordered set of checks Lint runs. Grouped by the four
// categories: referential/zone consistency, field/format sanity, duplicate
// detection, and overly-permissive rules.
var allRules = []rule{
	ruleFieldFormat,
	ruleReferentialIntegrity,
	ruleZoneConsistency,
	ruleDuplicateNames,
	ruleShadowedRules,
	ruleOverlyPermissive,
}

// ---------------------------------------------------------------------------
// Field & format sanity
// ---------------------------------------------------------------------------

func ruleFieldFormat(c *Config) []Finding {
	var out []Finding
	for _, o := range c.AddressObjects {
		out = append(out, addressObjectFormat(o)...)
	}
	for _, s := range c.ServiceObjects {
		if !s.HasPort {
			continue
		}
		if s.PortBegin < 1 || s.PortBegin > 65535 || s.PortEnd < 1 || s.PortEnd > 65535 {
			out = append(out, Finding{SeverityError, "format.port_range", s.Address,
				fmt.Sprintf("service %q has a port outside 1-65535 (%d-%d)", s.Name, s.PortBegin, s.PortEnd)})
		} else if s.PortEnd < s.PortBegin {
			out = append(out, Finding{SeverityError, "format.port_order", s.Address,
				fmt.Sprintf("service %q has port_end (%d) less than port_begin (%d)", s.Name, s.PortEnd, s.PortBegin)})
		}
	}
	return out
}

func addressObjectFormat(o AddressObject) []Finding {
	var out []Finding
	add := func(r, msg string) { out = append(out, Finding{SeverityError, r, o.Address, msg}) }
	validHost := isIPv4
	validNet := isIPv4
	family := "IPv4"
	if o.IPv6 {
		validHost = isIPv6
		validNet = isIPv6
		family = "IPv6"
	}

	switch o.Type {
	case "host":
		if o.Host != "" && !validHost(o.Host) {
			add("format.invalid_ip", fmt.Sprintf("address object %q host %q is not a valid %s address", o.Name, o.Host, family))
		}
	case "network":
		if o.NetworkSubnet != "" && !validNet(o.NetworkSubnet) {
			add("format.invalid_ip", fmt.Sprintf("address object %q network_subnet %q is not a valid %s address", o.Name, o.NetworkSubnet, family))
		}
		if !o.IPv6 && o.NetworkMask != "" && !isIPv4Mask(o.NetworkMask) {
			add("format.invalid_mask", fmt.Sprintf("address object %q network_mask %q is not a valid subnet mask", o.Name, o.NetworkMask))
		}
		if o.IPv6 && (o.NetworkPrefix < 0 || o.NetworkPrefix > 128) {
			add("format.invalid_prefix", fmt.Sprintf("address object %q network_prefix %d is outside 0-128", o.Name, o.NetworkPrefix))
		}
	case "range":
		okBegin := o.RangeBegin == "" || validHost(o.RangeBegin)
		okEnd := o.RangeEnd == "" || validHost(o.RangeEnd)
		if !okBegin {
			add("format.invalid_ip", fmt.Sprintf("address object %q range_begin %q is not a valid %s address", o.Name, o.RangeBegin, family))
		}
		if !okEnd {
			add("format.invalid_ip", fmt.Sprintf("address object %q range_end %q is not a valid %s address", o.Name, o.RangeEnd, family))
		}
		if okBegin && okEnd && o.RangeBegin != "" && o.RangeEnd != "" && !addrLessOrEqual(o.RangeBegin, o.RangeEnd) {
			add("format.range_order", fmt.Sprintf("address object %q range_begin %q is greater than range_end %q", o.Name, o.RangeBegin, o.RangeEnd))
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Referential integrity
// ---------------------------------------------------------------------------

func ruleReferentialIntegrity(c *Config) []Finding {
	var out []Finding
	addrs := c.addressObjectNames()
	svcs := c.serviceObjectNames()
	zones := c.zoneNames()
	ifaces := c.interfaceNames()

	checkAddr := func(addr, role, name, refName string) {
		if refName == "" {
			return
		}
		if _, ok := addrs[refName]; !ok {
			out = append(out, Finding{SeverityError, "referential.missing_address_object", addr,
				fmt.Sprintf("%s references address object %q, which is not defined in this configuration", role, refName)})
		}
	}
	checkZone := func(addr, role, zone string) {
		if zone == "" || isBuiltinZone(zone) {
			return
		}
		if _, ok := zones[zone]; !ok {
			out = append(out, Finding{SeverityWarning, "referential.unknown_zone", addr,
				fmt.Sprintf("%s zone %q is neither a built-in zone nor defined in this configuration (it may exist on the appliance already)", role, zone)})
		}
	}

	for _, r := range c.AccessRules {
		checkAddr(r.Address, "source_name", r.Name, r.SourceName)
		checkAddr(r.Address, "destination_name", r.Name, r.DestName)
		if r.ServiceName != "" {
			if _, ok := svcs[r.ServiceName]; !ok {
				out = append(out, Finding{SeverityError, "referential.missing_service_object", r.Address,
					fmt.Sprintf("service_name references service object %q, which is not defined in this configuration", r.ServiceName)})
			}
		}
		checkZone(r.Address, "from", r.From)
		checkZone(r.Address, "to", r.To)
	}

	for _, n := range c.NATPolicies {
		checkAddr(n.Address, "original_source", n.Name, n.OriginalSource)
		checkAddr(n.Address, "translated_source", n.Name, n.TranslatedSource)
		checkAddr(n.Address, "original_destination", n.Name, n.OriginalDestination)
		checkAddr(n.Address, "translated_destination", n.Name, n.TranslatedDestination)
		for _, ref := range []struct{ role, name string }{
			{"original_service", n.OriginalService}, {"translated_service", n.TranslatedService},
		} {
			if ref.name == "" {
				continue
			}
			if _, ok := svcs[ref.name]; !ok {
				out = append(out, Finding{SeverityError, "referential.missing_service_object", n.Address,
					fmt.Sprintf("%s references service object %q, which is not defined in this configuration", ref.role, ref.name)})
			}
		}
		for _, ref := range []struct{ role, name string }{
			{"inbound_interface", n.InboundInterface}, {"outbound_interface", n.OutboundInterface},
		} {
			if ref.name == "" {
				continue
			}
			if _, ok := ifaces[ref.name]; !ok {
				out = append(out, Finding{SeverityWarning, "referential.unknown_interface", n.Address,
					fmt.Sprintf("%s references interface %q, which is not managed in this configuration (it may exist on the appliance already)", ref.role, ref.name)})
			}
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Zone consistency
// ---------------------------------------------------------------------------

func ruleZoneConsistency(c *Config) []Finding {
	var out []Finding
	addrs := c.addressObjectNames()
	for _, r := range c.AccessRules {
		if o, ok := addrs[r.SourceName]; ok && o.Zone != "" && r.From != "" && o.Zone != r.From {
			out = append(out, Finding{SeverityWarning, "zone.source_zone_mismatch", r.Address,
				fmt.Sprintf("source address object %q is in zone %q but the rule's from zone is %q", r.SourceName, o.Zone, r.From)})
		}
		if o, ok := addrs[r.DestName]; ok && o.Zone != "" && r.To != "" && o.Zone != r.To {
			out = append(out, Finding{SeverityWarning, "zone.destination_zone_mismatch", r.Address,
				fmt.Sprintf("destination address object %q is in zone %q but the rule's to zone is %q", r.DestName, o.Zone, r.To)})
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Duplicate names
// ---------------------------------------------------------------------------

func ruleDuplicateNames(c *Config) []Finding {
	var out []Finding
	dup := func(kind, rule string, items map[string][]string) {
		for name, addrs := range items {
			if len(addrs) > 1 {
				out = append(out, Finding{SeverityError, rule, addrs[0],
					fmt.Sprintf("%s name %q is declared by %d resources (%v); names must be unique on the appliance", kind, name, len(addrs), addrs)})
			}
		}
	}

	addrNames := map[string][]string{}
	for _, o := range c.AddressObjects {
		addrNames[o.Name] = append(addrNames[o.Name], o.Address)
	}
	dup("address object", "duplicate.address_object_name", addrNames)

	svcNames := map[string][]string{}
	for _, s := range c.ServiceObjects {
		svcNames[s.Name] = append(svcNames[s.Name], s.Address)
	}
	dup("service object", "duplicate.service_object_name", svcNames)

	zoneNames := map[string][]string{}
	for _, z := range c.Zones {
		zoneNames[z.Name] = append(zoneNames[z.Name], z.Address)
	}
	dup("zone", "duplicate.zone_name", zoneNames)

	ruleNames := map[string][]string{}
	for _, r := range c.AccessRules {
		ruleNames[r.Name] = append(ruleNames[r.Name], r.Address)
	}
	dup("access rule", "duplicate.rule_name", ruleNames)

	natNames := map[string][]string{}
	for _, n := range c.NATPolicies {
		natNames[n.Name] = append(natNames[n.Name], n.Address)
	}
	dup("nat policy", "duplicate.nat_policy_name", natNames)

	return out
}

// ---------------------------------------------------------------------------
// Shadowed rules
// ---------------------------------------------------------------------------

// covers reports whether rule a, appearing earlier, makes rule b unreachable:
// same zone pair and family, a enabled, and a's match criteria are at least as
// broad as b's on every axis (empty means "any").
func covers(a, b AccessRule) bool {
	if a.IPv6 != b.IPv6 || a.From != b.From || a.To != b.To || !a.Enable {
		return false
	}
	broader := func(aVal, bVal string) bool { return aVal == "" || aVal == bVal }
	return broader(a.SourceName, b.SourceName) &&
		broader(a.DestName, b.DestName) &&
		broader(a.ServiceName, b.ServiceName)
}

func ruleShadowedRules(c *Config) []Finding {
	var out []Finding
	for i := range c.AccessRules {
		later := c.AccessRules[i]
		for j := 0; j < i; j++ {
			earlier := c.AccessRules[j]
			if !covers(earlier, later) {
				continue
			}
			detail := fmt.Sprintf("rule %q is shadowed by earlier rule %q (%s), which matches the same or broader traffic and will handle it first",
				later.Name, earlier.Name, earlier.Action)
			if earlier.Action != later.Action {
				detail += fmt.Sprintf("; note the earlier rule's action (%s) differs from this rule's (%s)", earlier.Action, later.Action)
			}
			out = append(out, Finding{SeverityWarning, "shadow.shadowed_rule", later.Address, detail})
			break // one shadowing rule is enough to report
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Overly-permissive rules
// ---------------------------------------------------------------------------

func ruleOverlyPermissive(c *Config) []Finding {
	var out []Finding
	for _, r := range c.AccessRules {
		if r.Action != "allow" || !r.Enable {
			continue
		}
		if r.SourceName == "" && r.DestName == "" && r.ServiceName == "" {
			out = append(out, Finding{SeverityWarning, "permissive.any_any_allow", r.Address,
				fmt.Sprintf("rule %q allows any source to any destination on any service (%s → %s); consider scoping it", r.Name, r.From, r.To)})
			continue
		}
		if isUntrustedZone(r.From) && r.ServiceName == "" {
			out = append(out, Finding{SeverityWarning, "permissive.untrusted_inbound_any_service", r.Address,
				fmt.Sprintf("rule %q allows inbound traffic from untrusted zone %q on any service; restrict it to specific services", r.Name, r.From)})
		}
	}
	return out
}
