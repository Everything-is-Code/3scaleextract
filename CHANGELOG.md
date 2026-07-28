# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/Everything-is-Code/3scaleextract/compare/v0.4.2...HEAD
[0.4.2]: https://github.com/Everything-is-Code/3scaleextract/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/Everything-is-Code/3scaleextract/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/Everything-is-Code/3scaleextract/compare/v0.3.2...v0.4.0
[0.3.2]: https://github.com/Everything-is-Code/3scaleextract/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/Everything-is-Code/3scaleextract/releases/tag/v0.3.1
