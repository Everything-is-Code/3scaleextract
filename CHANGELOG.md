# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.3] - 2026-08-19

### Added

- **threescale-seed** — `--fixtures PATH` loads an external YAML catalog (e.g. migration-toolkit-rhcl `testdata/seed/catalog.yaml`) instead of built-in fixtures.
- **threescale-seed** — Policy default configs for RHCL conversion demos: `logging`, `default_credentials`, `caching` / `3scale_auth_caching`, `payload_limits` / `content_limits`, `retry`, `keycloak_role_check`, `upstream_connection`, `headers`, `header_modification`, `token_introspection`, `edge_limiting`.

### Changed

- **threescale-seed** — `url_rewriting` seed config uses APIcast `commands` (regex/replace) instead of legacy `rules`, matching RHCL conversion input.
- **threescale-seed** — `jwt_claim_check` seed config includes explicit `op` / `jwt_claim_type` fields.
- Docs: `docs/SEED.md` documents `--fixtures`.

## [0.4.2] - 2026-07-28

### Fixed

- **threescale-visualize** — Topology HTML/canvas no longer crash when no backends are shared across products (`shared` marshals as `[]` instead of `null`).
- **threescale-visualize** — Backend usage ranking includes 1:1 product→backend links (not only multi-product shares); UI labels updated to “Most referenced backends” / “Backend usage detail”.

## [0.4.1] - 2026-07-13

### Fixed

- **threescale-export** — `--insecure` now passes `-k` to the 3scale toolbox so product YAML export succeeds on lab tenants with self-signed Admin Portal TLS (previously only Admin API and Analytics API skipped verification).
- **threescale-visualize** — Strip UTF-8 BOM from export artifacts; tolerate redacted or invalid product YAML without failing the full report (falls back to `system_name`).

### Changed

- README export example includes `--include-metrics` with a sample date window.
- README and TEST_CASES document unified `--insecure` behavior (Admin API, Analytics API, toolbox).
- GitHub Release workflow template documents metrics and TLS flags.

## [0.4.0] - 2026-07-09

### Added

- **threescale-export** — Optional Analytics API hit metrics export:
  - `--include-metrics` on full export writes `stats/query.json` and `stats/products/{system_name}/hits.json`
  - Standalone `threescale-export metrics` subcommand (stats only, no toolbox)
  - Manifest optional fields: `include_metrics`, metrics window, granularity, metric name (`schema_version` remains `1.0`)
  - `VerifyExport` validates stats layout when metrics are enabled

Requires Enterprise tier and a PAT with **Analytics** scope.

## [0.3.2] - 2026-06-18

### Added

- **threescale-visualize** — Products-by-domain summary table; HTML/canvas pie chart count/percent toggle.

## [0.3.1] - 2026-06-17

### Changed

- **threescale-visualize** — Policy chains show enabled user policies only (omit `enabled: false` and `apicast`).

[Unreleased]: https://github.com/Everything-is-Code/3scaleextract/compare/v0.4.3...HEAD
[0.4.3]: https://github.com/Everything-is-Code/3scaleextract/compare/v0.4.2...v0.4.3
[0.4.2]: https://github.com/Everything-is-Code/3scaleextract/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/Everything-is-Code/3scaleextract/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/Everything-is-Code/3scaleextract/compare/v0.3.2...v0.4.0
[0.3.2]: https://github.com/Everything-is-Code/3scaleextract/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/Everything-is-Code/3scaleextract/releases/tag/v0.3.1
