---
page_title: "Authentication - Grouper Provider"
subcategory: ""
description: |-
  How to authenticate the Grouper provider against the Grouper Web Services API.
---

# Authentication

The provider authenticates to the Grouper Web Services (WS) REST API using
**HTTP Basic authentication** with a Grouper service account.

## Credentials

Supply credentials via environment variables so they never appear in
configuration files or Terraform state:

```sh
export GROUPER_BASE_URL="https://grouper.example.edu/grouper-ws/servicesRest"
export GROUPER_USERNAME="terraform-svc"
export GROUPER_PASSWORD="…"
```

You may also set `base_url` (and, less ideally, `username`) directly in the
`provider` block, but keep `password` out of configuration.

```hcl
provider "grouper" {
  base_url = "https://grouper.example.edu/grouper-ws/servicesRest"
  # username / password come from GROUPER_USERNAME / GROUPER_PASSWORD
}
```

## Least privilege

Use a dedicated service account scoped to only the folders (stems) Terraform
manages. Depending on which resources you use, the account needs:

| To...                                   | Privilege                   | On                  |
| --------------------------------------- | --------------------------- | ------------------- |
| Create, update, or delete a folder      | `STEM_ADMIN`                | The parent folder   |
| Create a group                          | `CREATE`                    | The parent folder   |
| Update or delete a group                | `ADMIN`                     | The group           |
| Add or remove a member                  | `UPDATE` (or `ADMIN`)       | The group           |
| Read folders, groups, and members       | `VIEW` and `READ`           | The objects read    |

By default Grouper grants `ADMIN` on a new group to the account that created it, so groups
Terraform creates are manageable without extra grants. Granting privileges on a
parent folder with inheritance lets one grant cover everything Terraform builds
beneath it.

## API version

The provider talks to WS version segment `v2_6_000` by default, which is
confirmed against Grouper 4.16. Override with `ws_version` if your deployment
requires a different segment.

## TLS

Certificate verification is on by default. `insecure_skip_verify = true` disables
it and emits a warning; only use it against trusted development endpoints.

## What is never logged or stored

The provider never writes the password, `Authorization` header, cookies, or full
response bodies to logs, diagnostics, or state.
