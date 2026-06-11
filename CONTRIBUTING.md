# Contributing to 3scaleextract

## Language

**English only** for README, docs, issues, PR descriptions, code comments, and CLI help text.

Program policy: [rhcl-ai AGENTS.md — Language policy](https://github.com/Everything-is-Code/rhcl-ai/blob/main/AGENTS.md#language-policy).

## Pull requests

Use [.github/pull_request_template.md](.github/pull_request_template.md). Link related issues (`EXT-*` or GitHub `#number`).

## Development

```bash
go test ./...
go build -o bin/threescale-export ./cmd/threescale-export
```

When adding or updating tests, align with [docs/TEST_CASES.md](docs/TEST_CASES.md) and update the catalog if behavior or coverage changes.

See [README.md](README.md) for export prerequisites and [docs/SEED.md](docs/SEED.md) for lab fixtures.
