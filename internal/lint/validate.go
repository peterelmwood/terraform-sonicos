package lint

import (
	"net/netip"
	"strings"
)

// builtinZones are the zones SonicOS ships with. Referential checks treat these
// as always-present so a rule referencing, say, WAN is not flagged just because
// no sonicos_zone resource defines it. "any" is included for rules that span all
// zones.
var builtinZones = map[string]struct{}{
	"LAN": {}, "WAN": {}, "DMZ": {}, "WLAN": {}, "VPN": {},
	"SSLVPN": {}, "MULTICAST": {}, "ANY": {},
}

func isBuiltinZone(name string) bool {
	_, ok := builtinZones[strings.ToUpper(name)]
	return ok
}

// untrustedZones are zones generally considered the untrusted side of the
// firewall, used by the overly-permissive rules.
var untrustedZones = map[string]struct{}{
	"WAN": {},
}

func isUntrustedZone(name string) bool {
	_, ok := untrustedZones[strings.ToUpper(name)]
	return ok
}

// IsIPv4 reports whether s parses as an IPv4 address. Exported so the provider's
// in-plan validators can share the same format check as the linter.
func IsIPv4(s string) bool {
	addr, err := netip.ParseAddr(s)
	return err == nil && addr.Is4()
}

// IsIPv6 reports whether s parses as an IPv6 address.
func IsIPv6(s string) bool {
	addr, err := netip.ParseAddr(s)
	return err == nil && addr.Is6() && !addr.Is4In6()
}

// IsIPv4Mask reports whether s is a valid contiguous IPv4 subnet mask.
func IsIPv4Mask(s string) bool {
	return isIPv4Mask(s)
}

// AddrLessOrEqual reports whether a <= b as IP addresses of the same family.
func AddrLessOrEqual(a, b string) bool {
	return addrLessOrEqual(a, b)
}

// isIPv4 reports whether s parses as an IPv4 address.
func isIPv4(s string) bool { return IsIPv4(s) }

// isIPv6 reports whether s parses as an IPv6 address.
func isIPv6(s string) bool { return IsIPv6(s) }

// isIPv4Mask reports whether s is a valid contiguous IPv4 subnet mask, e.g.
// 255.255.255.0.
func isIPv4Mask(s string) bool {
	addr, err := netip.ParseAddr(s)
	if err != nil || !addr.Is4() {
		return false
	}
	b := addr.As4()
	mask := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	if mask == 0 {
		return true
	}
	// A valid mask is a run of 1s followed by a run of 0s: inverting and adding
	// one yields a power of two.
	inv := ^mask + 1
	return inv&(inv-1) == 0
}

// addrLessOrEqual reports whether a <= b, both parsed as IP addresses of the
// same family. It returns false if either fails to parse.
func addrLessOrEqual(a, b string) bool {
	aa, err1 := netip.ParseAddr(a)
	bb, err2 := netip.ParseAddr(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return aa.Compare(bb) <= 0
}
