// terraform-provider-grouper is a Terraform provider for Internet2 Grouper.
//
// It manages Grouper folders/stems, groups, and direct memberships through the
// Grouper Web Services REST API using the Terraform Plugin Framework.
package main

// Regenerate registry documentation (docs/) from schema descriptions, examples/,
// and templates/ with `make generate`.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name grouper

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/notlimahrelyt/terraform-provider-grouper/internal/provider"
)

// These values are overwritten at release time by the linker via ldflags.
var (
	// version is the provider version, set by GoReleaser.
	version = "dev"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		// The registry address that Terraform configurations reference in
		// required_providers. Adjust if the provider is published elsewhere.
		Address: "registry.terraform.io/notlimahrelyt/grouper",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
