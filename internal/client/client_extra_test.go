package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateNATPolicyRecoversUUID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/sonicos/nat-policies/ipv4":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/api/sonicos/nat-policies/ipv4":
			_ = json.NewEncoder(w).Encode(natPoliciesBody{NATPolicies: []natPolicyWrapper{
				{IPv4: &NATPolicy{UUID: "nat-1", Name: "masq-lan"}},
			}})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	uuid, err := newTestClient(t, srv).CreateNATPolicy(context.Background(), NATPolicy{Name: "masq-lan"})
	if err != nil {
		t.Fatalf("CreateNATPolicy: %v", err)
	}
	if uuid != "nat-1" {
		t.Fatalf("expected uuid nat-1, got %q", uuid)
	}
}

func TestConfigureInterfacePayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/sonicos/interfaces/ipv4/name/X2" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body interfacesBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(body.Interfaces) != 1 || body.Interfaces[0].IPv4 == nil {
			t.Fatalf("unexpected wrapper: %+v", body)
		}
		got := body.Interfaces[0].IPv4
		if got.Zone != "DMZ" || got.Static == nil || got.Static.IP != "10.1.1.1" {
			t.Errorf("unexpected interface: %+v", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := newTestClient(t, srv).ConfigureInterface(context.Background(), Interface{
		Name: "X2", Zone: "DMZ", Static: &InterfaceStatic{IP: "10.1.1.1", Netmask: "255.255.255.0"},
	})
	if err != nil {
		t.Fatalf("ConfigureInterface: %v", err)
	}
}

func TestGetAddressObjectIPv6FoundAndNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sonicos/address-objects/ipv6/name/v6net" {
			_ = json.NewEncoder(w).Encode(addressObjectsV6Body{AddressObjects: []addressObjectV6Wrapper{
				{IPv6: &AddressObjectIPv6{Name: "v6net", Zone: "LAN", Network: &AddressObjectV6Network{Subnet: "2001:db8::", Prefix: 64}}},
			}})
			return
		}
		_ = json.NewEncoder(w).Encode(addressObjectsV6Body{})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	obj, err := c.GetAddressObjectIPv6(context.Background(), "v6net")
	if err != nil {
		t.Fatalf("GetAddressObjectIPv6: %v", err)
	}
	if obj.Network == nil || obj.Network.Prefix != 64 {
		t.Errorf("unexpected object: %+v", obj)
	}

	if _, err := c.GetAddressObjectIPv6(context.Background(), "missing"); !IsNotFound(err) {
		t.Fatalf("expected IsNotFound, got %v", err)
	}
}

func TestCreateAccessRuleIPv6RecoversUUID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/sonicos/access-rules/ipv6":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/api/sonicos/access-rules/ipv6":
			_ = json.NewEncoder(w).Encode(accessRulesV6Body{AccessRules: []accessRuleV6Wrapper{
				{IPv6: &AccessRule{UUID: "v6-1", Name: "allow-v6", From: "LAN", To: "WAN", Action: "allow"}},
			}})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	uuid, err := newTestClient(t, srv).CreateAccessRuleIPv6(context.Background(), AccessRule{Name: "allow-v6", From: "LAN", To: "WAN", Action: "allow"})
	if err != nil {
		t.Fatalf("CreateAccessRuleIPv6: %v", err)
	}
	if uuid != "v6-1" {
		t.Fatalf("expected uuid v6-1, got %q", uuid)
	}
}
