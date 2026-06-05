# threescale-export

CLI para exportar la configuración completa de un tenant **Red Hat 3scale API Management** (products, backends, plans, auth, policies y applications).

## Inicio rápido

1. Descarga el binario desde [Releases](https://github.com/Everything-is-Code/3scaleextract/releases) y hazlo ejecutable.
2. Instala las dependencias de runtime (contenedor + imagen toolbox).
3. Configura las variables de entorno.
4. Ejecuta el export.

```bash
chmod +x threescale-export

export THREESCALE_ADMIN_URL="https://tenant-admin.example.com"
export THREESCALE_ACCESS_TOKEN="your-personal-access-token"

./threescale-export \
  --output ./export \
  --include-applications \
  --redact-secrets
```

## Prerrequisitos (ejecutar el binario)

| Requisito | Descripción |
|-----------|-------------|
| **Binario** | `threescale-export` para Linux x86_64 ([Releases](https://github.com/Everything-is-Code/3scaleextract/releases)) |
| **Docker** o **Podman** | Runtime de contenedores (**no** se requiere Podman; basta con Docker) |
| **Imagen toolbox 3scale** | `registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16` |
| **Red Hat Registry** | Cuenta de [Registry Service Account](https://access.redhat.com/terms-based-registry) para descargar la imagen |
| **Token Admin API** | Personal Access Token del Admin Portal 3scale |

No se requiere Go ni compilar nada para usar el binario.

Documentación Red Hat: [Installing the toolbox container image](https://docs.redhat.com/en/documentation/red_hat_3scale_api_management/2.16/html/operating_red_hat_3scale_api_management/the-threescale-toolbox#installing_the_toolbox_container_image)

## Instalar dependencias

### 1. Imagen del toolbox (Red Hat)

Con **Docker**:

```bash
docker login registry.redhat.io
docker pull registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16
docker run --rm registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16 3scale help
```

Con **Podman** (alternativa), sustituye `docker` por `podman`.

### 2. Runtime de contenedores

El binario **auto-detecta** Docker o Podman en el `PATH` (intenta Docker primero). **No hace falta configurar nada** si solo tienes Docker instalado.

Para forzar un runtime concreto:

```bash
export THREESCALE_TOOLBOX_RUNTIME=docker   # o podman
```

## Configuración

### Variables obligatorias

```bash
export THREESCALE_ADMIN_URL="https://tenant-admin.example.com"
export THREESCALE_ACCESS_TOKEN="your-personal-access-token"
```

### Variables opcionales

```bash
export THREESCALE_OUTPUT_DIR="./export"   # equivalente a --output

# Toolbox (opcional — auto-detecta docker o podman si se omite THREESCALE_TOOLBOX_RUNTIME)
export THREESCALE_TOOLBOX_IMAGE="registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16"
export THREESCALE_TOOLBOX_RUNTIME="docker"   # solo si quieres forzar el runtime
export THREESCALE_TOOLBOX_TLS_CERT="/path/to/ca.pem"   # solo si el Admin Portal usa TLS self-signed
```

También puedes pasar credenciales por flags (`--admin-url`, `--token`) en lugar de variables de entorno.

## Ejecutar export

```bash
./threescale-export \
  --output ./export \
  --include-applications \
  --redact-secrets
```

El export híbrido combina:

- **Admin API** — backends, applications, accounts, JSON complementario por producto.
- **Toolbox (contenedor Red Hat)** — YAML del producto (`3scale product export`).

### Flags

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
| `--toolbox-runtime` | `docker` o `podman` (auto-detecta si vacío; Docker primero) |
| `--toolbox-tls-cert` | Certificado CA montado en el contenedor toolbox |

### TLS self-signed

Si el Admin Portal o el toolbox requieren un CA interno:

```bash
./threescale-export \
  --toolbox-tls-cert ./ca.pem \
  --insecure \
  --output ./export
```

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

---

## Construir desde código (desarrolladores)

Esta sección es solo para quien desarrolla o modifica el proyecto. **Los clientes no necesitan Go.**

### Requisitos adicionales

- Go 1.22+

### Compilar

```bash
git clone https://github.com/Everything-is-Code/3scaleextract.git
cd 3scaleextract
go build -o bin/threescale-export ./cmd/threescale-export
```

### Ejecutar binario local

```bash
bin/threescale-export --output ./export --include-applications --redact-secrets
```

### Tests

```bash
go test ./...
go test -tags=integration ./internal/export/...   # tenant real (THREESCALE_*)
```

### Release (CI)

Los releases se publican automáticamente al pushear un tag semver:

```bash
git tag v0.1.1
git push origin v0.1.1
```

GitHub Actions ejecuta tests, compila `threescale-export`, `threescale-seed` y `threescale-visualize` para Linux amd64, y publica los `.tar.gz` con checksums en [Releases](https://github.com/Everything-is-Code/3scaleextract/releases).

### Datos de prueba (lab)

Para cargar fixtures en un tenant de lab y validar el export end-to-end, ver **[docs/SEED.md](docs/SEED.md)**.

Para generar un informe Markdown del export, ver **[docs/VISUALIZE.md](docs/VISUALIZE.md)**.
