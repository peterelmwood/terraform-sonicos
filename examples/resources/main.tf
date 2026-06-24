# Example resource definitions for the SonicOS provider.

# A host address object.
resource "sonicos_address_object" "web_server" {
  name = "web-server-01"
  zone = "LAN"
  type = "host"
  host = "192.0.2.10"
}

# A network (subnet) address object.
resource "sonicos_address_object" "internal_net" {
  name           = "internal-net"
  zone           = "LAN"
  type           = "network"
  network_subnet = "10.0.0.0"
  network_mask   = "255.255.255.0"
}

# An address range.
resource "sonicos_address_object" "dhcp_pool" {
  name        = "dhcp-pool"
  zone        = "LAN"
  type        = "range"
  range_begin = "10.0.0.100"
  range_end   = "10.0.0.200"
}

# An FQDN address object.
resource "sonicos_address_object" "update_server" {
  name        = "vendor-updates"
  zone        = "WAN"
  type        = "fqdn"
  fqdn_domain = "updates.example.com"
}

# A custom TCP service object (single port).
resource "sonicos_service_object" "app_https" {
  name       = "app-https"
  protocol   = "tcp"
  port_begin = 8443
}

# A UDP service object spanning a port range.
resource "sonicos_service_object" "voip" {
  name       = "voip-rtp"
  protocol   = "udp"
  port_begin = 16384
  port_end   = 32767
}

# A security zone.
resource "sonicos_zone" "dmz" {
  name          = "DMZ-APP"
  security_type = "public"
}

# An access rule allowing the web server out to the WAN over the custom service.
resource "sonicos_access_rule" "allow_web_out" {
  name             = "allow-web-out"
  from             = "LAN"
  to               = "WAN"
  action           = "allow"
  source_name      = sonicos_address_object.web_server.name
  destination_name = "" # any
  service_name     = sonicos_service_object.app_https.name
  comment          = "Managed by Terraform"
}

# An IPv6 network address object (prefix length instead of a dotted mask).
resource "sonicos_address_object_ipv6" "v6_lan" {
  name           = "v6-lan-net"
  zone           = "LAN"
  type           = "network"
  network_subnet = "2001:db8:0:1::"
  network_prefix = 64
}

# An IPv6 access rule (same shape as the IPv4 rule).
resource "sonicos_access_rule_ipv6" "allow_v6_out" {
  name        = "allow-v6-out"
  from        = "LAN"
  to          = "WAN"
  action      = "allow"
  source_name = sonicos_address_object_ipv6.v6_lan.name
  comment     = "Managed by Terraform"
}

# A NAT policy translating the internal network out a WAN interface (PAT/masquerade).
# Empty original_* slots mean "any"; empty translated_* slots mean "original"
# (no translation on that field).
resource "sonicos_nat_policy" "masq_lan" {
  name               = "masq-lan-out"
  original_source    = sonicos_address_object.internal_net.name
  translated_source  = "" # original — interface address used for outbound PAT
  outbound_interface = "X1"
  comment            = "Managed by Terraform"
}

# Configure an existing interface. This does not create a port; it configures
# the already-present interface named below.
resource "sonicos_interface" "dmz" {
  name             = "X2"
  zone             = "DMZ"
  ip_assignment    = "static"
  ip               = "10.1.1.1"
  netmask          = "255.255.255.0"
  management_https = true
  management_ping  = true
}

# Look up an existing address object created out-of-band.
data "sonicos_address_object" "existing_gateway" {
  name = "Default Gateway"
}

output "gateway_ip" {
  value = data.sonicos_address_object.existing_gateway.host
}
