// terraform-provider-sonicos is a Terraform provider for managing SonicOS 7+
// firewall configuration through the appliance's built-in REST API.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/peterelmwood/terraform-sonicos/internal/provider"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		// Registry address used in required_providers; this is the source the
		// provider is published and referenced under.
		Address: "registry.terraform.io/peterelmwood/sonicos",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err.Error())
	}
}
