# threescale-export

CLI to export the full configuration of a **Red Hat 3scale API Management** tenant (products, backends, plans, auth, policies, and applications).

**Language:** Documentation and contributions are **English only**. Program policy: [rhcl-ai AGENTS.md — Language policy](https://github.com/Everything-is-Code/rhcl-ai/blob/main/AGENTS.md#language-policy).

## Run the binary

Download binaries from [Releases](https://github.com/Everything-is-Code/3scaleextract/releases) (Linux x86_64). No Go toolchain or local build required.

### Prerequisites

| Requirement | Description |
|-------------|-------------|
| **Binary** | `threescale-export` ([Releases](https://github.com/Everything-is-Code/3scaleextract/releases)) |
| **Docker** or **Podman** | Container runtime (**Podman is not required**; Docker alone is enough) |
| **3scale toolbox image** | `registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16` |
| **Red Hat Registry** | [Registry Service Account](https://access.redhat.com/terms-based-registry) |
| **Admin API token** | Personal Access Token from the 3scale Admin Portal |

Red Hat documentation: [Installing the toolbox container image](https://docs.redhat.com/en/documentation/red_hat_3scale_api_management/2.16/html/operating_red_hat_3scale_api_management/the-threescale-toolbox#installing_the_toolbox_container_image)

### Runtime dependencies

**Toolbox image (Red Hat)** — with Docker:

```bash
docker login registry.redhat.io
docker pull registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16
docker run --rm registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16 3scale help
```

With **Podman**, replace `docker` with `podman`.

**Container runtime** — the binary auto-detects Docker or Podman on `PATH` (Docker first). To force one:

```bash
export THREESCALE_TOOLBOX_RUNTIME=docker   # or podman
```

### Configuration

Required variables:

```bash
export THREESCALE_ADMIN_URL="https://tenant-admin.example.com"
export THREESCALE_ACCESS_TOKEN="your-personal-access-token"
```

Optional variables:

```bash
export THREESCALE_OUTPUT_DIR="./export"   # equivalent to --output

export THREESCALE_TOOLBOX_IMAGE="registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16"
export THREESCALE_TOOLBOX_RUNTIME="docker"
export THREESCALE_TOOLBOX_TLS_CERT="/path/to/ca.pem"   # self-signed TLS on Admin Portal
```

You can also pass credentials via flags (`--admin-url`, `--token`).

### Export

```bash
chmod +x threescale-export

./threescale-export \
  --output ./export \
  --include-applications \
  --redact-secrets
```

The hybrid export combines:

- **Admin API** — backends, applications, accounts, supplementary JSON per product.
- **Toolbox (Red Hat container)** — product YAML (`3scale product export`).

| Flag | Description |
|------|-------------|
| `--admin-url` | 3scale Admin Portal URL |
| `--token` | Personal Access Token |
| `--output` | Output directory (default `./export`) |
| `--include-applications` | Include applications and accounts (paginated) |
| `--redact-secrets` | Mask API keys and OIDC secrets |
| `--per-page` | Admin API page size (max 500) |
| `--concurrency` | Concurrent requests (default 4) |
| `--insecure` | Skip TLS verification on Admin API |
| `--toolbox-image` | Toolbox image (default Red Hat 2.16) |
| `--toolbox-runtime` | `docker` or `podman` (auto-detect if empty) |
| `--toolbox-tls-cert` | CA certificate mounted in the toolbox container |

Self-signed TLS (Admin Portal or toolbox):

```bash
./threescale-export \
  --toolbox-tls-cert ./ca.pem \
  --insecure \
  --output ./export
```

### Visualize the export

Optional: generate a Markdown report from the export directory (no Admin API or containers):

```bash
chmod +x threescale-visualize
./threescale-visualize ./export -o ./report
```

See **[docs/VISUALIZE.md](docs/VISUALIZE.md)** for report layout.

---

## Build from source

For developers changing the project.

### Requirements

- Go 1.22+

### Compile

```bash
git clone https://github.com/Everything-is-Code/3scaleextract.git
cd 3scaleextract

go build -o bin/threescale-export ./cmd/threescale-export
go build -o bin/threescale-seed ./cmd/threescale-seed
go build -o bin/threescale-visualize ./cmd/threescale-visualize
```

### Run local binaries

```bash
bin/threescale-export --output ./export --include-applications --redact-secrets
bin/threescale-visualize ./export -o ./report
```

### Tests

```bash
go test ./...
go test -tags=integration ./internal/export/...   # live tenant (THREESCALE_*)
```

See [docs/TEST_CASES.md](docs/TEST_CASES.md) for the full catalog of user-journey test cases and automation status.

#### Integration CI

A separate GitHub Actions workflow (`.github/workflows/integration.yml`) runs the live export integration test on demand or on a weekly schedule. It does **not** run on pull requests; PR CI stays offline-only.

**Required repository secrets**

| Secret | Description |
|--------|-------------|
| `THREESCALE_ADMIN_URL` | 3scale Admin Portal base URL |
| `THREESCALE_ACCESS_TOKEN` | Personal Access Token |
| `THREESCALE_OUTPUT_DIR` | Writable output path on the runner (e.g. `/tmp/3scale-export`) |

**Optional repository secrets**

| Secret | Description |
|--------|-------------|
| `THREESCALE_TOOLBOX_IMAGE` | Override default Red Hat toolbox image |
| `THREESCALE_TOOLBOX_TLS_CERT` | CA/cert file path for toolbox TLS (lab tenants) |
| `THREESCALE_INSECURE_TLS` | Set to `true` to skip TLS verification for Admin API (lab only) |

**Run the workflow**

1. Configure the secrets under **Settings → Secrets and variables → Actions**.
2. Open **Actions → Integration → Run workflow**.

The workflow also runs every **Monday at 06:00 UTC**. If required secrets are missing, the job exits successfully with a skip message.

CI uses the Docker toolbox (`THREESCALE_TOOLBOX_RUNTIME=docker`). For local runs you may set `THREESCALE_TOOLBOX_BINARY=3scale` instead. Pulling `registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16` on GitHub-hosted runners may require Red Hat registry credentials; see [Installing the toolbox container image](https://docs.redhat.com/en/documentation/red_hat_3scale_api_management/2.16/html/operating_red_hat_3scale_api_management/the-threescale-toolbox#installing_the_toolbox_container_image).

---

## Export layout

```
export/
├── manifest.json
├── products/
│   ├── {system_name}.yaml          # toolbox (Red Hat image)
│   └── {system_name}/
│       ├── proxy.json
│       ├── policies.json
│       ├── oidc_configuration.json
│       ├── application_plans.json
│       ├── backend_usages.json
│       └── metrics.json
├── backends/{system_name}.json
├── policies/catalog.json
├── applications/page-{n}.json        # with --include-applications
└── accounts/{id}.json
```

## Limitations

- Does not export billing, Developer Portal, or analytics
- Requires access to `registry.redhat.io` and a container runtime
- Product YAML export depends on the official Red Hat toolbox image

## Release (CI)

Releases are published when a semver tag is pushed:

```bash
git tag v0.1.1
git push origin v0.1.1
```

GitHub Actions runs tests, builds `threescale-export`, `threescale-seed`, and `threescale-visualize` for Linux amd64, and publishes `.tar.gz` artifacts with checksums on [Releases](https://github.com/Everything-is-Code/3scaleextract/releases).

## Optional lab tools

| Tool | Description |
|------|-------------|
| **[docs/SEED.md](docs/SEED.md)** | Load fixtures into a lab tenant to validate export |
| **[docs/VISUALIZE.md](docs/VISUALIZE.md)** | Generate a Markdown report from an export |
| **[testdata/README.md](testdata/README.md)** | Offline export fixture tarball (`export-minimal-1.0.tar.gz`) for tests and GateForge import |

## License

Apache License 2.0 — see [LICENSE](LICENSE).
