# Contributing

Thanks for helping improve the Grouper provider. Bug reports, compatibility
reports against other Grouper versions, documentation fixes, and code are all
welcome.

## Ground rules

- **Open an issue before large changes.** New resources or behavior changes are
  easier to land when the design is agreed first.
- **Confirm API behavior before coding it.** Every endpoint the provider calls was
  checked against a live Grouper server, and the request and response shapes are
  recorded in [`docs/api-observations.md`](docs/api-observations.md). Do the same
  for anything new, and update that file with sanitized examples.
- **Never commit real data.** No credentials, real subject ids, internal
  hostnames, or captured responses from a production server.
- **Membership changes must stay non-authoritative.** The provider adds and removes
  only the members it manages. Changes that could remove members Terraform does
  not manage will not be accepted.

## Development setup

Requires Go (version in [`go.mod`](go.mod)) and Terraform. Optional:
[`golangci-lint`](https://golangci-lint.run/) and
[`goreleaser`](https://goreleaser.com/).

```shell
make fmt       # gofmt
make vet       # go vet
make lint      # golangci-lint
make test      # unit tests (no network)
make build     # compile the provider
make install   # install into ~/.terraform.d/plugins for local testing
make generate  # regenerate docs/ with tfplugindocs
```

### Trying a local build

Point Terraform at your build with a `dev_overrides` block in `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "notlimahrelyt/grouper" = "/path/to/your/go/bin"
  }
  direct {}
}
```

Run `go install .`, then use Terraform as normal (skip `terraform init`).

### Code layout

| Path                | Contents                                                                 |
| ------------------- | ------------------------------------------------------------------------ |
| `internal/client`   | Grouper Web Services client: HTTP, request/response types, typed errors. No Terraform imports. |
| `internal/provider` | Terraform schemas, resources, data sources, and their state models.    |
| `examples/`         | Example configurations, also embedded in the generated docs.           |
| `templates/`        | Sources for the provider index page and guides.                         |
| `docs/`             | Generated documentation published to the Terraform Registry.            |

Keep Grouper API details in `internal/client`. Resources work with the client's
types, never with raw API payloads, so a change in the Grouper API stays in one
package.

## Documentation

`docs/` is generated; do not edit it by hand. Change the source and run
`make generate`:

| To change                       | Edit                                              |
| ------------------------------- | ------------------------------------------------- |
| Attribute or resource text      | `MarkdownDescription` in `internal/provider/*.go` |
| Examples on a resource page     | `examples/resources/<type>/resource.tf`           |
| Import instructions             | `examples/resources/<type>/import.sh`             |
| The provider index page         | `templates/index.md.tmpl`                         |
| A guide                         | `templates/guides/<name>.md.tmpl`                 |

CI fails if `docs/` is out of date with its sources.

## Tests

### Unit tests

`make test` runs the unit tests. They use a mock HTTP server and never contact
Grouper. Every change to the client or provider should come with unit tests,
including error paths.

### Acceptance tests

Acceptance tests create and destroy **real** Grouper objects. They run only when
`TF_ACC` is set, and only against a folder whose name contains `test`.

```shell
export TF_ACC=1
export GROUPER_BASE_URL="https://grouper.example.edu/grouper-ws/servicesRest"
export GROUPER_USERNAME="terraform-svc"
export GROUPER_PASSWORD="..."
export GROUPER_TEST_STEM="example:apps:myteam:test"   # must exist; name must contain "test"

# Optional: enable the membership and subject tests
export GROUPER_TEST_SUBJECT_ID="abc123"
export GROUPER_TEST_SUBJECT_ID_2="xyz456"             # enables the unrelated-member test
export GROUPER_TEST_SUBJECT_SOURCE_ID="ldap"          # if your ids need a source

make testacc
```

Safety rules:

- Use a dedicated test folder that holds nothing real. The account needs
  `STEM_ADMIN` and `CREATE` on it.
- Every test object gets a unique `tfacc-*` name under the test folder, and is
  destroyed when the test ends, including on failure.
- Never point acceptance tests at a production folder.

Mention in your pull request which Grouper version you ran the acceptance tests
against.

## Pull requests

1. Fork the repository and create a branch.
2. Make the change with tests and documentation.
3. Run `make fmt lint test generate` and commit any regenerated docs.
4. Add a line under `## [Unreleased]` in [`CHANGELOG.md`](CHANGELOG.md).
5. Open the pull request using [Conventional Commits](https://www.conventionalcommits.org/)
   for the title, e.g. `fix(membership): handle subjects with a slash in the id`.

## Releasing (maintainers)

1. Move the `[Unreleased]` entries in `CHANGELOG.md` under a new version heading
   and merge.
2. Tag the merge commit and push the tag:
   ```shell
   git tag -a v0.2.0 -m "v0.2.0"
   git push origin v0.2.0
   ```
3. The release workflow builds, signs, and publishes a GitHub release using that
   version's `CHANGELOG.md` section as the notes. The Terraform Registry picks it
   up automatically.
