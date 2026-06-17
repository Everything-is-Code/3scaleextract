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

- [x] 1.1 Add `visiblePoliciesFromConfig` helper (`*bool` enabled, skip `apicast`)
- [x] 1.2 Refactor `parsePolicies` to handle root `policies_config` and wrapped `policies`
- [x] 1.3 Refactor `policyNamesFromConfig` to share visibility filter
- [x] 1.4 Confirm auth path unchanged (`auth.go` reads raw config)

## Phase 2: Tests

- [x] 2.1 Unit tests: disabled omitted, apicast omitted, absent `enabled` → visible
- [x] 2.2 Update `TestPolicyNamesFromProxyFile` and export-minimal loader tests
- [x] 2.3 Report/catalog integration assertions for filtered chains
- [x] 2.4 `go test ./...`

## Phase 3: Documentation

- [x] 3.1 Update `docs/TEST_CASES.md` (TC-VIZ enabled-only policies)
- [x] 3.2 Mark tasks complete; grep diff for customer terms

## Privacy checklist

- [x] P.1 No customer/tenant names in code, tests, commits, or SDD examples
- [x] P.2 Policy test JSON uses fixture names only (`seed_*`, generic policy names)
- [ ] P.3 Grep committed diff before PR
