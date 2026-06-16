# Delta for visualize-catalog

## ADDED Requirements

### Requirement: Product catalog Markdown

The visualizer SHALL write `products-catalog.md` in the report output directory for every successful run.

The catalog SHALL contain one row per API product with columns: Product (system name, linked to `products/{name}.md`), Category, Auth, Backends (count), Applications (count when export included applications), Policies (count), Policy names (chain separated by ` → `).

#### Scenario: Catalog from minimal fixture

- **GIVEN** a valid export at `internal/visualize/testdata/export-minimal`
- **WHEN** `threescale-visualize export-minimal -o /tmp/report` runs
- **THEN** `/tmp/report/products-catalog.md` exists
- **AND** it contains `seed_alpha` with auth and policy name `cors`

#### Scenario: Index links to catalog

- **GIVEN** a generated report
- **WHEN** a reader opens `index.md`
- **THEN** a navigation link to `products-catalog.md` is present
