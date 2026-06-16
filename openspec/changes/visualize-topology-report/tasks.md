# Tasks: Visualize topology catalog (Markdown + HTML)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 800–1200 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR1: shared data + Markdown catalog · PR2: HTML dashboard + docs |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
400-line budget risk: High

### Suggested Work Units

1. **Topology data refactor** — rename `CanvasData` → `TopologyData`, update canvas tests
2. **Markdown catalog** — `catalog.go`, `report.go` links, tests, TC-VIZ
3. **HTML dashboard** — template, `html.go`, `--html` CLI, JS charts/graph/table
4. **Docs & demo** — `VISUALIZE.md`, `topology-demo.html` from export-minimal

---

## Phase 1: Shared topology data

- [ ] 1.1 Rename `canvas_data.go` → `topology_data.go`; export `BuildTopologyData`, type `TopologyData`
- [ ] 1.2 Update `canvas.go` / `canvas_test.go` to use new names (no behavior change)
- [ ] 1.3 Run `go test ./internal/visualize/...`

## Phase 2: Markdown product catalog

- [ ] 2.1 Add `catalog.go` with `WriteProductsCatalog(tenant, path)`
- [ ] 2.2 Wire into `WriteReport`; add link in `renderIndex`
- [ ] 2.3 Tests: catalog rows for `seed_alpha`, policy chain, backend count
- [ ] 2.4 Update `docs/TEST_CASES.md` (TC-VIZ-005 catalog)

## Phase 3: HTML topology dashboard

- [ ] 3.1 Create `html/topology.html.tmpl` — stats, Chart.js charts, table, graph sections
- [ ] 3.2 Add `html.go` with `WriteTopologyHTML` (embed template, inject JSON)
- [ ] 3.3 Port graph layout + table sort/filter/pagination to vanilla JS
- [ ] 3.4 Add `--html` flag to CLI; default path `{output}/topology.html` when flag without value (or explicit path)
- [ ] 3.5 Tests: HTML written, JSON valid, contains fixture product names only
- [ ] 3.6 Generate `docs/examples/topology-demo.html` from export-minimal

## Phase 4: Documentation & verify

- [ ] 4.1 Update `docs/VISUALIZE.md` — catalog, HTML, browser workflow, canvas comparison
- [ ] 4.2 Update `README.md` visualize section (brief)
- [ ] 4.3 `go test ./...`; coverage gate
- [ ] 4.4 Manual: open `topology-demo.html` in browser; verify table and charts

## Privacy checklist (every phase)

- [ ] P.1 No customer/tenant names in code, tests, commits, or SDD examples
- [ ] P.2 Demo artifacts regenerated only from `export-minimal`
- [ ] P.3 Grep committed diff before PR for forbidden terms
