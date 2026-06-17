# Proposal: Visualize enabled policies only

## Intent

3scale exports include policies with `enabled: false` that remain in the chain but are inactive. The visualizer lists every configured policy, inflating counts and misleading migration reviews. Show **only active user policies** consistently in Markdown, canvas, and HTML.

**Product decision (confirmed):** exclude the built-in `apicast` policy from display (infrastructure, not user-configured).

## Scope

### In Scope

- Unified policy parser in `internal/visualize/loader.go` for both export shapes:
  - `{ "policies": [{ "policy": { "name" } }] }` (fixtures)
  - `{ "policies_config": [{ "name", "enabled", ... }] }` (live exports)
- Filter rules: skip `enabled: false`; treat missing `enabled` as **true**; skip `name == "apicast"`
- All outputs via `product.Policies` / `BuildTopologyData`: product pages, catalog, canvas, HTML
- Fixture test data with mixed enabled flags (`export-minimal` extension or inline test JSON)
- Delta specs for `visualize-report`, `visualize-catalog`, `visualize-html-dashboard`
- Update `docs/TEST_CASES.md` (TC-VIZ policy scenarios)

### Out of Scope

- Changing auth inference (`authTypeFromPoliciesConfig` reads raw config unchanged)
- `--show-apicast` CLI flag (future if needed)
- Export-side normalization of `policies.json` format
- Policy configuration details beyond name/count/chain

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `visualize-report`: Policy Chain section lists enabled user policies only (no `apicast`)
- `visualize-catalog`: Policy count and names reflect enabled user policies only
- `visualize-html-dashboard`: Embedded policy count/names match same filtered chain

## Approach

Single helper (e.g. `visiblePoliciesFromJSON`) shared by `parsePolicies` and `policyNamesFromConfig`:

1. Parse wrapped `policies` or root `policies_config`
2. Keep entry if `enabled` is absent or true; drop if explicitly false
3. Drop `apicast` after filtering
4. Preserve export order among remaining entries

No renderer/template changes if loader output is correct.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/visualize/loader.go` | Modified | Unified parse + filter |
| `internal/visualize/loader_test.go` | Modified | Format + enabled + apicast cases |
| `internal/visualize/canvas_test.go` | Modified | Proxy fallback expectations |
| `internal/visualize/report_test.go` | Modified | Policy chain assertions |
| `internal/visualize/catalog_test.go` | Modified | Count/chain assertions |
| `openspec/specs/visualize-*/spec.md` | Delta | Enabled-only requirements |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Missing `enabled` treated as false | High | Use `*bool` or explicit default-true |
| Auth regression | Low | Do not filter raw `PoliciesConfig` used by auth |
| Live count drop (apicast removed) | Med | Document in spec; expected behavior |
| Dual-format drift | Med | One shared parser; table-driven tests |

## Rollback Plan

Revert loader filter logic. Renderers unchanged; behavior returns to listing all configured policies including disabled and `apicast`.

## Dependencies

None.

## Success Criteria

- [ ] Fixture export without `enabled` field: same visible policies as today (minus any `apicast`)
- [ ] `policies_config` with `enabled: false` entries: disabled names absent from report, catalog, canvas, HTML
- [ ] `apicast` never appears in policy chain, count, or table column
- [ ] `go test ./...` passes; coverage ≥ 80%
- [ ] No customer names in committed tests or SDD artifacts
