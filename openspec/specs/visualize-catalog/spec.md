# visualize-catalog

## Requirements

### Requirement: Product catalog Markdown

The visualizer SHALL write `products-catalog.md` in the report output directory for every successful run.

The catalog SHALL contain one row per API product with columns: Product (system name, linked to `products/{name}.md`), Category, Auth, Backends (count), Applications (count when export included applications), Policies (count), Policy names (chain separated by ` → `).

The **Policies** count and **Policy names** column SHALL reflect **visible policies only**: enabled entries (missing `enabled` treated as enabled) excluding `apicast`. Disabled policies (`enabled: false`) SHALL NOT be counted or named.

#### Scenario: Catalog from minimal fixture

- **GIVEN** a valid export at `internal/visualize/testdata/export-minimal`
- **WHEN** `threescale-visualize export-minimal -o /tmp/report` runs
- **THEN** `/tmp/report/products-catalog.md` exists
- **AND** it contains `seed_alpha` with auth and policy name `cors`
- **AND** the Policies column for `seed_alpha` is `1`

#### Scenario: Index links to catalog

- **GIVEN** a generated report
- **WHEN** a reader opens `index.md`
- **THEN** a navigation link to `products-catalog.md` is present

#### Scenario: Disabled policy excluded from count and chain

- **GIVEN** a product with visible policies `edge_limit` and `url_rewriting` and disabled policy `camel` in `policies_config`
- **WHEN** visualize runs
- **THEN** the catalog row shows Policies count `2`
- **AND** Policy names `edge_limit → url_rewriting`
- **AND** neither `camel` nor `apicast` appear
