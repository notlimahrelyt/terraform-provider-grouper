---
page_title: "Development - Grouper Provider"
subcategory: ""
description: |-
  Building, testing, and generating docs for the Grouper provider.
---

# Development

Requires Go (see `go.mod`). Optional tools: `golangci-lint`, `goreleaser`, and
`tfplugindocs` (pinned via `tools.go`).

## Common tasks

```sh
make fmt      # gofmt -s -w .
make vet      # go vet ./...
make lint     # golangci-lint run
make test     # unit tests (no network)
make build    # compile the provider
make install  # build + install into ~/.terraform.d/plugins for local testing
make generate # regenerate registry docs (tfplugindocs)
```

## Local testing with dev overrides

Point Terraform at your locally built provider with a CLI config:

```hcl
# ~/.terraformrc
provider_installation {
  dev_overrides {
    "notlimahrelyt/grouper" = "/Users/you/go/bin" # dir containing the built binary
  }
  direct {}
}
```

Then `make install` (or `go build`) and run Terraform normally; skip
`terraform init` when using dev overrides.

## Regenerating documentation

`docs/` is generated from schema descriptions, the `examples/` directory, and
guide templates in `templates/guides/`:

```sh
make generate
```

Edit attribute `Description`s, the `examples/**/*.tf` / `import.sh` files, and the
guide sources under `templates/guides/`, then regenerate. **Do not edit
`docs/guides/` directly** — `tfplugindocs` overwrites it from `templates/guides/`
on every run.

## Layered design

- `internal/client` — HTTP/API concerns only; no Terraform dependency. API DTOs
  (`ws*`) are kept separate from exported types so Grouper API changes stay
  contained here.
- `internal/provider` — Terraform schema, resources, data sources; depends on the
  client. State models are separate from client DTOs.

Confirm any new Grouper endpoint against the API and record it in
[`docs/api-observations.md`](../api-observations.md) before implementing it.
