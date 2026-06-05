# Visualizador de export (threescale-visualize)

Herramienta **opcional** que genera un informe Markdown a partir del directorio exportado por `threescale-export`. Útil para revisiones de migración sin abrir JSON/YAML manualmente.

Para exportar un tenant, usa el [README principal](../README.md).

## Requisitos

- Go 1.22+ (solo para compilar; el binario de release no requiere Go)
- Un export existente con `schema_version` **1.0** (`manifest.json` en la raíz)
- **No** requiere Admin API, Docker ni variables `THREESCALE_*`

## Instalar

```bash
go build -o bin/threescale-visualize ./cmd/threescale-visualize
```

O descarga el tarball `threescale-visualize-v*.*.*-linux-amd64.tar.gz` desde [Releases](https://github.com/Everything-is-Code/3scaleextract/releases).

## Uso

```bash
# Tras un export (por defecto ./export)
bin/threescale-visualize ./export -o ./report
```

| Flag | Descripción |
|------|-------------|
| `-o`, `--output` | Directorio del informe (default `./report`) |
| `--version` | Versión del binario |

## Contenido del informe

```
report/
├── index.md              # overview, auth matrix, grafo Mermaid
├── backends.md           # catálogo de backends
├── applications.md       # solo si el export incluyó applications
└── products/
    └── {system_name}.md  # auth, policies, planes, backends
```

Abre `index.md` en GitHub, VS Code o Cursor para navegar por enlaces relativos. Los diagramas Mermaid se renderizan en GitHub y en editores compatibles.

## Demo con datos seed

```bash
# 1. Cargar fixtures (ver docs/SEED.md)
bin/threescale-seed

# 2. Exportar tenant de lab
bin/threescale-export --output ./export --include-applications --redact-secrets

# 3. Generar informe
bin/threescale-visualize ./export -o ./report
```

## Limitaciones (v1)

- Solo lectura del export en disco; no llama a Admin API
- No incluye contenido de `policies/catalog.json` (referencia global, no config del tenant)
- Secretos enmascarados (`***REDACTED***`) se muestran tal cual — no se de-redactan
- Sin servidor HTTP ni HTML (previsto para versiones futuras)
