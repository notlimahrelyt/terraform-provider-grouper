//go:build tools

// Package tools pins developer tooling as module dependencies so that
// `go generate` and CI use consistent versions. It is never compiled into the
// provider binary (guarded by the "tools" build tag).
package tools

import (
	// Documentation generation for the Terraform Registry.
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
