# Proposal: Visualize topology catalog (Markdown + HTML)

## Intent

Stakeholders need a **portable product topology view** (products, backends, auth, policy count and names) without Cursor IDE. The Markdown report today splits this across many files; the canvas has the right UX but requires Cursor. Add catalog Markdown and a browser-openable HTML dashboard for client workshops.

## Scope

### In Scope

- `products-catalog.md` in report output: sortable columns as Markdown table (Product, Category, Auth, Backends, Apps, Policies, Policy names)
- `topology.html`: self-contained dashboard mirroring canvas (summary stats, domain pie, shared-backends bar, product→backend graph, filterable/sortable product table, shared-backend section)
- Shared topology payload builder reused by canvas, catalog, and HTML
- CLI flag `--html` (writes `{output}/topology.html`; default off)
- Link from `index.md` to catalog and HTML
- Tests with `export-minimal` fixture only; demo HTML generated in CI/docs from fixture
- Update `docs/VISUALIZE.md` and TC-VIZ test cases

### Out of Scope

- Live Admin API calls; editing exports
- npm/webpack frontend toolchain in repo
- Replacing or removing `--canvas` (Cursor remains optional)
- De-redacting secrets

## Capabilities

### New Capabilities

- `visualize-catalog`: Markdown product catalog derived from export topology data
- `visualize-html-dashboard`: Self-contained HTML topology dashboard for browser viewing

### Modified Capabilities

- `visualize-report`: `index.md` navigation links to catalog and HTML outputs

## Approach

Extract/rename `BuildCanvasData` → shared `BuildTopologyData`. Add `renderProductsCatalog` and `WriteTopologyHTML` with `//go:embed html/topology.html.tmpl`. HTML embeds JSON payload; Chart.js (CDN) for charts; vanilla JS for table sort/filter/pagination and SVG graph layout (same logic as canvas template, port to JS).

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/visualize/catalog.go` | New | Markdown catalog renderer |
| `internal/visualize/html.go` | New | HTML writer + embed |
| `internal/visualize/html/topology.html.tmpl` | New | Dashboard template |
| `internal/visualize/canvas_data.go` | Modified | Rename/export shared builder |
| `internal/visualize/report.go` | Modified | Link catalog in index |
| `internal/visualize/cli/cli.go` | Modified | `--html` flag |
| `docs/examples/topology-demo.html` | New | Fixture-based demo |
| `docs/VISUALIZE.md` | Modified | Usage docs |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Large HTML file for 100+ products | Med | Pagination in JS; lazy graph render |
| Canvas/HTML drift | Med | Single JSON schema; shared Go builder |
| CDN unavailable offline | Low | Document offline limitation; charts degrade gracefully |
| Customer data in committed demos | Med | Only `export-minimal` in repo; `.gitignore` generated reports |

## Rollback Plan

Remove `--html` flag, catalog renderer, and HTML template. Revert to canvas-only interactive view. No export schema changes.

## Dependencies

None external to repo (Chart.js via CDN at view time only).

## Success Criteria

- [ ] `threescale-visualize export-minimal -o /tmp/r` produces `products-catalog.md` with auth and policy names
- [ ] `--html` produces openable `topology.html` with stats, charts, table, and graph
- [ ] No customer names in committed code/tests/SDD
- [ ] `go test ./...` passes; coverage ≥ 80%
