package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// Acceptance tests run real plan/apply cycles against a live SonicOS appliance.
// They are gated by the standard TF_ACC environment variable (set by
// terraform-plugin-testing) and additionally require the same connection
// settings the provider reads at runtime:
//
//	TF_ACC=1
//	SONICOS_HOST, SONICOS_USERNAME, SONICOS_PASSWORD
//	SONICOS_INSECURE=true                  # for the default self-signed cert
//	SONICOS_ACC_ZONE   (optional)          # an existing zone for object tests, default "LAN"
//	SONICOS_ACC_INTERFACE (optional)       # an interface to configure, e.g. "X2"
//
// Run them with, for example:
//
//	TF_ACC=1 SONICOS_HOST=https://192.0.2.1 SONICOS_USERNAME=tf \
//	  SONICOS_PASSWORD=... SONICOS_INSECURE=true go test ./internal/provider -run TestAcc -v
//
// Because SonicOS permits only one API session at a time, run acceptance tests
// serially (the default) against a dedicated lab device — never production.

// testAccProtoV6ProviderFactories wires the in-process provider server used by
// every acceptance test under the "sonicos" name.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"sonicos": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck verifies the required connection environment is present before
// an acceptance test runs, failing fast with a clear message otherwise.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	for _, key := range []string{"SONICOS_HOST", "SONICOS_USERNAME", "SONICOS_PASSWORD"} {
		if os.Getenv(key) == "" {
			t.Fatalf("%s must be set for acceptance tests", key)
		}
	}
}

// accZone returns the zone used by object acceptance tests.
func accZone() string {
	if z := os.Getenv("SONICOS_ACC_ZONE"); z != "" {
		return z
	}
	return "LAN"
}
