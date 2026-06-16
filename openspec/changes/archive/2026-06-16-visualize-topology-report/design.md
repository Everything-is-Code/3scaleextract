# Design: Visualize topology catalog (Markdown + HTML)

## Technical Approach

Unify topology serialization in Go (`BuildTopologyData`), then fan out to three renderers: existing canvas TSX, new Markdown catalog, new HTML dashboard. HTML replicates canvas sections using the same embedded JSON blob and client-side Chart.js + vanilla JS (no build step).

## Architecture Decisions

| Decision | Choice | Alternatives | Rationale |
|----------|--------|--------------|-----------|
| Data source | Reuse `BuildCanvasData` → `BuildTopologyData` | Duplicate from `Tenant` in each renderer | Single schema; prevents canvas/HTML/catalog drift |
| HTML delivery | Self-contained file + CDN Chart.js | React SPA, iframe canvas | No npm toolchain; opens in any browser for clients |
| Catalog format | Separate `products-catalog.md` | Only extend `index.md` | Keeps index readable; full table for large tenants |
| CLI | `--html` opt-in, catalog always with report | `--html` default | Backward compatible; HTML is heavier |
| Graph in HTML | Port canvas DAG layout to JS/SVG | Static Mermaid only | Matches canvas interactivity |
| Styling | Minimal CSS in template (flat, no gradients) | Match Cursor tokens exactly | Good enough for workshops; avoid heavy design system |

## Data Flow

```
LoadExport(dir) → Tenant
       │
       ▼
BuildTopologyData(tenant) → TopologyData (JSON-serializable)
       │
       ├── WriteReport → index.md (+ link) + products-catalog.md
       ├── WriteTopologyHTML (--html) → topology.html
       └── WriteCanvasTSX (--canvas) → *.canvas.tsx
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/visualize/topology_data.go` | Create | Rename/move from `canvas_data.go`; type `TopologyData` |
| `internal/visualize/catalog.go` | Create | `renderProductsCatalog`, policy chain formatting |
| `internal/visualize/html.go` | Create | `WriteTopologyHTML`, JSON inject |
| `internal/visualize/html/topology.html.tmpl` | Create | Dashboard: stats, charts, graph, table |
| `internal/visualize/canvas.go` | Modify | Use `TopologyData` |
| `internal/visualize/report.go` | Modify | Write catalog; index links |
| `internal/visualize/cli/cli.go` | Modify | `--html` flag |
| `internal/visualize/*_test.go` | Create/Modify | Catalog + HTML smoke tests |
| `docs/examples/topology-demo.html` | Create | From export-minimal |
| `docs/VISUALIZE.md` | Modify | Document outputs |

## HTML Dashboard Sections (parity with canvas)

1. Header — export metadata (`exported_at`, `admin_url` from manifest)
2. Stat cards — products, backends, applications, link count
3. Charts — pie (domain), horizontal bar (shared backends)
4. Graph card — product selector, optional apps toggle, SVG DAG
5. Table — all products; sort/filter/pagination in JS
6. Shared backends — collapsible list

## Interfaces / Contracts

```go
type TopologyData struct { /* same fields as CanvasData today */ }

func BuildTopologyData(tenant *Tenant) TopologyData
func WriteProductsCatalog(t *Tenant, path string) error
func WriteTopologyHTML(t *Tenant, path string) error
```

JSON keys remain compact (`m`, `products`, `pol`, etc.) for parity with canvas.

## Testing Strategy

| Layer | What | How |
|-------|------|-----|
| Unit | Catalog markdown rows, policy chains | `export-minimal` tenant |
| Unit | HTML contains embedded JSON, no placeholder | `WriteTopologyHTML` |
| Unit | Topology builder unchanged behavior | Existing canvas tests updated |
| CLI | `--html` smoke | `cli_test.go` with temp dir |
| Golden | Optional substring checks for demo HTML | fixture product names only |

## Migration / Rollout

No migration. New files appear in report directory on next visualize run. Document in release notes.

## Open Questions

- [ ] Generate demo HTML in CI or only document manual regen command? → **Manual regen in docs** (same as canvas demo)
- [ ] Chart.js CDN vs vendored minimal SVG charts? → **CDN** for v1; note offline caveat in docs
