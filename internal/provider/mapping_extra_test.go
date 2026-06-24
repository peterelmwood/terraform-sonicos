package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNATPolicyRefSemantics(t *testing.T) {
	// Empty original slot -> any; empty translated slot -> original (no
	// translation); named slots carry the name through.
	m := natPolicyModel{
		Name:                  types.StringValue("p"),
		Enable:                types.BoolValue(true),
		Comment:               types.StringValue(""),
		OriginalSource:        types.StringValue("internal-net"),
		TranslatedSource:      types.StringValue(""),
		OriginalDestination:   types.StringValue(""),
		TranslatedDestination: types.StringValue(""),
		OriginalService:       types.StringValue(""),
		TranslatedService:     types.StringValue("https"),
		InboundInterface:      types.StringValue(""),
		OutboundInterface:     types.StringValue("X1"),
	}
	p := m.toAPI()

	if p.OriginalSource.Name != "internal-net" || p.OriginalSource.Any {
		t.Errorf("original_source: want name, got %+v", p.OriginalSource)
	}
	if !p.TranslatedSource.Original || p.TranslatedSource.Name != "" {
		t.Errorf("translated_source: want original, got %+v", p.TranslatedSource)
	}
	if !p.OriginalDestination.Any {
		t.Errorf("original_destination: want any, got %+v", p.OriginalDestination)
	}
	if p.TranslatedService.Name != "https" {
		t.Errorf("translated_service: want https, got %+v", p.TranslatedService)
	}
	if !p.Inbound.Any {
		t.Errorf("inbound: want any, got %+v", p.Inbound)
	}
	if p.Outbound.Name != "X1" {
		t.Errorf("outbound: want X1, got %+v", p.Outbound)
	}

	// Round-trip back to the model: built-ins collapse to empty strings.
	back := fromAPINATPolicy(&p)
	if back.OriginalSource.ValueString() != "internal-net" {
		t.Errorf("round-trip original_source: %q", back.OriginalSource.ValueString())
	}
	if back.TranslatedSource.ValueString() != "" {
		t.Errorf("round-trip translated_source should be empty, got %q", back.TranslatedSource.ValueString())
	}
	if back.OutboundInterface.ValueString() != "X1" {
		t.Errorf("round-trip outbound: %q", back.OutboundInterface.ValueString())
	}
}

func TestInterfaceToAPI(t *testing.T) {
	t.Run("static requires ip and netmask", func(t *testing.T) {
		m := interfaceModel{
			Name: types.StringValue("X2"), Zone: types.StringValue("DMZ"),
			IPAssignment: types.StringValue("static"),
			IP:           types.StringValue(""), Netmask: types.StringValue("255.255.255.0"),
			Gateway: types.StringValue(""), MTU: types.Int64Value(1500),
			MgmtHTTPS: types.BoolValue(false), MgmtPing: types.BoolValue(false),
			MgmtSSH: types.BoolValue(false), MgmtSNMP: types.BoolValue(false),
		}
		if _, diags := m.toAPI(); !diags.HasError() {
			t.Fatal("expected error for missing ip")
		}
	})

	t.Run("valid static", func(t *testing.T) {
		m := interfaceModel{
			Name: types.StringValue("X2"), Zone: types.StringValue("DMZ"),
			IPAssignment: types.StringValue("static"),
			IP:           types.StringValue("10.1.1.1"), Netmask: types.StringValue("255.255.255.0"),
			Gateway: types.StringValue(""), MTU: types.Int64Value(1500),
			MgmtHTTPS: types.BoolValue(true), MgmtPing: types.BoolValue(false),
			MgmtSSH: types.BoolValue(false), MgmtSNMP: types.BoolValue(false),
		}
		iface, diags := m.toAPI()
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if iface.Static == nil || iface.Static.IP != "10.1.1.1" {
			t.Errorf("unexpected static: %+v", iface.Static)
		}
		if iface.Management == nil || !iface.Management.HTTPS {
			t.Errorf("expected https management enabled")
		}
	})

	t.Run("dhcp", func(t *testing.T) {
		m := interfaceModel{
			Name: types.StringValue("X3"), Zone: types.StringValue("WAN"),
			IPAssignment: types.StringValue("dhcp"),
			IP:           types.StringNull(), Netmask: types.StringNull(),
			Gateway: types.StringValue(""), MTU: types.Int64Value(1500),
			MgmtHTTPS: types.BoolValue(false), MgmtPing: types.BoolValue(false),
			MgmtSSH: types.BoolValue(false), MgmtSNMP: types.BoolValue(false),
		}
		iface, diags := m.toAPI()
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if !iface.DHCP || iface.Static != nil {
			t.Errorf("expected dhcp assignment, got %+v", iface)
		}
	})
}

func TestAddressObjectV6ToAPI(t *testing.T) {
	t.Run("network requires subnet and prefix", func(t *testing.T) {
		m := addressObjectV6Model{
			Name: types.StringValue("n"), Zone: types.StringValue("LAN"),
			Type: types.StringValue("network"), NetworkSubnet: types.StringValue("2001:db8::"),
			NetworkPrefix: types.Int64Null(),
		}
		if _, diags := m.toAPI(); !diags.HasError() {
			t.Fatal("expected error for missing prefix")
		}
	})
	t.Run("valid network", func(t *testing.T) {
		m := addressObjectV6Model{
			Name: types.StringValue("n"), Zone: types.StringValue("LAN"),
			Type: types.StringValue("network"), NetworkSubnet: types.StringValue("2001:db8::"),
			NetworkPrefix: types.Int64Value(64),
		}
		obj, diags := m.toAPI()
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if obj.Network == nil || obj.Network.Prefix != 64 {
			t.Errorf("unexpected network: %+v", obj.Network)
		}
	})
}
