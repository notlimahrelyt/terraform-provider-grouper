---
page_title: "Troubleshooting - Grouper Provider"
subcategory: ""
description: |-
  Common Grouper provider errors and how to resolve them.
---

# Troubleshooting

Enable detailed logs with `TF_LOG=DEBUG` (or `TF_LOG_PROVIDER=DEBUG`). The
provider logs HTTP method, sanitized path, status, duration, and Grouper result
codes — never credentials or subject details.

## Authentication failed

> Grouper authentication failed. Check the provider username/password.

The service account username/password is wrong or not set. Verify
`GROUPER_USERNAME` / `GROUPER_PASSWORD`.

## Forbidden / insufficient privileges

> Grouper returned a forbidden result … Confirm the provider service account has
> the required privileges …

The account authenticated but lacks the privilege for the operation (for example
`CREATE` on the parent folder to create a group, or `UPDATE` on a group to change
its members). The [authentication guide](authentication) lists the privilege
each operation needs.

## Subject not found

> Grouper could not resolve subject "…". Confirm the subject id/identifier and
> (if needed) set `subject_source_id`.

The value does not resolve as either a subject id or identifier. At many
institutions a login name (e.g. a NetID) is the *identifier*, not the id — the provider tries both.
If the same value exists in multiple sources, set `subject_source_id` to
disambiguate. Use the `grouper_subject` data source to confirm resolution.

## "Not expecting a blank string for version"

The `ws_version` segment is empty or invalid. The default is `v2_6_000`; set
`ws_version` to a segment your server accepts.

## Stem/group won't delete

Grouper refuses to delete a stem that still contains child stems/groups. Remove
or move the children first. The provider surfaces this rather than cascading a
delete.

## Perpetual diffs

Usually caused by managing the same object from two Terraform states/modules.
Consolidate management to a single state.
