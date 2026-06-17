# Delta for visualize-report

## ADDED Requirements

### Requirement: Visible policy chain on product pages

Each `products/{system_name}.md` page SHALL include a Policy Chain section listing **only visible policies**: entries that are enabled (or have no `enabled` field) and are not the built-in `apicast` policy.

Disabled policies (`enabled: false`) SHALL NOT appear in the numbered list.

#### Scenario: Fixture product without enabled field

- **GIVEN** a valid export at `internal/visualize/testdata/export-minimal`
- **WHEN** visualize runs with `-o /tmp/report`
- **THEN** `products/seed_alpha.md` lists policy `cors` in the Policy Chain section
- **AND** `apicast` does not appear

#### Scenario: Mixed enabled flags in policies_config

- **GIVEN** a product whose `policies.json` contains `policies_config` with `headers` (`enabled: true`) and `camel` (`enabled: false`)
- **WHEN** visualize runs
- **THEN** the product page Policy Chain includes `headers`
- **AND** does not include `camel` or `apicast`
