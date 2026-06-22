package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestServiceObjectToAPI(t *testing.T) {
	tests := []struct {
		name      string
		model     serviceObjectModel
		wantError bool
		wantBegin int64
		wantEnd   int64
		wantPort  bool
	}{
		{
			name:      "single tcp port defaults end to begin",
			model:     serviceObjectModel{Name: types.StringValue("a"), Protocol: types.StringValue("tcp"), PortBegin: types.Int64Value(443), PortEnd: types.Int64Null()},
			wantBegin: 443, wantEnd: 443, wantPort: true,
		},
		{
			name:      "explicit udp range",
			model:     serviceObjectModel{Name: types.StringValue("a"), Protocol: types.StringValue("udp"), PortBegin: types.Int64Value(16384), PortEnd: types.Int64Value(32767)},
			wantBegin: 16384, wantEnd: 32767, wantPort: true,
		},
		{
			name:      "tcp missing port is an error",
			model:     serviceObjectModel{Name: types.StringValue("a"), Protocol: types.StringValue("tcp"), PortBegin: types.Int64Null(), PortEnd: types.Int64Null()},
			wantError: true,
		},
		{
			name:      "end before begin is an error",
			model:     serviceObjectModel{Name: types.StringValue("a"), Protocol: types.StringValue("tcp"), PortBegin: types.Int64Value(443), PortEnd: types.Int64Value(80)},
			wantError: true,
		},
		{
			name:     "icmp ignores ports",
			model:    serviceObjectModel{Name: types.StringValue("a"), Protocol: types.StringValue("icmp"), PortBegin: types.Int64Null(), PortEnd: types.Int64Null()},
			wantPort: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj, diags := tt.model.toAPI()
			if tt.wantError {
				if !diags.HasError() {
					t.Fatalf("expected error diagnostic, got none")
				}
				return
			}
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if tt.wantPort {
				if obj.Port == nil {
					t.Fatalf("expected port, got nil")
				}
				if obj.Port.Begin != tt.wantBegin || obj.Port.End != tt.wantEnd {
					t.Errorf("got port %d-%d, want %d-%d", obj.Port.Begin, obj.Port.End, tt.wantBegin, tt.wantEnd)
				}
			} else if obj.Port != nil {
				t.Errorf("expected no port, got %+v", obj.Port)
			}
		})
	}
}

func TestAddressObjectToAPIBlankValuesRejected(t *testing.T) {
	tests := []struct {
		name      string
		model     addressObjectModel
		wantError bool
	}{
		{
			name:  "valid host",
			model: addressObjectModel{Name: types.StringValue("h"), Zone: types.StringValue("LAN"), Type: types.StringValue("host"), Host: types.StringValue("192.0.2.1")},
		},
		{
			name:      "empty host string rejected",
			model:     addressObjectModel{Name: types.StringValue("h"), Zone: types.StringValue("LAN"), Type: types.StringValue("host"), Host: types.StringValue("")},
			wantError: true,
		},
		{
			name:      "empty network_subnet rejected",
			model:     addressObjectModel{Name: types.StringValue("n"), Zone: types.StringValue("LAN"), Type: types.StringValue("network"), NetworkSubnet: types.StringValue(""), NetworkMask: types.StringValue("255.255.255.0")},
			wantError: true,
		},
		{
			name:      "empty range_end rejected",
			model:     addressObjectModel{Name: types.StringValue("r"), Zone: types.StringValue("LAN"), Type: types.StringValue("range"), RangeBegin: types.StringValue("10.0.0.1"), RangeEnd: types.StringValue("")},
			wantError: true,
		},
		{
			name:  "valid network",
			model: addressObjectModel{Name: types.StringValue("n"), Zone: types.StringValue("LAN"), Type: types.StringValue("network"), NetworkSubnet: types.StringValue("10.0.0.0"), NetworkMask: types.StringValue("255.255.255.0")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, diags := tt.model.toAPI()
			if tt.wantError && !diags.HasError() {
				t.Fatalf("expected error diagnostic, got none")
			}
			if !tt.wantError && diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
		})
	}
}
