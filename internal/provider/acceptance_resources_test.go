package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAddressObject_host(t *testing.T) {
	zone := accZone()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "sonicos_address_object" "test" {
  name = "tf-acc-host"
  zone = %[1]q
  type = "host"
  host = "192.0.2.10"
}
`, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sonicos_address_object.test", "host", "192.0.2.10"),
					resource.TestCheckResourceAttr("sonicos_address_object.test", "id", "tf-acc-host"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "sonicos_address_object" "test" {
  name = "tf-acc-host"
  zone = %[1]q
  type = "host"
  host = "192.0.2.20"
}
`, zone),
				Check: resource.TestCheckResourceAttr("sonicos_address_object.test", "host", "192.0.2.20"),
			},
			{
				ResourceName:      "sonicos_address_object.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccServiceObject_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "sonicos_service_object" "test" {
  name       = "tf-acc-svc"
  protocol   = "tcp"
  port_begin = 8443
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sonicos_service_object.test", "port_begin", "8443"),
					resource.TestCheckResourceAttr("sonicos_service_object.test", "port_end", "8443"),
				),
			},
			{
				ResourceName:      "sonicos_service_object.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccZone_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "sonicos_zone" "test" {
  name          = "TF-ACC-ZONE"
  security_type = "public"
}
`,
				Check: resource.TestCheckResourceAttr("sonicos_zone.test", "security_type", "public"),
			},
			{
				ResourceName:      "sonicos_zone.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAccessRule_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "sonicos_access_rule" "test" {
  name   = "tf-acc-rule"
  from   = "LAN"
  to     = "WAN"
  action = "allow"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sonicos_access_rule.test", "action", "allow"),
					resource.TestCheckResourceAttrSet("sonicos_access_rule.test", "id"),
				),
			},
		},
	})
}

func TestAccAddressObjectIPv6_network(t *testing.T) {
	zone := accZone()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "sonicos_address_object_ipv6" "test" {
  name           = "tf-acc-v6net"
  zone           = %[1]q
  type           = "network"
  network_subnet = "2001:db8::"
  network_prefix = 64
}
`, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sonicos_address_object_ipv6.test", "network_subnet", "2001:db8::"),
					resource.TestCheckResourceAttr("sonicos_address_object_ipv6.test", "network_prefix", "64"),
				),
			},
			{
				ResourceName:      "sonicos_address_object_ipv6.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAccessRuleIPv6_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "sonicos_access_rule_ipv6" "test" {
  name   = "tf-acc-rule-v6"
  from   = "LAN"
  to     = "WAN"
  action = "allow"
}
`,
				Check: resource.TestCheckResourceAttrSet("sonicos_access_rule_ipv6.test", "id"),
			},
		},
	})
}

func TestAccNATPolicy_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "sonicos_nat_policy" "test" {
  name              = "tf-acc-nat"
  original_source   = ""
  outbound_interface = "X1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sonicos_nat_policy.test", "outbound_interface", "X1"),
					resource.TestCheckResourceAttrSet("sonicos_nat_policy.test", "id"),
				),
			},
		},
	})
}

func TestAccInterface_static(t *testing.T) {
	iface := os.Getenv("SONICOS_ACC_INTERFACE")
	if iface == "" {
		t.Skip("set SONICOS_ACC_INTERFACE (e.g. X2) to run the interface acceptance test")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "sonicos_interface" "test" {
  name          = %[1]q
  zone          = "DMZ"
  ip_assignment = "static"
  ip            = "10.99.99.1"
  netmask       = "255.255.255.0"
}
`, iface),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sonicos_interface.test", "ip", "10.99.99.1"),
					resource.TestCheckResourceAttr("sonicos_interface.test", "ip_assignment", "static"),
				),
			},
		},
	})
}
