# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Provider overview page on the Terraform Registry covering what the provider
  manages, authentication, environment variables, compatibility, and security.
- A complete example (`examples/complete`) that builds an application's folder,
  admin and user groups, and memberships.
- Lifecycle tests that run every resource and data source through real
  Terraform plans, applies, imports, and out-of-band changes against an
  in-memory Grouper, on Terraform 1.5 through 1.14.
- `CONTRIBUTING.md`, `SECURITY.md`, issue and pull request templates, and
  Dependabot configuration.

### Changed

- Attribute and resource descriptions are written in Markdown, so the Registry
  shows values such as `30s` and `v2_6_000` as code.
- The authentication guide lists the Grouper privilege each operation needs.
- Contributor-only guides (development and acceptance testing) moved from the
  Registry documentation to `CONTRIBUTING.md`.
- GitHub releases now use the matching `CHANGELOG.md` section as release notes.
- CI checks that generated documentation is current, validates it for the
  Registry, checks example formatting, and validates the release configuration.

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
