# Exploration: visualize-enabled-policies-only

## Current State

The visualizer loads product policy chains in `loadProduct` (`internal/visualize/loader.go`):

1. **`policies.json`** via `parsePolicies` — expects Admin API wrapped format `{ "policies": [{ "policy": { "name" } }] }` (used by `export-minimal` fixtures). Does **not** read `enabled`. Does **not** handle root-level `policies_config`.
2. **Fallback** when `len(product.Policies) == 0` — re-reads `proxy.json` via `policyNamesFromProxyFile` → `policyNamesFromConfig`, which unmarshals `proxy.policies_config` as `[{ "name" }]`. Also ignores `enabled`.

Live exports (Admin API `/services/{id}/proxy/policies`) store **`policies_config` at the root** of `policies.json`, not the wrapped `policies` array:

```json
{ "policies_config": [ { "name": "edge_limit", "enabled": true, ... }, { "name": "camel", "enabled": false, ... } ] }
```

Because `parsePolicies` misses this shape, the loader currently relies on the **proxy.json fallback** for live exports. Verified by running `threescale-visualize export-live -o /tmp/report-check`: `seed_multi_backend` shows `edge_limit`, `url_rewriting`, `apicast` (all from `proxy.policies_config`).

**No filtering anywhere downstream.** All renderers consume `product.Policies` or `BuildTopologyData` → `PolicyNames`:

| Consumer | File | Usage |
|----------|------|-------|
| Product Markdown | `report.go` | Policy Chain numbered list |
| Catalog Markdown | `catalog.go` via `BuildTopologyData` | Count + `formatPolicyChain` |
| Canvas / HTML | `topology_data.go` → templates | `PolicyNames` in embedded JSON |

Fixing at the loader/build layer is sufficient; templates only display embedded JSON.

**Existing precedent:** `authTypeFromPoliciesConfig` in `auth.go` already parses `policies_config` entries with `name`, `enabled`, and `configuration`, preferring enabled `default_credentials` policies. `internal/seed/policies.go` sets `Enabled: true` on seeded chains and always appends `apicast`.

**Fixture vs live format gap:**

| Source | `policies.json` shape | `enabled` field | Includes `apicast` |
|--------|----------------------|-----------------|-------------------|
| `export-minimal` | `{ "policies": [{ "policy": {...} }] }` | absent | no |
| Live export sample | `{ "policies_config": [...] }` | present | yes (terminal entry) |

Current `export-live` seed products have no `enabled: false` entries; large tenant exports are expected to include disabled policies (e.g. `camel`, `headers`) per user report.

## Affected Areas

- `internal/visualize/loader.go` — `parsePolicies`, `policyNamesFromConfig`; unify parsing and enabled filtering
- `internal/visualize/loader_test.go` — fixture backward compat + new `policies_config` / disabled-policy cases
- `internal/visualize/canvas_test.go` — `TestPolicyNamesFromProxyFile` may need enabled-filter expectations
- `internal/visualize/report_test.go`, `catalog_test.go` — assert enabled-only chains when fixtures extended
- `openspec/specs/visualize-report/spec.md`, `visualize-catalog/spec.md`, `visualize-html-dashboard/spec.md` — delta specs for enabled-only display (proposal/spec phase)

No changes needed in canvas/HTML templates, `report.go`, `catalog.go`, or `topology_data.go` if loader output is correct.

## Format Parsing Gap

`parsePolicies` does **not** handle root `policies_config` in `policies.json`. Today this is masked by proxy fallback, but:

- If `proxy.json` is missing or lacks `policies_config`, live `policies.json` yields **zero policies** (stale `report-live/` artifacts show this symptom).
- `policies.json` should be authoritative when present; parsing both shapes in one code path avoids silent fallback dependency.

## Open Question: Exclude `apicast`?

`apicast` is the built-in terminal gateway policy. Live exports always include it as the last entry (`enabled: true`). Fixtures omit it.

| Option | Rationale |
|--------|-----------|
| **Show `apicast`** | Faithful to raw export order; matches API payload |
| **Hide `apicast`** | Matches 3scale Admin UI mental model (user policies only); avoids inflating counts; aligns fixture/live display |

