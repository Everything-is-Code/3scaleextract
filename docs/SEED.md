# Datos de prueba (threescale-seed)

Herramienta **opcional** de desarrollo/demo. Carga fixtures en un tenant 3scale vía **Admin API** para validar `threescale-export` — no forma parte del flujo de migración en producción.

Para exportar un tenant real, usa el [README principal](../README.md).

## Requisitos

- Go 1.22+
- Personal Access Token con permisos de administración en el tenant de lab
- **No** requiere Podman/Docker (solo usa Admin API)

## Instalar

```bash
go build -o bin/threescale-seed ./cmd/threescale-seed
```

## Configuración

```bash
cp .env.example .env   # editar URL y token; .env está en .gitignore
source scripts/load-env.sh
```

Variables mínimas:

| Variable | Descripción |
|----------|-------------|
| `THREESCALE_ADMIN_URL` | URL del Admin Portal |
| `THREESCALE_ACCESS_TOKEN` | Personal Access Token |

## Fixtures incluidos

| Producto | Auth | Backends | Policies | Applications |
|----------|------|----------|----------|--------------|
| `seed_api_key` | API Key | 1 (`seed_payments`) | `cors` | 2 |
| `seed_oidc` | OIDC (RH-SSO simulado) | 1 (`seed_billing`) | `jwt_claim_check`, `cors` | 1 (con `client_id` / `client_secret`) |
| `seed_app_id` | App ID + App Key | 2 | `ip_check`, `cors` | 3 |
| `seed_multi_backend` | API Key | 3 | `edge_limit`, `url_rewriting` | 5 |

Backends compartidos: `seed_payments`, `seed_inventory`, `seed_billing`.

### Matriz de cobertura del export

Cada fixture valida dimensiones concretas del output de `threescale-export`:

| Producto | Qué verificar en `./export` |
|----------|----------------------------|
| `seed_api_key` | `authentication.userkey`, policies CORS, 2 applications con `user_key` |
| `seed_oidc` | `authentication.oidc`, `oidc_configuration.json`, `client_id` en applications |
| `seed_app_id` | App ID/Key auth, pricing rules, policies IP check |
| `seed_multi_backend` | 3 `backend_usages`, policies edge limit y URL rewriting |

### OIDC / RH-SSO simulado

El producto `seed_oidc` configura:

- `backend_version=oidc` en el servicio
- Issuer Keycloak con credenciales Zync: `https://zync-admin:…@sso.example.com/auth/realms/seed-demo`
- Flows: standard + service accounts
- Applications OIDC con `client_id` y `client_secret` (no `user_key`)

> El IdP es ficticio (`sso.example.com`). Sirve para probar el export, no para tráfico real.

## Uso

```bash
source scripts/load-env.sh

# Ver plan de fixtures
bin/threescale-seed --list-fixtures

# Cargar en tenant (idempotente con --skip-existing)
bin/threescale-seed --skip-existing
```

### Flags

| Flag | Descripción |
|------|-------------|
| `--skip-existing` | Omite recursos ya presentes por `system_name` (default: `true`) |
| `--dry-run` | Muestra fixtures sin llamar a la Admin API |
| `--list-fixtures` | Imprime la matriz de cobertura y sale |
| `--admin-url`, `--token`, `--insecure` | Credenciales (o variables `THREESCALE_*`) |

### Comportamiento OIDC al re-ejecutar

Cada ejecución del seed **recrea las applications OIDC** del producto `seed_oidc` para evitar credenciales API Key obsoletas. Los `client_id` / `client_secret` cambiarán entre ejecuciones.

## Seed + export en un paso (lab)

Script de conveniencia para pipelines locales o CI de demo:

```bash
chmod +x scripts/demo/seed-and-export.sh
source scripts/load-env.sh
./scripts/demo/seed-and-export.sh
```

El script compila ambos binarios, ejecuta el seed y exporta a `THREESCALE_OUTPUT_DIR` (default `./export`).

## Código

| Ruta | Descripción |
|------|-------------|
| `cmd/threescale-seed/` | CLI del seeder |
| `internal/seed/fixtures.go` | Catálogo de fixtures |
| `internal/seed/seeder.go` | Lógica Admin API |
| `internal/seed/policies.go` | Configuraciones de policy chain |
| `scripts/demo/seed-and-export.sh` | Flujo lab seed → export |

Comparte el cliente HTTP con el exportador (`internal/admin`, `internal/config`).

## Tests

```bash
go test ./internal/seed/...
```
