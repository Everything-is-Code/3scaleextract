# Test cases catalog

Consolidated test cases for common user journeys across `threescale-export`, `threescale-seed`, and `threescale-visualize`, plus the lab pipeline and CI workflows.

**Audience:** contributors, product owners, and downstream consumers (e.g. GateForge offline tests).

This catalog complements `go test ./...`; it does not replace running the test suite or lab validation.

## How to read this catalog

| Field | Meaning |
|-------|---------|
| **ID** | Stable identifier: `TC-{AREA}-{NNN}` where `AREA` is `EXP`, `SEED`, `VIZ`, `PIPE`, or `CI` |
| **Priority** | P0 = core happy path; P1 = lab validation; P2 = errors and edges; P3 = CI automation |
| **CLI** | Binary or workflow involved |
| **Automation** | Existing Go test function(s), `manual` lab procedure, or `gap` (not yet automated) |

Automation references are verified against the repository at the time of writing. Update this document when adding or renaming tests.

## Summary

| Metric | Count |
|--------|------:|
| Total cases | 18 |
| Automated | 15 |
| Manual | 2 |
| Gap | 1 |

| Priority | Count |
|----------|------:|
| P0 | 5 |
| P1 | 5 |
| P2 | 6 |
| P3 | 2 |

---

## threescale-export

### TC-EXP-001 — Default export layout

| Field | Value |
|-------|-------|
| Priority | P0 |
| CLI | `threescale-export` |
| Automation | `TestExportDefaultScope` (`internal/export/exporter_test.go`), `TestVerifyExportLayout`, `TestExportGoldenLayout` (`internal/export/verify_test.go`) |

**Preconditions**

- Valid `THREESCALE_ADMIN_URL` and `THREESCALE_ACCESS_TOKEN` (or flags)
- Docker or Podman on `PATH` and Red Hat toolbox image available

**Steps**

1. Run `threescale-export --output ./export`
2. Inspect the output directory

**Expected results**

