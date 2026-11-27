# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0]

Initial public release.

### Added

- Provider configuration with environment-variable fallbacks
  (`GROUPER_BASE_URL`, `GROUPER_USERNAME`, `GROUPER_PASSWORD`,
  `GROUPER_REQUEST_TIMEOUT`), configurable `ws_version`, request timeout, and
  optional `insecure_skip_verify`.
- Reusable Grouper Web Services API client with typed errors, shared result-
  envelope handling (including application-level failures returned on HTTP 200),
  and sanitized debug logging.
- `grouper_stem` resource: create, read, update, delete, and import
  (by UUID or `name:<full-name>`).
- `grouper_group` resource: create, read, update, delete, and import
  (by UUID or `name:<full-name>`).
- `grouper_membership` resource: manages a single direct (immediate)
  membership; idempotent create, safe delete, replace on identity change, and
  import (`<group_name>/<subject_source_id>/<subject_id>`). Never adds or removes
  members it does not manage.
- `grouper_stem`, `grouper_group`, and `grouper_subject` data sources. The
  subject data source can return opt-in subject attributes via
  `subject_attribute_names`.
- Unit tests (mock HTTP server) and gated acceptance tests, including a check
  that deleting one membership does not affect unrelated members.

### Notes

- Verified against Grouper server 4.16.0 using WS version segment `v2_6_000` and
  HTTP Basic authentication.

[Unreleased]: https://github.com/notlimahrelyt/terraform-provider-grouper/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/notlimahrelyt/terraform-provider-grouper/releases/tag/v0.1.0
