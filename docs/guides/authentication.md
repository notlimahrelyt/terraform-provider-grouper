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
export GROUPER_USERNAME="svc-account"
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
manages. The account needs:

- **CREATE** on a parent stem to create child stems/groups within it.
- **ADMIN**/**UPDATE** on a stem or group to modify or delete it.
- **UPDATE** (member management) on a group to add/remove memberships.

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