- `manifest.json` with `schema_version: "1.0"`
- `products/{system_name}.yaml` (toolbox) and `products/{system_name}/*.json` per [README export layout](../README.md#export-layout)
- `backends/{system_name}.json` for each backend
- `policies/catalog.json` present

**Notes**

- Unit test uses mock Admin API and mock toolbox. Live behavior is covered by TC-CI-002.
- `TestVerifyExportLayout` asserts manifest fields and on-disk counts after mock export.
- `TestExportGoldenLayout` compares relative paths to `internal/export/testdata/golden-export-layout.txt` (10 files for default mock scope).

---

### TC-EXP-002 — Export with `--include-applications`

| Field | Value |
|-------|-------|
| Priority | P0 |
| CLI | `threescale-export` |
| Automation | `TestExportIncludeApplications` (`internal/export/exporter_test.go`) |

**Preconditions**

- Same as TC-EXP-001
- Tenant has at least one application (lab: use seeded fixtures)

**Steps**

1. Run `threescale-export --output ./export --include-applications`
2. Inspect `applications/` and `accounts/`

**Expected results**

- `manifest.json` includes `include_applications: true`
- `applications/page-{n}.json` paginated files
- `accounts/{id}.json` for linked accounts

---

### TC-EXP-003 — Export with `--redact-secrets`

| Field | Value |
|-------|-------|
| Priority | P0 |
| CLI | `threescale-export` |
| Automation | `TestExportRedactSecrets`, `TestExportWithoutRedactPreservesSecrets`, `TestExportRedactSecretsVerifyGateFailsOnResidual` (`internal/export/exporter_test.go`); `TestVerifyNoCleartextSecretsClean`, `TestVerifyNoCleartextSecretsFailsWithPath` (`internal/export/verify_test.go`); `TestRedactExtendedSensitiveKeys`, `TestRedactIssuerStripsUserinfoJSON`, `TestRedactPreservesAuthProxyFlags`, `TestContainsCleartextSecretYAMLIssuerUserinfo` (`internal/export/redact_test.go`) |

**Preconditions**

- Same as TC-EXP-001
- Export includes credentials (API keys, OIDC secrets, application identifiers, issuer URLs with embedded credentials, etc.)

**Steps**

1. Run `threescale-export --output ./export --redact-secrets --include-applications`
2. Search output for cleartext secrets and issuer URL userinfo

**Expected results**

- Core and extended sensitive keys (`provider_verification_key`, `client_id`, `app_id`, plus existing secret keys) replaced with `***REDACTED***` in JSON and YAML artifacts
- `issuer_endpoint` and `oidc_issuer_endpoint` have URL userinfo stripped; host and path remain visible
- `auth_user_key`, `auth_app_id`, and `auth_app_key` are unchanged (auth-mode flags, not secrets)
- Post-redaction cleartext scan passes; export completes without error
- If cleartext remains after redaction, export fails with a path-qualified error (e.g. `cleartext secret in products/api/proxy.json`)

**Opt-in default (no flag)**

| Field | Value |
|-------|-------|
| Automation | `TestExportWithoutRedactPreservesSecrets` (`internal/export/exporter_test.go`) |

**Expected results (without `--redact-secrets`)**

- Credential values remain cleartext in JSON and YAML artifacts
- No `***REDACTED***` markers in output

---

### TC-EXP-004 — Missing authentication

| Field | Value |
|-------|-------|
| Priority | P2 |
| CLI | `threescale-export` |
| Automation | `TestRunExportMissingAuth`, `TestExecuteMissingAuth` (`internal/cli/cli_test.go`), `TestValidateAuthMissingURL`, `TestValidateAuthMissingToken` (`internal/config/config_test.go`) |

**Preconditions**

- `THREESCALE_ADMIN_URL` and/or `THREESCALE_ACCESS_TOKEN` unset and not passed via flags

**Steps**

1. Run `threescale-export --output ./export` without credentials

**Expected results**

- Non-zero exit code
- Clear error message indicating missing admin URL or token

---

### TC-EXP-005 — Toolbox runtime unavailable

| Field | Value |
|-------|-------|
| Priority | P2 |
| CLI | `threescale-export` |
| Automation | `TestResolveContainerRuntimeMissing`, `TestNewToolboxDefaults` (`internal/export/toolbox_test.go`) |

**Preconditions**

- No `docker` or `podman` on `PATH`
- `THREESCALE_TOOLBOX_BINARY` unset

**Steps**

1. Attempt export against a valid tenant

**Expected results**

- Export fails before or during toolbox product export
- Error indicates docker or podman is required (see Red Hat toolbox docs)

---

### TC-EXP-006 — Self-signed TLS (lab tenant)

| Field | Value |
|-------|-------|
| Priority | P2 |
| CLI | `threescale-export` |
| Automation | **manual** |

**Preconditions**

- Lab Admin Portal uses self-signed or private CA certificate
- Valid credentials

**Steps**

1. Export with `--insecure` for Admin API TLS, and/or set `THREESCALE_TOOLBOX_TLS_CERT` for toolbox container TLS
2. For integration CI: set repository secret `THREESCALE_INSECURE_TLS=true`

**Expected results**

- Export completes against lab tenant without TLS verification errors

**Notes**

- Lab-only configuration. Do not use `--insecure` in production.

---

### TC-EXP-007 — Strict export (`--strict`)

| Field | Value |
|-------|-------|
| Priority | P2 |
| CLI | `threescale-export` |
| Automation | `TestExportStrictFailsOnMissingSidecar` (`internal/export/verify_test.go`), `TestExportRecordsWarningsOnSkippedSidecars` (`internal/export/exporter_test.go`) |

**Preconditions**

- Valid credentials and toolbox (same as TC-EXP-001)
- At least one product sidecar GET fails (mock: missing optional JSON such as `oidc_configuration.json`)

**Steps**

1. Run export **without** `--strict` when a sidecar is unavailable
2. Run export **with** `--strict` under the same failure condition

**Expected results**

- Default: export completes with `manifest.warnings[]` entries and `incomplete: true` when sidecars are skipped (`RecordSkip`)
- `--strict`: export aborts with `ErrStrictSidecar`; manifest may still be written with `incomplete: true`
- Visualizer surfaces warnings in the report index (`## Export warnings`)

---

## threescale-seed

### TC-SEED-001 — Dry-run seed (no API writes)

| Field | Value |
|-------|-------|
| Priority | P1 |
| CLI | `threescale-seed` |
| Automation | `TestDryRunSeeder` (`internal/seed/fixtures_test.go`) |

**Preconditions**

- Valid credentials configured (dry-run still validates auth path in CLI)

**Steps**

1. Run `threescale-seed --dry-run`

**Expected results**

- No Admin API create/update calls
- Fixture plan printed (products, backends, policies)

---

### TC-SEED-002 — Skip-existing idempotent load

| Field | Value |
|-------|-------|
| Priority | P1 |
| CLI | `threescale-seed` |
| Automation | `TestSkipExistingService`, `TestSkipExistingBackend` (`internal/seed/seeder_test.go`) |

**Preconditions**

- Lab tenant reachable with admin token
- Fixtures may or may not already exist

**Steps**

1. Run `threescale-seed --skip-existing` twice against the same tenant

**Expected results**

- First run creates missing resources
- Second run skips existing `system_name` resources without error
- OIDC applications for `seed_oidc` are recreated on each run (see [SEED.md](SEED.md))

---

### TC-SEED-003 — Fixture coverage matrix completeness

| Field | Value |
|-------|-------|
| Priority | P0 |
| CLI | `threescale-seed` |
| Automation | `TestDefaultFixturesCoverage` (`internal/seed/fixtures_test.go`) |

**Preconditions**

- None (offline code validation)

**Steps**

1. Run `go test ./internal/seed/...`
2. Review `CoverageMatrix` in `internal/seed/fixtures.go` and [SEED.md export coverage matrix](SEED.md#export-coverage-matrix)

**Expected results**

- All four default products (`seed_api_key`, `seed_oidc`, `seed_app_id`, `seed_multi_backend`) have coverage entries
- Each row documents export dimensions to verify manually after TC-PIPE-001

---

### TC-SEED-004 — Seed HTTP/API errors

| Field | Value |
|-------|-------|
| Priority | P2 |
| CLI | `threescale-seed` |
| Automation | `TestHTTPErrorBackend`, `TestHTTPErrorService` (`internal/seed/seeder_test.go`) |

**Preconditions**

- Unit test uses mock server returning errors

**Steps**

1. Run seed tests with simulated 4xx/5xx responses

**Expected results**

- Seeder returns error with context (backend or service failure)
- No partial silent success

---

### TC-SEED-005 — List fixtures CLI smoke

| Field | Value |
|-------|-------|
| Priority | P1 |
| CLI | `threescale-seed` |
| Automation | `TestListFixtures` (`cmd/threescale-seed/main_test.go`) |

**Preconditions**

- None

**Steps**

1. Run `threescale-seed --list-fixtures`

**Expected results**

- Exit code 0
- stdout includes fixture names and coverage hints

---

## threescale-visualize

### TC-VIZ-001 — Visualize valid export directory

| Field | Value |
|-------|-------|
| Priority | P0 |
| CLI | `threescale-visualize` |
| Automation | `TestLoadExportMinimal`, `TestLoadExportMultiBackendJoins`, `TestWriteReportBundle` (`internal/visualize/loader_test.go`, `internal/visualize/report_test.go`) |

**Preconditions**

- Valid export directory with `manifest.json` schema 1.0
- Offline fixture: `internal/visualize/testdata/export-minimal/` or published tarball `testdata/export-minimal-1.0.tar.gz` ([testdata/README.md](../testdata/README.md))

**Steps**

1. Load fixture in unit tests, or manually: `threescale-visualize ./export -o ./report`
2. Open `report/index.md`

**Expected results**

- Report directory contains `index.md`, `products-catalog.md`, `backends.md`, `products/{system_name}.md`
- `applications.md` when export included applications
- Mermaid graph and auth matrix in index

**Notes**

- Offline fixture uses `seed_alpha` / `seed_multi_backend` naming — not identical to live seed fixture names. See [#10](https://github.com/Everything-is-Code/3scaleextract/issues/10) for versioned tarball strategy.

---

### TC-VIZ-002 — Invalid or missing export

| Field | Value |
|-------|-------|
| Priority | P2 |
| CLI | `threescale-visualize` |
| Automation | `TestLoadExportMissingManifest`, `TestLoadExportUnsupportedSchema` (`internal/visualize/loader_test.go`) |

**Preconditions**

- Directory without `manifest.json`, or manifest with unsupported `schema_version`

**Steps**

1. Run visualize against invalid input directory

**Expected results**

- Non-zero exit or load error
- Clear message about missing manifest or unsupported schema

---

### TC-VIZ-003 — Visualize CLI against fixture directory

| Field | Value |
|-------|-------|
| Priority | P1 |
| CLI | `threescale-visualize` |
| Automation | **gap** |

**Preconditions**

- Built `threescale-visualize` binary
- `internal/visualize/testdata/export-minimal/` present

**Steps**

1. Run `threescale-visualize internal/visualize/testdata/export-minimal -o /tmp/report`
2. Verify report files exist

**Expected results**

- Exit code 0
- Report layout matches TC-VIZ-001

**Notes**

- Loader/report package tests cover logic; CLI entrypoint from `cmd/threescale-visualize` is not yet smoke-tested against fixture dir. Only `TestVisualizeHelp` exists in `cmd/threescale-visualize/main_test.go`.

---

### TC-VIZ-004 — Generate Cursor topology canvas from export

| Field | Value |
|-------|-------|
| Priority | P2 |
| CLI | `threescale-visualize --canvas` |
| Automation | **covered** (`TestVisualizeCanvasFlag`, `TestWriteCanvasTSX`) |

**Preconditions**

- Built `threescale-visualize` binary
- Valid export directory (fixture or lab export)

**Steps**

1. Run `threescale-visualize ./export --canvas ./topology.canvas.tsx`
2. Open the generated file in Cursor (`~/.cursor/projects/<workspace>/canvases/`)

**Expected results**

- Exit code 0
- `.canvas.tsx` contains `TopologyCanvas`, embedded product data, and `cursor/canvas` import
- No customer-specific paths or tenant names in committed demo (`docs/examples/topology-demo.canvas.tsx` uses `seed_alpha` fixture)

**Notes**

- Canvas is optional; Markdown report is still generated by default
- Policy chains fall back to `proxy.json` when `policies.json` is empty

---

### TC-VIZ-005 — Generate HTML topology dashboard from export

| Field | Value |
|-------|-------|
| Priority | P2 |
| CLI | `threescale-visualize --html` |
| Automation | **covered** (`TestVisualizeHTMLFlag`, `TestWriteTopologyHTML`, `TestWriteProductsCatalog`) |

**Preconditions**

- Built `threescale-visualize` binary
- Valid export directory (fixture or lab export)

**Steps**

1. Run `threescale-visualize ./export -o ./report --html`
2. Open `report/topology.html` in a browser

**Expected results**

- Exit code 0
- `topology.html` is self-contained with embedded JSON and Chart.js CDN
- `index.md` links to `topology.html` and `products-catalog.md`
- Demo artifact `docs/examples/topology-demo.html` uses fixture names only (`seed_alpha`, `seed_multi_backend`)

---

### TC-VIZ-006 — Visible policies only (enabled, no apicast)

| Field | Value |
|-------|-------|
| Priority | P1 |
| CLI | `threescale-visualize` |
| Automation | **covered** (`TestVisiblePoliciesFromConfig`, `TestParsePoliciesPoliciesConfigRoot`, `TestPolicyNamesFromProxyFile`) |

**Preconditions**

- Export with `policies_config` entries mixing `enabled: true/false` and terminal `apicast`

**Steps**

1. Run visualize against fixture or unit-test JSON with disabled policy
2. Inspect product page, catalog, canvas/HTML policy columns

**Expected results**

- Disabled policies (`enabled: false`) absent from chain, count, and names
- `apicast` never listed
- Missing `enabled` field treated as enabled (fixture backward compat)

---

## Lab pipeline

### TC-PIPE-001 — Seed → export → visualize

| Field | Value |
|-------|-------|
| Priority | P1 |
| CLI | `threescale-seed`, `threescale-export`, `threescale-visualize` |
| Automation | **manual** |

**Preconditions**

- Lab tenant credentials (`THREESCALE_ADMIN_URL`, `THREESCALE_ACCESS_TOKEN`)
- Docker/Podman + Red Hat toolbox image for export
- Optional: `source scripts/load-env.sh`

**Steps**

1. `threescale-seed --skip-existing` (or `./scripts/demo/seed-and-export.sh` for seed + export only)
2. `threescale-export --output ./export --include-applications --redact-secrets`
3. Verify [SEED coverage matrix](SEED.md#export-coverage-matrix) rows in `./export`
4. `threescale-visualize ./export -o ./report`
5. Open `report/index.md` and spot-check product/backend pages

**Expected results**

- Export artifacts match README layout
- Each seeded product dimension (auth mode, policies, backends) visible in export
- Visualize report navigable with relative links

**Notes**

- PR template checklist: "Tested in local lab (seed → export → analyze)"
- Full pipeline automation is a known **gap** for a future `[TEST]` issue

---

## CI and automation

### TC-CI-001 — PR unit CI

| Field | Value |
|-------|-------|
| Priority | P3 |
| CLI | GitHub Actions `CI` workflow |
| Automation | `.github/workflows/ci.yml`, `scripts/check-coverage.sh`, `scripts/pack-export-minimal.sh --check` |

**Preconditions**

- Pull request or push to `main`

**Steps**

1. Open PR; wait for CI workflow

**Expected results**

- `go test ./...` passes (integration tag excluded) with coverage profile
- Statement coverage meets the repository threshold (currently 80% via `check-coverage.sh`)
- golangci-lint passes (includes `revive`, `bodyclose`, `errorlint`)
- `export-minimal` tarball freshness check passes
- All three CLI binaries build successfully

---

### TC-CI-002 — Integration workflow dispatch

| Field | Value |
|-------|-------|
| Priority | P3 |
| CLI | GitHub Actions `Integration` workflow |
| Automation | `.github/workflows/integration.yml`, `TestIntegrationExport`, `export.VerifyExport` (`internal/export/integration_test.go`, `internal/export/verify.go`) |

**Preconditions**

- Required secrets configured: `THREESCALE_ADMIN_URL`, `THREESCALE_ACCESS_TOKEN`, `THREESCALE_OUTPUT_DIR`
- Or secrets absent to test skip path

**Steps**

1. Actions → Integration → Run workflow
2. Review job logs

**Expected results**

- With secrets: `go test -tags=integration ./internal/export/...` runs live export
- After live export: `VerifyExport` checks `manifest.json` (schema 1.0, `admin_url`, `exported_at`), required directories, and product/backend file counts
- Without secrets: job succeeds with skip message (no spurious failure)

**Notes**

- `VerifyExport` does not fail on `incomplete: true` — lab tenants may have partial sidecars; use `--strict` locally when you need fail-fast behavior (TC-EXP-007).

---

## Gap backlog

Cases or aspects marked `gap` — candidates for future `[TEST]` issues:

| Gap | Related case | Suggested follow-up |
|-----|--------------|---------------------|
| Visualize CLI smoke against fixture | TC-VIZ-003 | Add `cmd/threescale-visualize` test with `export-minimal` |
| Full pipeline automation | TC-PIPE-001 | Script or e2e test for seed → export → visualize |
| Offline fixture parity | TC-VIZ-001 | [#10](https://github.com/Everything-is-Code/3scaleextract/issues/10) — versioned `export-minimal` tarball aligned with seed catalog |

Implementing these gaps is **out of scope** for the test-case definition work ([issue #3](https://github.com/Everything-is-Code/3scaleextract/issues/3)).

---

## Related documentation

| Document | Content |
|----------|---------|
| [README.md](../README.md) | Export prerequisites, layout, local tests, Integration CI |
| [SEED.md](SEED.md) | Lab fixtures, coverage matrix, seed-and-export script |
| [VISUALIZE.md](VISUALIZE.md) | Report layout, demo pipeline |
| [CONTRIBUTING.md](../CONTRIBUTING.md) | Development and PR expectations |
| Issue [#10](https://github.com/Everything-is-Code/3scaleextract/issues/10) | Versioned `export-minimal` tarball for offline consumers |
