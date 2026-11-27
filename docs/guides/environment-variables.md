---
page_title: "Environment variables - Grouper Provider"
subcategory: ""
description: |-
  Environment variables recognized by the Grouper provider and its tests.
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

## Acceptance tests

These are used only by `make testacc` / `go test -run TestAcc`. They never affect
normal provider use.

| Variable                        | Purpose                                                        |
| ------------------------------- | ------------------------------------------------------------- |
| `TF_ACC`                        | Must be set (any value) to enable acceptance tests            |
| `GROUPER_TEST_STEM`             | Parent stem for test objects; **must match `test`** (guardrail) |
| `GROUPER_TEST_SUBJECT_ID`       | A resolvable subject id or login identifier; membership/subject tests skip without it |
| `GROUPER_TEST_SUBJECT_ID_2`     | A second subject; enables the "preserves unrelated members" test |
| `GROUPER_TEST_SUBJECT_SOURCE_ID`| Optional subject source id for the tests                       |

See the [acceptance testing guide](./acceptance-testing.md) for safety rules.
