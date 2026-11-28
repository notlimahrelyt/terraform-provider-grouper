---
page_title: "Environment variables - Grouper Provider"
subcategory: ""
description: |-
  Environment variables recognized by the Grouper provider.
---

# Environment variables

## Provider configuration

| Variable                  | Maps to           | Notes                                   |
| ------------------------- | ----------------- | --------------------------------------- |
| `GROUPER_BASE_URL`        | `base_url`        | Grouper WS REST base URL                |
| `GROUPER_USERNAME`        | `username`        | Service account username                |
| `GROUPER_PASSWORD`        | `password`        | Service account password (sensitive)    |
| `GROUPER_REQUEST_TIMEOUT` | `request_timeout` | Go duration string, default `30s`       |

`ws_version` and `insecure_skip_verify` are set in the `provider` block only.

An explicit value in the `provider` block takes precedence over the environment
variable.

Variables used only by the acceptance tests are listed in
[CONTRIBUTING.md](https://github.com/notlimahrelyt/terraform-provider-grouper/blob/main/CONTRIBUTING.md#acceptance-tests).
