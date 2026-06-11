# Export visualizer (threescale-visualize)

**Optional** tool that generates a Markdown report from a directory exported by `threescale-export`. Useful for migration reviews without opening JSON/YAML manually.

It can also generate a **Cursor IDE topology canvas** (`.canvas.tsx`) for interactive exploration of products, backends, applications, and policies.

To export a tenant, use the [main README](../README.md).

## Requirements

- Go 1.22+ (to compile; release binary does not require Go)
- An existing export with `schema_version` **1.0** (`manifest.json` at the root)
- **No** Admin API, Docker, or `THREESCALE_*` variables required
- Cursor IDE (only if you open the optional topology canvas)

## Install

```bash
go build -o bin/threescale-visualize ./cmd/threescale-visualize
```

Or download `threescale-visualize-v*.*.*-linux-amd64.tar.gz` from [Releases](https://github.com/Everything-is-Code/3scaleextract/releases).

## Usage

```bash
# Markdown report (default ./report)
bin/threescale-visualize ./export -o ./report

# Report + Cursor topology canvas
bin/threescale-visualize ./export -o ./report --canvas ./topology.canvas.tsx
```

| Flag | Description |
|------|-------------|
| `-o`, `--output` | Report directory (default `./report`) |
| `--canvas` | Write a Cursor IDE topology canvas (`.canvas.tsx`) |
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

## Cursor topology canvas

The canvas is an optional interactive view (charts, product→backend graph, sortable product table with policy names). Generate it from the same export directory:

```bash
bin/threescale-visualize ./export --canvas ./topology.canvas.tsx
```

### Open the canvas in Cursor

1. Copy or move the generated file into your Cursor project canvases folder:
   `~/.cursor/projects/<workspace-id>/canvases/topology.canvas.tsx`
2. Open the file from the Cursor canvases panel (beside the chat).

The canvas embeds data from **your local export only**. Do not commit generated `.canvas.tsx` files from production tenants to public repositories.

### Demo canvas (lab fixture)

The repository includes a demo generated from the offline fixture (`seed_alpha`, `seed_multi_backend`):

- [`docs/examples/topology-demo.canvas.tsx`](examples/topology-demo.canvas.tsx)

Regenerate it after template changes:

```bash
bin/threescale-visualize internal/visualize/testdata/export-minimal \
  --canvas docs/examples/topology-demo.canvas.tsx
```

## Demo with seed data

```bash
# 1. Load fixtures (see docs/SEED.md)
bin/threescale-seed

# 2. Export lab tenant
bin/threescale-export --output ./export --include-applications --redact-secrets

# 3. Generate report and optional canvas
bin/threescale-visualize ./export -o ./report --canvas ./topology.canvas.tsx
```

Use `--include-applications` on export when you want subscribed applications in the canvas graph.

## Limitations (v1)

- Read-only from on-disk export; does not call Admin API
- Does not include `policies/catalog.json` content (global reference, not tenant config)
- Policy chains are read from `policies.json` when present, otherwise from `proxy.json` (`policies_config`)
- Redacted secrets (`***REDACTED***`) are shown as-is — not de-redacted
- Canvas requires Cursor IDE; the Markdown report works everywhere
