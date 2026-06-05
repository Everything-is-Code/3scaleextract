# threescale-export

CLI para exportar la configuración completa de un tenant **Red Hat 3scale API Management** (products, backends, plans, auth, policies y applications).

## Requisitos

- Go 1.22+
- **Podman** o Docker (método oficial soportado por Red Hat)
- Imagen del toolbox: `registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16`
- Cuenta de [Red Hat Registry Service Account](https://access.redhat.com/terms-based-registry)
- Personal Access Token de 3scale Admin API

Documentación: [Installing the toolbox container image](https://docs.redhat.com/en/documentation/red_hat_3scale_api_management/2.16/html/operating_red_hat_3scale_api_management/the-threescale-toolbox#installing_the_toolbox_container_image)

## Instalar la imagen del toolbox (Red Hat)

```bash
podman login registry.redhat.io
podman pull registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16
podman run registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16 3scale help
```

## Instalar

```bash
cd 3scaleextract
go build -o bin/threescale-export ./cmd/threescale-export
```

## Variables de entorno

Copia `.env.example` a `.env` (gitignored) o exporta las variables manualmente:

```bash
export THREESCALE_ADMIN_URL="https://tenant-admin.example.com"
export THREESCALE_ACCESS_TOKEN="your-token"
export THREESCALE_OUTPUT_DIR="./export"   # opcional, equivalente a --output

# Toolbox (opcional — valores por defecto según documentación Red Hat)
export THREESCALE_TOOLBOX_IMAGE="registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16"
export THREESCALE_TOOLBOX_RUNTIME="podman"   # o docker
export THREESCALE_TOOLBOX_TLS_CERT="/path/to/ca.pem"  # TLS interno / self-signed
```

## Exportar tenant

```bash
bin/threescale-export \
  --admin-url "$THREESCALE_ADMIN_URL" \
  --token "$THREESCALE_ACCESS_TOKEN" \
  --output ./export \
  --include-applications \
  --redact-secrets
```

El export de productos YAML usa internamente:

```bash
podman run --rm registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16 \
  3scale product export https://TOKEN@tenant-admin.example.com {system_name}
```

### Flags

| Flag | Descripción |
|------|-------------|
| `--admin-url` | URL del Admin Portal 3scale |
| `--token` | Personal Access Token |
| `--output` | Directorio de salida |
| `--include-applications` | Incluye applications y accounts (paginado) |
| `--redact-secrets` | Enmascara API keys y secretos OIDC |
| `--per-page` | Tamaño de página Admin API (máx. 500) |
| `--concurrency` | Peticiones concurrentes (default 4) |
| `--insecure` | Omitir verificación TLS en Admin API |
| `--toolbox-image` | Imagen del toolbox (default Red Hat 2.16) |
| `--toolbox-runtime` | `podman` o `docker` (auto-detecta si vacío) |
| `--toolbox-tls-cert` | Certificado CA montado en el contenedor toolbox |
| `--toolbox-binary` | Binario local `3scale` (solo desarrollo; no es el método soportado) |

### TLS self-signed en el toolbox

Según Red Hat, monta el certificado en el contenedor:

```bash
bin/threescale-export \
  --toolbox-tls-cert ./self-signed-cert.pem \
  --output ./export
```

Equivale a `podman run --env SSL_CERT_FILE=... -v cert:/tmp/...`.

## Contenido del export

```
export/
├── manifest.json
├── products/
│   ├── {system_name}.yaml          # toolbox (imagen Red Hat)
│   └── {system_name}/
│       ├── proxy.json
│       ├── policies.json
│       ├── oidc_configuration.json
│       ├── application_plans.json
│       ├── backend_usages.json
│       └── metrics.json
├── backends/{system_name}.json
├── policies/catalog.json
├── applications/page-{n}.json        # con --include-applications
└── accounts/{id}.json
```

## Limitaciones

- No exporta billing, Developer Portal ni analytics
- Requiere acceso a `registry.redhat.io` y runtime de contenedores
- `--toolbox-binary` es alternativa local; Red Hat recomienda la imagen oficial

## Tests

```bash
go test ./...
go test -tags=integration ./internal/export/...   # tenant real (THREESCALE_*)
```

## Desarrollo y demo

Para cargar datos de prueba en un tenant de lab (fixtures, OIDC simulado, policies) y validar el export end-to-end, ver **[docs/SEED.md](docs/SEED.md)**.
