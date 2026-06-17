# Design: Visualize enabled policies only

## Technical Approach

Filter policy chains once in `internal/visualize/loader.go` when building `product.Policies`. All renderers (product Markdown, catalog, `BuildTopologyData` → canvas/HTML) consume the filtered slice with no per-output logic.

Auth inference in `auth.go` continues to read raw `proxy.policies_config` JSON unchanged.

## Architecture Decisions

### Decision: Filter at load time, not render time

**Choice:** Unified parser + visibility filter in loader  
**Alternatives considered:** Filter in `report.go`, `topology_data.go`, each template  
**Rationale:** Single source of truth; prevents canvas/HTML/catalog drift; matches existing `BuildTopologyData` sharing pattern

### Decision: Missing `enabled` defaults to true

**Choice:** Use `*bool` (nil → enabled) or equivalent explicit check  
**Alternatives considered:** Go `bool` default false; treat absent as disabled  
**Rationale:** `export-minimal` fixtures omit `enabled`; naive unmarshal would hide all fixture policies

### Decision: Exclude `apicast` from display

**Choice:** Drop `name == "apicast"` after enabled filter  
**Alternatives considered:** Show all enabled including apicast; `--show-apicast` flag  
**Rationale:** User-confirmed; aligns Admin UI mental model and fixture/live parity

### Decision: Parse both export JSON shapes in one helper

**Choice:** Shared `visiblePoliciesFrom…` used by `parsePolicies` and `policyNamesFromConfig`  
**Alternatives considered:** Rely on proxy.json fallback only  
**Rationale:** Live `policies.json` uses root `policies_config`; fallback breaks when proxy sidecar missing

## Data Flow

```
policies.json ──┐
                ├──► parsePolicies / policyNamesFromConfig
proxy.json ─────┘         │
                          ▼
              visiblePoliciesFromJSON
                • parse policies[] OR policies_config[]
                • skip enabled == false
                • default absent enabled → true
                • skip name == "apicast"
                          │
                          ▼
                   []Policy on Product
                          │
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
    report.go      BuildTopologyData   (future consumers)
    Policy Chain   PolicyNames → canvas / HTML / catalog
```

Auth path (unchanged):

```
proxy.json policies_config (raw) ──► inferAuthTypeFromProxy / authTypeFromPoliciesConfig
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/visualize/loader.go` | Modify | `visiblePoliciesFromJSON`, refactor `parsePolicies`, `policyNamesFromConfig` |
| `internal/visualize/loader_test.go` | Modify | Table tests: both formats, enabled false, apicast, default-true |
| `internal/visualize/canvas_test.go` | Modify | `TestPolicyNamesFromProxyFile` expects filtered chain |
| `internal/visualize/report_test.go` | Modify | Policy chain excludes disabled |
| `internal/visualize/catalog_test.go` | Modify | Count/chain with filtered policies |
| `internal/visualize/testdata/export-minimal/...` | Modify | Optional: add `policies_config` sample with disabled entry |
| `docs/TEST_CASES.md` | Modify | TC-VIZ scenario for enabled-only policies |

No changes: `report.go`, `catalog.go`, `topology_data.go`, canvas/HTML templates.

## Interfaces / Contracts

```go
// visibility rules applied when building Product.Policies
func visiblePolicyEntries(entries []policyEntry) []Policy

// policyEntry — internal parse struct with Name string, Enabled *bool

const hiddenPolicyApicast = "apicast"
```

`Policy` struct remains `{ Name string }` — no `Enabled` field on exported model (filter applied before append).

## Parsing Rules

| Input shape | Source | Parse path |
|-------------|--------|------------|
| `{ "policies": [{ "policy": { "name" } }] }` | Fixture `policies.json` | Wrapped array; no `enabled` → visible |
| `{ "policies_config": [{ "name", "enabled" }] }` | Live `policies.json` | Root config array |
| `proxy.policies_config` | Fallback when policies.json empty | Same filter via shared helper |

Order preserved among visible entries.

## Testing Strategy

| Test | Assert |
|------|--------|
| `TestVisiblePoliciesFromConfig` | enabled false omitted; apicast omitted; default true |
| `TestLoadExportMinimal` | Unchanged: `seed_alpha` → `[cors]` |
| `TestLoadExportPoliciesConfigShape` | New inline or fixture dir with mixed flags |
| `TestWriteReportBundle` / catalog / canvas | Disabled name absent; apicast absent |
| Regression | `auth_test.go` unchanged |

Use inline JSON in unit tests where possible; extend `export-minimal` only if integration-style load test needed.

## Migration / Rollout

Behavior change only — no export schema change. Re-run `threescale-visualize` to refresh reports. Document in release notes (patch/minor).

## Open Questions

- [x] Exclude apicast? → **Yes**
- [ ] Extend `export-minimal` vs inline test JSON only? → **Prefer inline unit tests first**; extend fixture if load test adds value
