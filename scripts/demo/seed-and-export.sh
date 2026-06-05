#!/usr/bin/env bash
# Lab/demo: seed dummy tenant data, then export with threescale-export.
# See docs/SEED.md
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

: "${THREESCALE_ADMIN_URL:?set THREESCALE_ADMIN_URL}"
: "${THREESCALE_ACCESS_TOKEN:?set THREESCALE_ACCESS_TOKEN}"

OUTPUT="${THREESCALE_OUTPUT_DIR:-./export}"

go build -o bin/threescale-seed ./cmd/threescale-seed
go build -o bin/threescale-export ./cmd/threescale-export

echo "==> Seeding tenant..."
bin/threescale-seed --skip-existing

echo "==> Exporting tenant to ${OUTPUT}..."
bin/threescale-export --output "${OUTPUT}" --include-applications --redact-secrets

echo "==> Done. Inspect ${OUTPUT}/manifest.json"
