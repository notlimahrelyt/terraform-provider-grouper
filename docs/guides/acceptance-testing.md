---
page_title: "Acceptance testing safety - Grouper Provider"
subcategory: ""
description: |-
  How to run acceptance tests safely against a real Grouper deployment.
---

# Acceptance testing safety

Acceptance tests create and destroy **real** Grouper objects. They are disabled
unless `TF_ACC` is set, and they refuse to run against anything that does not
look like a test namespace.

## Hard requirements

- `TF_ACC=1`
- `GROUPER_BASE_URL`, `GROUPER_USERNAME`, `GROUPER_PASSWORD`
- `GROUPER_TEST_STEM` — a dedicated test folder. The test harness validates that
  its value **matches `test`** (case-insensitive) and aborts otherwise. This
  guardrail prevents tests from ever touching a production namespace.

Membership and subject tests additionally require a resolvable
`GROUPER_TEST_SUBJECT_ID`; they skip (do not fail) if it is absent.

## Running

```sh
export TF_ACC=1
export GROUPER_BASE_URL="https://grouper.example.edu/grouper-ws/servicesRest"
export GROUPER_USERNAME="svc-account"
export GROUPER_PASSWORD="…"
export GROUPER_TEST_STEM="example:apps:myteam:test"
export GROUPER_TEST_SUBJECT_ID="abc123"          # optional
export GROUPER_TEST_SUBJECT_ID_2="xyz456"        # optional: enables preservation test
make testacc
```

## Safety rules

- The `GROUPER_TEST_STEM` folder must already exist and the service account must
  have privileges to create/delete children within it.
- Every test object uses a unique `tfacc-*` name under the test stem.
- Tests clean up after themselves (Terraform destroy), including on failure.
- The membership tests assert that deleting one membership does **not** affect
  other members of the group.
- Never point acceptance tests at a production stem, and never reuse a stem that
  holds real memberships.
