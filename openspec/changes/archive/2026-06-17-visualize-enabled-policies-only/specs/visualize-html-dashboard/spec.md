# Delta for visualize-html-dashboard

## MODIFIED Requirements

### Requirement: Self-contained topology HTML

When `--html` is set, the visualizer SHALL write a single self-contained HTML file (default: `{output}/topology.html`) embeds topology JSON from the same builder used by the Cursor canvas.

The HTML page SHALL include: summary statistics, products-by-domain chart, shared-backends chart, interactive product→backend graph with product selector, sortable/filterable product table (auth, backend count, app count, policy count, policy names), and shared-backend detail section.

Embedded product **policy count** and **policy names** in the topology JSON SHALL match visible policies only (enabled, excluding `apicast`), consistent with Markdown report and canvas output.

The HTML SHALL NOT require Cursor IDE or a build step; it SHALL open in a standard browser.

(Previously: embedded policy data included all configured policies regardless of `enabled` or `apicast`.)

#### Scenario: HTML generation from fixture

- **GIVEN** `internal/visualize/testdata/export-minimal`
- **WHEN** `threescale-visualize export-minimal -o /tmp/report --html` runs
- **THEN** `topology.html` exists and contains valid embedded JSON with `seed_alpha`
- **AND** the file contains no unresolved template placeholders
- **AND** embedded policy names for `seed_alpha` include `cors` and exclude `apicast`

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

## ADDED Requirements

### Requirement: Canvas topology policy parity

When `--canvas` is set, embedded topology JSON policy count and names SHALL use the same visible-policy rules as HTML and Markdown (enabled only, no `apicast`).

#### Scenario: Canvas excludes disabled and apicast

- **GIVEN** a product with `edge_limit` (enabled), `url_rewriting` (enabled), `camel` (disabled), and `apicast` (enabled) in export data
- **WHEN** `threescale-visualize` writes a canvas file
- **THEN** embedded JSON policy names for that product are `edge_limit` and `url_rewriting` only
