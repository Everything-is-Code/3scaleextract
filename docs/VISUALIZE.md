# Export visualizer (threescale-visualize)

**Optional** tool that generates a Markdown report from a directory exported by `threescale-export`. Useful for migration reviews without opening JSON/YAML manually.

It can also generate a **self-contained HTML topology dashboard** (`topology.html`) for workshops in any browser, or a **Cursor IDE topology canvas** (`.canvas.tsx`) for interactive exploration of products, backends, applications, and policies.

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

# Report + HTML topology dashboard (open topology.html in a browser)
bin/threescale-visualize ./export -o ./report --html

# Report + Cursor topology canvas
bin/threescale-visualize ./export -o ./report --canvas ./topology.canvas.tsx
```

| Flag | Description |
|------|-------------|
| `-o`, `--output` | Report directory (default `./report`) |
| `--html` | Write self-contained topology dashboard (`topology.html` in output dir) |
| `--canvas` | Write a Cursor IDE topology canvas (`.canvas.tsx`) |
| `--version` | Binary version |

## Report layout

```
report/
├── index.md              # overview, domain summary, auth matrix, Mermaid graph
├── products-catalog.md   # sortable table: auth, backends, apps, policies
├── topology.html         # only with --html
├── backends.md           # backend catalog
├── applications.md       # only if export included applications
└── products/
    └── {system_name}.md  # auth, policies, plans, backends
```

Open `index.md` in GitHub, VS Code, or Cursor to navigate via relative links. The index includes a **Products by domain** table (count and percent per naming domain). Mermaid diagrams render on GitHub and in compatible editors.

## HTML topology dashboard

The HTML dashboard replicates the canvas UX (stats, charts, product→backend graph, sortable product table with policy names) without requiring Cursor. Generate it from the same export directory:

```bash
bin/threescale-visualize ./export -o ./report --html
```

Open `report/topology.html` in any modern browser. Chart.js is loaded from a CDN; the file embeds export data inline (no server required).

The **Products by domain** pie chart includes a **Show percentages** toggle that switches legend and tooltip labels between count-only (`Domain: N`) and count with percent (`Domain: N (P%)`). Domains with zero products are omitted from the pie.

### Demo HTML (lab fixture)

The repository includes a demo generated from the offline fixture (`seed_alpha`, `seed_multi_backend`):

- [`docs/examples/topology-demo.html`](examples/topology-demo.html)

Regenerate it after template changes:

```bash
bin/threescale-visualize internal/visualize/testdata/export-minimal \
  -o docs/examples --html
mv docs/examples/topology.html docs/examples/topology-demo.html
```

## Cursor topology canvas

The canvas is an optional interactive view (charts, product→backend graph, sortable product table with policy names). Generate it from the same export directory:

```bash
bin/threescale-visualize ./export --canvas ./topology.canvas.tsx
```

The **Products by domain** card matches the HTML toggle: use **Show percentages** to switch legend rows between count-only and count with percent. Domains with zero products are omitted from the pie.

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

# 3. Generate report, optional HTML dashboard, and optional canvas
bin/threescale-visualize ./export -o ./report --html --canvas ./topology.canvas.tsx
```

Use `--include-applications` on export when you want subscribed applications in the canvas graph.

## Policy visibility

Policy chains in the report, catalog, canvas, and HTML dashboard show **visible policies only**:

| Rule | Behavior |
|------|----------|
| `enabled: false` | Omitted from chain, count, and policy names column |
| `enabled` absent | Treated as enabled (fixture exports without the field) |
| `apicast` | Always omitted (built-in gateway policy, not user-configured) |

Sources: `products/{name}/policies.json` (wrapped `policies` or root `policies_config`), falling back to `proxy.json` when needed. Auth inference still reads raw `policies_config` unchanged.

## Limitations (v1)

- Read-only from on-disk export; does not call Admin API
- Does not include `policies/catalog.json` content (global reference, not tenant config)
- Policy chains are read from `policies.json` when present, otherwise from `proxy.json` (`policies_config`); only **enabled** user policies are shown (`apicast` excluded)
- Redacted secrets (`***REDACTED***`) are shown as-is — not de-redacted
- Canvas requires Cursor IDE; the Markdown report and HTML dashboard work everywhere
