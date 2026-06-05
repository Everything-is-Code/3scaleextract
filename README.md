# threescale-export

CLI para exportar la configuración completa de un tenant **Red Hat 3scale API Management** (products, backends, plans, auth, policies y applications).

## Ejecutar el binario

Descarga los binarios desde [Releases](https://github.com/Everything-is-Code/3scaleextract/releases) (Linux x86_64). No requiere Go ni compilar nada.

### Prerrequisitos

| Requisito | Descripción |
|-----------|-------------|
| **Binario** | `threescale-export` ([Releases](https://github.com/Everything-is-Code/3scaleextract/releases)) |
| **Docker** o **Podman** | Runtime de contenedores (**no** se requiere Podman; basta con Docker) |
| **Imagen toolbox 3scale** | `registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16` |
| **Red Hat Registry** | Cuenta de [Registry Service Account](https://access.redhat.com/terms-based-registry) |
| **Token Admin API** | Personal Access Token del Admin Portal 3scale |

Documentación Red Hat: [Installing the toolbox container image](https://docs.redhat.com/en/documentation/red_hat_3scale_api_management/2.16/html/operating_red_hat_3scale_api_management/the-threescale-toolbox#installing_the_toolbox_container_image)

### Dependencias de runtime

**Imagen del toolbox (Red Hat)** — con Docker:

```bash
docker login registry.redhat.io
docker pull registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16
docker run --rm registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16 3scale help
```

Con **Podman**, sustituye `docker` por `podman`.

**Runtime de contenedores** — el binario auto-detecta Docker o Podman en el `PATH` (Docker primero). Para forzar uno:

```bash
export THREESCALE_TOOLBOX_RUNTIME=docker   # o podman
```

### Configuración

Variables obligatorias:

```bash
export THREESCALE_ADMIN_URL="https://tenant-admin.example.com"
export THREESCALE_ACCESS_TOKEN="your-personal-access-token"
```

Variables opcionales:

```bash
export THREESCALE_OUTPUT_DIR="./export"   # equivalente a --output

export THREESCALE_TOOLBOX_IMAGE="registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16"
export THREESCALE_TOOLBOX_RUNTIME="docker"
export THREESCALE_TOOLBOX_TLS_CERT="/path/to/ca.pem"   # TLS self-signed en Admin Portal
```

También puedes pasar credenciales por flags (`--admin-url`, `--token`).

### Export

```bash
chmod +x threescale-export

./threescale-export \
  --output ./export \
  --include-applications \
  --redact-secrets
```

El export híbrido combina:

- **Admin API** — backends, applications, accounts, JSON complementario por producto.
- **Toolbox (contenedor Red Hat)** — YAML del producto (`3scale product export`).

| Flag | Descripción |
|------|-------------|
| `--admin-url` | URL del Admin Portal 3scale |
| `--token` | Personal Access Token |
| `--output` | Directorio de salida (default `./export`) |
| `--include-applications` | Incluye applications y accounts (paginado) |
| `--redact-secrets` | Enmascara API keys y secretos OIDC |
| `--per-page` | Tamaño de página Admin API (máx. 500) |
| `--concurrency` | Peticiones concurrentes (default 4) |
| `--insecure` | Omitir verificación TLS en Admin API |
| `--toolbox-image` | Imagen del toolbox (default Red Hat 2.16) |
| `--toolbox-runtime` | `docker` o `podman` (auto-detecta si vacío) |
| `--toolbox-tls-cert` | Certificado CA montado en el contenedor toolbox |

TLS self-signed (Admin Portal o toolbox):

```bash
./threescale-export \
  --toolbox-tls-cert ./ca.pem \
  --insecure \
  --output ./export
```

### Visualizar el export

Opcional: genera un informe Markdown del directorio exportado (sin Admin API ni contenedores):

```bash
chmod +x threescale-visualize
./threescale-visualize ./export -o ./report
```

Ver **[docs/VISUALIZE.md](docs/VISUALIZE.md)** para el layout del informe.

---

## Construir desde código

Para desarrolladores que modifican el proyecto.

### Requisitos

- Go 1.22+

### Compilar

```bash
git clone https://github.com/Everything-is-Code/3scaleextract.git
cd 3scaleextract

go build -o bin/threescale-export ./cmd/threescale-export
go build -o bin/threescale-seed ./cmd/threescale-seed
go build -o bin/threescale-visualize ./cmd/threescale-visualize
```

### Ejecutar binarios locales

```bash
bin/threescale-export --output ./export --include-applications --redact-secrets
bin/threescale-visualize ./export -o ./report
```

### Tests

```bash
go test ./...
go test -tags=integration ./internal/export/...   # tenant real (THREESCALE_*)
```

---

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
- El export YAML de productos depende de la imagen oficial del toolbox Red Hat

## Release (CI)

Los releases se publican al pushear un tag semver:

```bash
git tag v0.1.1
git push origin v0.1.1
```

GitHub Actions ejecuta tests, compila `threescale-export`, `threescale-seed` y `threescale-visualize` para Linux amd64, y publica los `.tar.gz` con checksums en [Releases](https://github.com/Everything-is-Code/3scaleextract/releases).

## Herramientas opcionales (lab)

| Herramienta | Descripción |
|-------------|-------------|
| **[docs/SEED.md](docs/SEED.md)** | Carga fixtures en un tenant de lab para validar el export |
| **[docs/VISUALIZE.md](docs/VISUALIZE.md)** | Genera informe Markdown del export |
