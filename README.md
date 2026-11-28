# Terraform Provider for Grouper

[![CI](https://github.com/notlimahrelyt/terraform-provider-grouper/actions/workflows/ci.yml/badge.svg)](https://github.com/notlimahrelyt/terraform-provider-grouper/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/notlimahrelyt/terraform-provider-grouper)](https://github.com/notlimahrelyt/terraform-provider-grouper/releases/latest)
[![Terraform Registry](https://img.shields.io/badge/registry-notlimahrelyt%2Fgrouper-844FBA?logo=terraform)](https://registry.terraform.io/providers/notlimahrelyt/grouper/latest)
[![License: MPL-2.0](https://img.shields.io/badge/license-MPL--2.0-blue)](LICENSE)

Manage [Internet2 Grouper](https://www.internet2.edu/products-services/trust-identity/grouper/)
folders, groups, and memberships as code.

```hcl
resource "grouper_stem" "inventory" {
  name         = "example:apps:inventory"
  display_name = "Inventory"
}

resource "grouper_group" "admins" {
  name         = "${grouper_stem.inventory.name}:admins"
  display_name = "Inventory Admins"
}

resource "grouper_membership" "admins" {
  for_each = toset(["abc123", "xyz456"])

  group_name = grouper_group.admins.name
  subject_id = each.value
}
```

## Documentation

Full documentation is on the
[Terraform Registry](https://registry.terraform.io/providers/notlimahrelyt/grouper/latest/docs):

- [Resources](https://registry.terraform.io/providers/notlimahrelyt/grouper/latest/docs/resources/stem):
  `grouper_stem`, `grouper_group`, `grouper_membership`
- [Data sources](https://registry.terraform.io/providers/notlimahrelyt/grouper/latest/docs/data-sources/stem):
  `grouper_stem`, `grouper_group`, `grouper_subject`
- Guides: [authentication](docs/guides/authentication.md),
  [understanding memberships](docs/guides/memberships.md),
  [installation](docs/guides/installation.md),
  [environment variables](docs/guides/environment-variables.md),
  [troubleshooting](docs/guides/troubleshooting.md)
- A [complete example](examples/complete) that builds an application's folder,
  groups, and members

## Getting started

```hcl
terraform {
  required_providers {
    grouper = {
      source  = "notlimahrelyt/grouper"
      version = "~> 0.1"
    }
  }
}

provider "grouper" {
  base_url = "https://grouper.example.edu/grouper-ws/servicesRest"
}
```

```shell
export GROUPER_USERNAME="terraform-svc"
export GROUPER_PASSWORD="..."
terraform init
terraform plan
```

The account needs privileges on the folders Terraform manages; the
[authentication guide](docs/guides/authentication.md) lists them.

## Design principles

- **Memberships are never authoritative.** Each `grouper_membership` manages one
  direct member. Members added by hand, by the Grouper Loader, or by another tool
  are never removed. There is deliberately no "set all members of this group"
  resource.
- **Stable identity.** Folders and groups are tracked by Grouper UUID, so state
  survives display-name changes. Renaming a folder or group replaces it.
- **No surprises on delete.** Folders are never deleted recursively; Grouper's
  refusal to delete a non-empty folder is reported, not worked around.
- **Verified against a real server.** Every API call was confirmed against a live
  Grouper before it was written, and is recorded in
  [`docs/api-observations.md`](docs/api-observations.md).

Out of scope for now: privileges, composite groups, attribute definitions,
Grouper Loader jobs, and provisioning.

## Compatibility

| Provider | Grouper | `ws_version`         | Terraform | Status                          |
| -------- | ------- | -------------------- | --------- | ------------------------------- |
| 0.1.x    | 4.16    | `v2_6_000` (default) | >= 1.5    | Verified with acceptance tests  |

Other Grouper versions that serve the `v2_6_000` API are expected to work but
have not been verified. If you run the acceptance tests against another version,
please [open an issue](https://github.com/notlimahrelyt/terraform-provider-grouper/issues/new/choose)
with the results.

## Security

- Credentials are read from the environment and never written to state or logs.
- Terraform state contains group names and subject identifiers; keep it in an
  encrypted, access-controlled backend.
- TLS certificates are verified unless you explicitly opt out.

Report vulnerabilities privately as described in [SECURITY.md](SECURITY.md).

## Contributing

Bug reports, compatibility reports, and pull requests are welcome. See
[CONTRIBUTING.md](CONTRIBUTING.md) for the development setup, test suites, and
release process.

## License

[Mozilla Public License 2.0](LICENSE)
