# Offline export fixtures

Versioned tarball of a minimal 3scale tenant export for offline tests and cross-repo import validation (GateForge INT-3).

## Version

| Field | Value |
|-------|-------|
| Export schema | **1.0** (`manifest.json`) |
| Artifact | `export-minimal-1.0.tar.gz` |
| Checksum | `export-minimal-1.0.tar.gz.sha256` |

Fixture version follows export **schema_version**, not the application release semver (`v0.1.x`).

## Contents

Source tree: `internal/visualize/testdata/export-minimal/`

| Item | Description |
|------|-------------|
| Products | `seed_alpha`, `seed_multi_backend` (YAML + JSON sidecars) |
| Backends | `shared_payments`, `billing_api` |
| Applications | `applications/page-1.json` |
| Manifest | 2 products, 2 backends, applications included |

Used by `internal/visualize` unit tests and documented in [docs/TEST_CASES.md](../docs/TEST_CASES.md) (TC-VIZ-001).

## Download

**Latest on main (raw):**

```text
https://github.com/Everything-is-Code/3scaleextract/raw/main/testdata/export-minimal-1.0.tar.gz
https://github.com/Everything-is-Code/3scaleextract/raw/main/testdata/export-minimal-1.0.tar.gz.sha256
```

**Release asset (semver-pinned app release):**

```text
https://github.com/Everything-is-Code/3scaleextract/releases/download/vX.Y.Z/export-minimal-1.0.tar.gz
https://github.com/Everything-is-Code/3scaleextract/releases/download/vX.Y.Z/export-minimal-1.0.tar.gz.sha256
```

## Verify and extract

```bash
curl -LO https://github.com/Everything-is-Code/3scaleextract/raw/main/testdata/export-minimal-1.0.tar.gz
curl -LO https://github.com/Everything-is-Code/3scaleextract/raw/main/testdata/export-minimal-1.0.tar.gz.sha256
sha256sum -c export-minimal-1.0.tar.gz.sha256
tar xzf export-minimal-1.0.tar.gz
```

Expected layout after extract:

```text
export-minimal/
├── manifest.json
├── products/
├── backends/
└── applications/
```

Example with visualize:

```bash
threescale-visualize export-minimal -o ./report
```

## Regenerate (maintainers)

After editing `internal/visualize/testdata/export-minimal/`:

```bash
./scripts/pack-export-minimal.sh
./scripts/pack-export-minimal.sh --check   # CI uses this
git add testdata/export-minimal-1.0.tar.gz testdata/export-minimal-1.0.tar.gz.sha256
```

## Related

- [3scaleextract #10](https://github.com/Everything-is-Code/3scaleextract/issues/10) — publish fixture tarball
- [GateForge #19](https://github.com/Everything-is-Code/gateforge/issues/19) — consume in import tests
