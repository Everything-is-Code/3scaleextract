# Exploration: visualize-topology-report

## Current State

- `threescale-visualize` writes Markdown (`index.md`, `products/*.md`, Mermaid graph) and optional Cursor `.canvas.tsx` (`--canvas`).
- Canvas embeds `BuildCanvasData()` with product table (auth, backends, policies, policy names), charts, and product→backend graph.
- Markdown report lacks a consolidated product catalog; Mermaid does not scale for large tenants.

## Affected Areas

- `internal/visualize/canvas_data.go` — shared topology payload
- `internal/visualize/report.go` — new catalog renderer
- `internal/visualize/html.go` (new) — self-contained HTML dashboard
- `internal/visualize/cli/cli.go` — `--html` flag
- `docs/VISUALIZE.md`, `docs/TEST_CASES.md`

## Recommendation

Phase 1: Markdown `products-catalog.md` from shared payload.
Phase 2: Self-contained `topology.html` replicating canvas sections (stats, charts, graph, sortable table) via embedded JSON + CDN chart library + vanilla JS.

## Ready for Proposal

Yes.
