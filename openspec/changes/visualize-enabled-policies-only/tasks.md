# Tasks: Visualize enabled policies only

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 120–200 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: No
400-line budget risk: Low

### Suggested Work Units

1. **Loader filter** — unified parser + visibility rules in `loader.go`
2. **Tests + docs** — loader/canvas/report/catalog tests, TC-VIZ update

---

## Phase 1: Policy visibility in loader

- [ ] 1.1 Add `visiblePoliciesFromConfig` helper (`*bool` enabled, skip `apicast`)
- [ ] 1.2 Refactor `parsePolicies` to handle root `policies_config` and wrapped `policies`
- [ ] 1.3 Refactor `policyNamesFromConfig` to share visibility filter
- [ ] 1.4 Confirm auth path unchanged (`auth.go` reads raw config)

## Phase 2: Tests

- [ ] 2.1 Unit tests: disabled omitted, apicast omitted, absent `enabled` → visible
- [ ] 2.2 Update `TestPolicyNamesFromProxyFile` and export-minimal loader tests
- [ ] 2.3 Report/catalog integration assertions for filtered chains
- [ ] 2.4 `go test ./...`

## Phase 3: Documentation

- [ ] 3.1 Update `docs/TEST_CASES.md` (TC-VIZ enabled-only policies)
- [ ] 3.2 Mark tasks complete; grep diff for customer terms

## Privacy checklist

- [ ] P.1 No customer/tenant names in code, tests, commits, or SDD examples
- [ ] P.2 Policy test JSON uses fixture names only (`seed_*`, generic policy names)
- [ ] P.3 Grep committed diff before PR