**Recommendation for proposal:** exclude `apicast` from visualization policy chains (count + names). It is infrastructure, not a user-configured policy. If product owners prefer raw fidelity, make exclusion configurable later (`--show-apicast`).

## Default When `enabled` Is Absent

Go `json.Unmarshal` into `bool` yields `false` when the field is missing — **wrong** for backward compat with fixture exports that omit `enabled`.

**Recommend:** treat absent `enabled` as **true** (same effective behavior as today for fixtures). Implementation options:

- `*bool` with `nil` → enabled, or
- custom unmarshaler / post-pass using `json.RawMessage` per entry

Do **not** change auth inference semantics in `authTypeFromPoliciesConfig` as part of this change unless explicitly scoped.

## Approaches

| Approach | Pros | Cons | Complexity |
|----------|------|------|------------|
| **A. Filter in loader (unified parser)** | Single source of truth; all outputs fixed automatically; matches existing `BuildTopologyData` sharing pattern | Requires parsing both JSON shapes + enabled default logic | Low–Medium |
| **B. Filter in each renderer** | Loader stays “raw” | Duplication across report, catalog, topology; easy to miss a consumer; `Policy` has no `Enabled` field today | Medium |
| **C. Filter only in `BuildTopologyData`** | Small diff | **Incomplete** — product Markdown pages bypass topology builder | Low (rejected) |

### 1. Filter in loader (unified parser) — **recommended**

Introduce shared helper (e.g. `policiesFromConfigJSON`) used by both `parsePolicies` and `policyNamesFromConfig`:

- Accept wrapped `policies[].policy.name` **or** root / nested `policies_config[].name`
- Skip entries where `enabled == false` (explicit only)
- Default missing `enabled` to true
- Optionally skip `name == "apicast"` (proposal decision)

- **Pros:** One fix; canvas, HTML, catalog, and product pages stay in sync; aligns with `openspec/config.yaml` design rule to reuse shared topology payload
- **Cons:** Test matrix grows (both formats, enabled variants, apicast exclusion)
- **Effort:** Low–Medium (~80–150 LOC including tests)

### 2. Filter in each renderer

Add `enabledOnlyPolicyNames(p.Policies)` in `report.go`, `topology_data.go`, etc.

- **Pros:** Minimal loader change
- **Cons:** Violates DRY; future consumers may show all policies again; loader still mis-parses `policies_config` in `policies.json` without proxy fallback
- **Effort:** Medium

## Recommendation

**Approach A — filter in loader with a unified parser.**

1. Extend `parsePolicies` to try `policies_config` (root) after wrapped `policies` format.
2. Refactor `policyNamesFromConfig` to share the same enabled-filter logic.
3. Default absent `enabled` to **true**.
4. Exclude `apicast` from displayed chains (confirm in proposal).
5. Add fixture snippets under `internal/visualize/testdata/` with `enabled: false` entries (no live tenant paths in repo).

Downstream files unchanged. Existing `export-minimal` tests should pass unchanged (no `enabled` field → all listed; no `apicast` in fixtures).

## Risks

- **Backward compat:** Treating missing `enabled` as false (naive bool unmarshal) would hide all fixture policies — must use explicit default-true logic.
- **Auth side effects:** Loader change must not alter `inferAuthTypeFromProxy` / `authTypeFromPoliciesConfig` behavior (those read raw `PoliciesConfig` before display filtering).
- **Dual-format maintenance:** Both Admin wrapped and `policies_config` shapes must stay supported until export normalizes format.
- **`apicast` decision:** Excluding it changes live-export display (3 policies → 2 for seed products); document in spec scenarios.
- **Review budget:** Estimated within 400-line PR budget if scoped to loader + tests only.

## Ready for Proposal

**Yes.** Proposal should confirm `apicast` exclusion, document enabled-default semantics, and define spec scenarios for: fixture backward compat, live `policies_config` with mixed enabled flags, and catalog/canvas/HTML/product-page consistency.
