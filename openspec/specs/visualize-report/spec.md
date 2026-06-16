# visualize-report

## Requirements

### Requirement: Report bundle layout

The visualizer SHALL write a multi-file Markdown report under the output directory including `index.md`, `backends.md`, `products/{system_name}.md`, optional `applications.md`, and **`products-catalog.md`**.

`index.md` SHALL link to the product catalog and, when `--html` is used, to `topology.html`.

#### Scenario: Report layout after change

- **GIVEN** a valid export
- **WHEN** visualize runs with `-o ./report`
- **THEN** `./report/products-catalog.md` is created alongside existing report files
