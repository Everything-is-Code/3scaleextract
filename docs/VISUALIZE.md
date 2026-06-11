# Export visualizer (threescale-visualize)

**Optional** tool that generates a Markdown report from a directory exported by `threescale-export`. Useful for migration reviews without opening JSON/YAML manually.

To export a tenant, use the [main README](../README.md).

## Requirements

- Go 1.22+ (to compile; release binary does not require Go)
- An existing export with `schema_version` **1.0** (`manifest.json` at the root)
- **No** Admin API, Docker, or `THREESCALE_*` variables required

## Install

```bash
go build -o bin/threescale-visualize ./cmd/threescale-visualize
```

Or download `threescale-visualize-v*.*.*-linux-amd64.tar.gz` from [Releases](https://github.com/Everything-is-Code/3scaleextract/releases).

## Usage

```bash
# After an export (default ./export)
bin/threescale-visualize ./export -o ./report
```

| Flag | Description |
|------|-------------|
| `-o`, `--output` | Report directory (default `./report`) |
| `--version` | Binary version |

## Report layout

```
report/
├── index.md              # overview, auth matrix, Mermaid graph
├── backends.md           # backend catalog
├── applications.md       # only if export included applications
└── products/
    └── {system_name}.md  # auth, policies, plans, backends
```

Open `index.md` in GitHub, VS Code, or Cursor to navigate via relative links. Mermaid diagrams render on GitHub and in compatible editors.

## Demo with seed data

```bash
# 1. Load fixtures (see docs/SEED.md)
bin/threescale-seed

# 2. Export lab tenant
bin/threescale-export --output ./export --include-applications --redact-secrets

# 3. Generate report
bin/threescale-visualize ./export -o ./report
```

## Limitations (v1)

- Read-only from on-disk export; does not call Admin API
- Does not include `policies/catalog.json` content (global reference, not tenant config)
- Redacted secrets (`***REDACTED***`) are shown as-is — not de-redacted
- No HTTP server or HTML (planned for future versions)
