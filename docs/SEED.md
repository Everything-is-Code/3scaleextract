# Lab fixtures (threescale-seed)

**Optional** development/demo tool. Loads fixtures into a 3scale tenant via the **Admin API** to validate `threescale-export` — not part of the production migration flow.

To export a real tenant, use the [main README](../README.md).

## Requirements

- Go 1.22+
- Personal Access Token with admin permissions on the lab tenant
- **No** Podman/Docker required (Admin API only)

## Install

```bash
go build -o bin/threescale-seed ./cmd/threescale-seed
```

## Configuration

```bash
cp .env.example .env   # edit URL and token; .env is gitignored
source scripts/load-env.sh
```

Minimum variables:

| Variable | Description |
|----------|-------------|
| `THREESCALE_ADMIN_URL` | Admin Portal URL |
| `THREESCALE_ACCESS_TOKEN` | Personal Access Token |

## Included fixtures

| Product | Auth | Backends | Policies | Applications |
|---------|------|----------|----------|--------------|
| `seed_api_key` | API Key | 1 (`seed_payments`) | `cors` | 2 |
| `seed_oidc` | OIDC (simulated RH-SSO) | 1 (`seed_billing`) | `jwt_claim_check`, `cors` | 1 (with `client_id` / `client_secret`) |
| `seed_app_id` | App ID + App Key | 2 | `ip_check`, `cors` | 3 |
| `seed_multi_backend` | API Key | 3 | `edge_limit`, `url_rewriting` | 5 |

Shared backends: `seed_payments`, `seed_inventory`, `seed_billing`.

### Export coverage matrix

Each fixture validates specific dimensions of `threescale-export` output:

| Product | What to verify in `./export` |
|---------|------------------------------|
| `seed_api_key` | `authentication.userkey`, CORS policies, 2 applications with `user_key` |
| `seed_oidc` | `authentication.oidc`, `oidc_configuration.json`, `client_id` on applications |
| `seed_app_id` | App ID/Key auth, pricing rules, IP check policies |
| `seed_multi_backend` | 3 `backend_usages`, edge limit and URL rewriting policies |

### Simulated OIDC / RH-SSO

The `seed_oidc` product configures:

- `backend_version=oidc` on the service
- Keycloak issuer with Zync credentials: `https://zync-admin:…@sso.example.com/auth/realms/seed-demo`
- Flows: standard + service accounts
- OIDC applications with `client_id` and `client_secret` (not `user_key`)

> The IdP is fictional (`sso.example.com`). Used to test export, not real traffic.

## Usage

```bash
source scripts/load-env.sh

# Show fixture plan
bin/threescale-seed --list-fixtures

# Load into tenant (idempotent with --skip-existing)
bin/threescale-seed --skip-existing
```

### Flags

| Flag | Description |
|------|-------------|
| `--skip-existing` | Skip resources already present by `system_name` (default: `true`) |
| `--dry-run` | Print fixtures without calling Admin API |
| `--list-fixtures` | Print coverage matrix and exit |
| `--admin-url`, `--token`, `--insecure` | Credentials (or `THREESCALE_*` env vars) |

### OIDC behavior on re-run

Each seed run **recreates OIDC applications** for `seed_oidc` to avoid stale API Key credentials. `client_id` / `client_secret` will change between runs.

## One-step seed + export (lab)

Convenience script for local pipelines or demo CI:

```bash
chmod +x scripts/demo/seed-and-export.sh
source scripts/load-env.sh
./scripts/demo/seed-and-export.sh
```

The script builds both binaries, runs seed, and exports to `THREESCALE_OUTPUT_DIR` (default `./export`).

**Full E2E** (seed → export → visualize → GateForge analyze): use [GateForge `scripts/e2e-seed-export-analyze.sh`](https://github.com/Everything-is-Code/gateforge/blob/main/scripts/e2e-seed-export-analyze.sh) or [rhcl-ai `scripts/e2e-lab.sh`](https://github.com/Everything-is-Code/rhcl-ai/blob/main/scripts/e2e-lab.sh) ([PR #6](https://github.com/Everything-is-Code/rhcl-ai/pull/6)).

## Code

| Path | Description |
|------|-------------|
| `cmd/threescale-seed/` | Seeder CLI |
| `internal/seed/fixtures.go` | Fixture catalog |
| `internal/seed/seeder.go` | Admin API logic |
| `internal/seed/policies.go` | Policy chain configurations |
| `scripts/demo/seed-and-export.sh` | Lab seed → export flow |

Shares HTTP client with the exporter (`internal/admin`, `internal/config`).

## Tests

```bash
go test ./internal/seed/...
```
