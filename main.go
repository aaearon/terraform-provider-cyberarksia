// Package main is the entry point for the CyberArk SIA Terraform provider
package main

import (
	"context"
	"flag"
	"log"

	"github.com/aaearon/terraform-provider-cyberarksia/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// version, commit, and date are set via ldflags during build
var (
	version = "dev"
	commit  = "" //nolint:unused // Set by ldflags during release builds
	date    = "" //nolint:unused // Set by ldflags during release builds
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/aaearon/cyberarksia",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
