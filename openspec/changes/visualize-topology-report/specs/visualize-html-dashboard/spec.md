# Delta for visualize-html-dashboard

## ADDED Requirements

### Requirement: Self-contained topology HTML

When `--html` is set, the visualizer SHALL write a single self-contained HTML file (default: `{output}/topology.html`) embeds topology JSON from the same builder used by the Cursor canvas.

The HTML page SHALL include: summary statistics, products-by-domain chart, shared-backends chart, interactive product→backend graph with product selector, sortable/filterable product table (auth, backend count, app count, policy count, policy names), and shared-backend detail section.

The HTML SHALL NOT require Cursor IDE or a build step; it SHALL open in a standard browser.

#### Scenario: HTML generation from fixture

- **GIVEN** `internal/visualize/testdata/export-minimal`
- **WHEN** `threescale-visualize export-minimal -o /tmp/report --html /tmp/report/topology.html` runs
- **THEN** `topology.html` exists and contains valid embedded JSON with `seed_alpha`
- **AND** the file contains no unresolved template placeholders

#### Scenario: HTML without network for data

- **GIVEN** a generated `topology.html`
- **WHEN** opened offline
- **THEN** product table and graph render from embedded data (charts MAY require CDN unless cached)

### Requirement: No customer data in repository artifacts

Committed demo HTML and tests SHALL use only `export-minimal` fixture data (`seed_alpha`, `example.com`). Generated HTML from production exports SHALL NOT be committed.

#### Scenario: Demo artifact source

- **GIVEN** `docs/examples/topology-demo.html`
- **WHEN** regenerated from export-minimal
- **THEN** it contains fixture product names only
