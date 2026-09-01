# threescale-export

CLI to export the full configuration of a **Red Hat 3scale API Management** tenant (products, backends, plans, auth, policies, and applications).

**Language:** Documentation and contributions are **English only**. Program policy: [rhcl-ai AGENTS.md — Language policy](https://github.com/Everything-is-Code/rhcl-ai/blob/main/AGENTS.md#language-policy).

---

## Quick start (read this first)

Follow these steps **in order**. Do not skip the verification step after each phase.

| Step | What | Verify before continuing |
|------|------|--------------------------|
| 1 | Download the Linux binary | `file threescale-export` shows `ELF 64-bit` |
| 2 | Install Docker **or** Podman | `docker version` or `podman version` works |
| 3 | Log in and pull the Red Hat toolbox image | `docker run --rm registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16 3scale help` prints help |
| 4 | Set credentials (Admin URL + PAT) | `echo "$THREESCALE_ADMIN_URL"` and `echo "$THREESCALE_ACCESS_TOKEN"` are non-empty |
| 5 | Run export with `--output` | stderr ends with `Export complete: … → ./export` and `export/manifest.json` exists |

**Platform:** pre-built binaries are **Linux x86_64 only**. There is no Windows or macOS release. Run on a Linux VM, WSL2, or build from source (see [Build from source](#build-from-source)).

---

## 1. Download and install the binary

1. Open [Releases](https://github.com/Everything-is-Code/3scaleextract/releases) and download **`threescale-export-vX.Y.Z-linux-amd64.tar.gz`** (latest: check the tag on the release page).
2. Extract and make it executable:

```bash
tar -xzf threescale-export-v*.tar.gz
chmod +x threescale-export
./threescale-export --version
```

You should see a version string (for example `v0.4.4`). If you get `Permission denied`, run `chmod +x threescale-export` again.

> **Do not** run the `.tar.gz` file directly. Extract it first.
>
> **Do not** rename the binary to `3scaleextract` or `3scale-export` unless you update your commands accordingly. The command name is **`threescale-export`**.

---

## 2. One-time runtime setup

### 2.1 Container runtime

You need **Docker** or **Podman** running. Podman is optional; Docker alone is enough.

The binary auto-detects `docker` or `podman` on `PATH` (Docker first). To force one:

```bash
export THREESCALE_TOOLBOX_RUNTIME=docker   # or podman
```

### 2.2 Red Hat toolbox image (required)

Product YAML export uses the official Red Hat toolbox container. You must log in to the Red Hat registry and pull the image **once** before your first export.

| Requirement | Description |
|-------------|-------------|
| **Red Hat Registry account** | [Registry Service Account](https://access.redhat.com/terms-based-registry) |
| **Toolbox image** | `registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16` |

```bash
docker login registry.redhat.io
docker pull registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16
docker run --rm registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16 3scale help
```

With **Podman**, replace `docker` with `podman`.

Red Hat documentation: [Installing the toolbox container image](https://docs.redhat.com/en/documentation/red_hat_3scale_api_management/2.16/html/operating_red_hat_3scale_api_management/the-threescale-toolbox#installing_the_toolbox_container_image)

**If this step fails**, export will fail later with errors about docker/podman or toolbox. Fix the container setup before retrying export.

---

## 3. Credentials

### 3.1 Admin URL — use the Admin Portal base URL

Set **`THREESCALE_ADMIN_URL`** (or `--admin-url`) to the **Admin Portal** URL — the same host you open in the browser to manage the tenant.

| Correct | Wrong (will fail or export the wrong tenant) |
|---------|---------------------------------------------|
| `https://mytenant-admin.apps.cluster.example.com` | `https://mytenant-admin.../admin/api` (do **not** add `/admin/api`) |
| `https://mytenant-admin.example.com` | APIcast / gateway URL (`*.apicast.io`, `*.3scale.net`) |
| | Developer Portal URL |
| | Provider account URL from a different tenant |

Rules:

- Must start with `http://` or `https://`
- No trailing slash required (it is stripped automatically)
- No path suffix — the tool appends `/admin/api/...` itself

### 3.2 Access token — Personal Access Token (PAT)

Create a **Personal Access Token** in the 3scale **Admin Portal** (Account Settings → Personal Access Tokens).

| Export scope | PAT requirement |
|--------------|-----------------|
| Default export (config + product YAML) | Valid PAT with Admin API access |
| `--include-metrics` or `metrics` subcommand | **Enterprise** tier **and** PAT with **Analytics** scope |

```bash
export THREESCALE_ADMIN_URL="https://mytenant-admin.apps.cluster.example.com"
export THREESCALE_ACCESS_TOKEN="paste-your-pat-here"
```

Optional — output directory via environment variable:

```bash
export THREESCALE_OUTPUT_DIR="./export"
```

You can also pass credentials via flags: `--admin-url`, `--token`.

> **Security:** do not commit tokens. Do not paste tokens into chat or tickets. Prefer env vars over flags in shared screen sessions.

---

## 4. Run export

### 4.1 Full export (applications + metrics + redaction)

Only use this after the minimal export works. Metrics need Enterprise + Analytics PAT scope.

```bash
./threescale-export \
  --output ./export \
  --include-applications \
  --include-metrics \
  --metrics-since 2026-01-01 \
  --metrics-until 2026-01-31 \
  --redact-secrets
```

The hybrid export combines:

- **Admin API** — backends, applications, accounts, supplementary JSON per product.
- **Toolbox (Red Hat container)** — product YAML (`3scale product export`).

### 4.3 Lab tenant with self-signed TLS

```bash
./threescale-export \
  --output ./export \
  --insecure
```

For production tenants with a proper CA, omit `--insecure`. You can mount a custom CA with `--toolbox-tls-cert` instead.

---

## 5. How to know it worked

A successful run prints a summary on **stderr**, for example:

```text
Export complete: 12 products, 3 backends → ./export
```

Check the output directory:

```bash
test -f ./export/manifest.json && echo "OK: manifest exists"
ls ./export/products/*.yaml | head
```

Minimum expected layout:

```text
export/
├── manifest.json          ← must exist
├── products/*.yaml        ← one per API product (toolbox)
├── backends/*.json
└── policies/catalog.json
```

Open `manifest.json` and confirm `product_count` matches the number of `products/*.yaml` files.

| Symptom | Likely cause |
|---------|----------------|
| `admin URL is required` / `access token is required` | Env vars not set in **this** shell session, or typo in variable names |
| `output directory is required` | Missing `--output` and `THREESCALE_OUTPUT_DIR` |
| `admin URL must start with http:// or https://` | URL missing scheme |
| `docker or podman is required` | Container runtime not installed or not on `PATH` |
| `3scale toolbox product export failed` | Toolbox image not pulled, registry login expired, or wrong Admin URL |
| `401` / `403` on Admin API | Invalid or expired PAT, or wrong tenant URL |
| Metrics export fails | Non-Enterprise tenant, or PAT without Analytics scope |
| `cleartext secret in …` with `--redact-secrets` | Redaction could not mask all secrets; inspect the named file |

Use `--verbose` to see toolbox invocations (credentials are redacted in logs). Use `--quiet` in scripts.

---

## Reference

### Configuration (environment variables)

Required:

```bash
export THREESCALE_ADMIN_URL="https://tenant-admin.example.com"
export THREESCALE_ACCESS_TOKEN="your-personal-access-token"
```

Optional:

```bash
export THREESCALE_OUTPUT_DIR="./export"   # alternative to --output

export THREESCALE_TOOLBOX_IMAGE="registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16"
export THREESCALE_TOOLBOX_RUNTIME="docker"
export THREESCALE_TOOLBOX_TLS_CERT="/path/to/ca.pem"   # self-signed TLS on Admin Portal
```

### Flags

| Flag | Description |
|------|-------------|
| `--admin-url` | 3scale Admin Portal URL |
| `--token` | Personal Access Token |
| `--output` | Output directory (**required** unless `THREESCALE_OUTPUT_DIR` is set) |
| `--include-applications` | Include applications and accounts (paginated) |
| `--include-metrics` | Include Analytics API hit traffic under `stats/` (Enterprise + PAT Analytics scope) |
| `--metrics-since` | Metrics window start (`YYYY-MM-DD` UTC; default 30 days before `--metrics-until`) |
| `--metrics-until` | Metrics window end (`YYYY-MM-DD` UTC; default today UTC) |
| `--metrics-granularity` | `day`, `hour`, or `month` (default `day`) |
| `--metrics-metric` | Metric name for usage queries (default `hits`) |
| `--redact-secrets` | Opt-in: mask sensitive keys in JSON/YAML artifacts (see [Redaction](#redaction) below) |
| `--per-page` | Admin API page size (max 500) |
| `--concurrency` | Concurrent requests (default 4) |
| `--insecure` | Skip TLS verification on Admin API, Analytics API, and toolbox (`-k`) |
| `--toolbox-image` | Toolbox image (default Red Hat 2.16) |
| `--toolbox-runtime` | `docker` or `podman` (auto-detect if empty) |
| `--toolbox-tls-cert` | CA certificate mounted in the toolbox container |
| `--quiet` | Suppress progress output on stderr |
| `--verbose` | Show detailed progress (e.g. toolbox invocations; credentials redacted) |

Progress is written to **stderr** by default: phase banners, `[i/n]` per API product, live warnings for skipped sidecars, and a completion summary.

Self-signed TLS (Admin Portal or toolbox):

`--insecure` passes `-k` to the 3scale toolbox container/native binary so product YAML export also skips certificate verification. Alternatively, mount a custom CA with `--toolbox-tls-cert` without disabling verification.

```bash
./threescale-export \
  --toolbox-tls-cert ./ca.pem \
  --insecure \
  --output ./export
```

### Redaction

`--redact-secrets` is **opt-in** (default off). When set, every `.json`, `.yaml`, and `.yml` file under the export root is processed before `manifest.json` is written.

**Fully redacted keys** (value becomes `***REDACTED***`):

`access_token`, `api_key`, `app_id`, `app_key`, `client_id`, `client_secret`, `provider_key`, `provider_verification_key`, `secret`, `user_key`

**Issuer URL stripping** (`issuer_endpoint`, `oidc_issuer_endpoint`): embedded credentials are removed (`https://user:pass@host/path` → `https://host/path`); host, path, and query stay visible.

**Preserved auth-mode flags** (not secrets): `auth_user_key`, `auth_app_id`, `auth_app_key`

After redaction, a cleartext scan runs over the same artifacts. If any sensitive value or issuer userinfo remains, **export fails** with a path-qualified error.

### Hit metrics (Analytics API)

Optional hit traffic can be exported with `--include-metrics` on a full export, or with the standalone subcommand:

```bash
./threescale-export metrics \
  --output ./export \
  --metrics-since 2026-01-01 \
  --metrics-until 2026-01-31
```

Requires **Enterprise** tier and a Personal Access Token with **Analytics** scope. On failure the export exits non-zero (no partial stats when the flag is set).

### Visualize the export

Optional: generate a Markdown report from the export directory (no Admin API or containers):

```bash
chmod +x threescale-visualize
./threescale-visualize ./export -o ./report
```

See **[docs/VISUALIZE.md](docs/VISUALIZE.md)** for report layout, product catalog, optional HTML dashboard, Cursor topology canvas, and policy visibility rules.

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
bin/threescale-visualize ./export -o ./report --html --canvas ./topology.canvas.tsx
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
├── accounts/{id}.json
└── stats/                              # with --include-metrics or `metrics` subcommand
    ├── query.json
    └── products/{system_name}/hits.json
```

## Limitations

- Does not export billing or Developer Portal content
- Analytics hit metrics require Enterprise tier and PAT Analytics scope (`--include-metrics` or `metrics` subcommand)
- Requires access to `registry.redhat.io` and a container runtime
- Product YAML export depends on the official Red Hat toolbox image
- Pre-built binaries: Linux amd64 only

## Release (CI)

Releases are published when a semver tag is pushed:

```bash
git tag v0.2.0
git push origin v0.2.0
```

GitHub Actions runs tests, builds `threescale-export`, `threescale-seed`, and `threescale-visualize` for Linux amd64, and publishes `.tar.gz` artifacts with checksums on [Releases](https://github.com/Everything-is-Code/3scaleextract/releases).

See [CHANGELOG.md](CHANGELOG.md) for version history.

## Optional lab tools

| Tool | Description |
|------|-------------|
| **[docs/SEED.md](docs/SEED.md)** | Load fixtures into a lab tenant to validate export |
| **[docs/VISUALIZE.md](docs/VISUALIZE.md)** | Markdown report, product catalog, HTML dashboard, and Cursor canvas from an export |
| **[testdata/README.md](testdata/README.md)** | Offline export fixture tarball (`export-minimal-1.0.tar.gz`) for tests and ApiShift import |

## License

Apache License 2.0 — see [LICENSE](LICENSE).
