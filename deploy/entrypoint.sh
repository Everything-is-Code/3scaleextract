#!/bin/bash
set -euo pipefail

# This pod already IS the toolbox image — use the native CLI, never docker/podman.
export THREESCALE_TOOLBOX_BINARY="${THREESCALE_TOOLBOX_BINARY:-/opt/toolbox/bin/3scale}"
unset THREESCALE_TOOLBOX_RUNTIME THREESCALE_TOOLBOX_IMAGE

OUTPUT_DIR="${THREESCALE_OUTPUT_DIR:-/export/data}"
WWW_DIR="${EXPORT_WWW_DIR:-/export/www}"
PORT="${EXPORT_HTTP_PORT:-8080}"
SKIP_EXPORT="${SKIP_EXPORT:-false}"
FORCE_EXPORT="${FORCE_EXPORT:-false}"

if [[ "$(id -u)" -eq 0 ]]; then
  echo "error: refusing to run as root (OpenShift non-root required)" >&2
  exit 1
fi

mkdir -p "${OUTPUT_DIR}" "${WWW_DIR}" \
  /tmp/nginx/client_body /tmp/nginx/proxy /tmp/nginx/fastcgi /tmp/nginx/uwsgi /tmp/nginx/scgi

run_export() {
  local -a args=(--output "${OUTPUT_DIR}")

  if [[ "${THREESCALE_INCLUDE_APPLICATIONS:-}" == "true" ]]; then
    args+=(--include-applications)
  fi
  if [[ "${THREESCALE_INCLUDE_METRICS:-}" == "true" ]]; then
    args+=(--include-metrics)
  fi
  if [[ "${THREESCALE_INSECURE_TLS:-}" == "true" ]]; then
    args+=(--insecure)
  fi
  if [[ "${THREESCALE_REDACT_SECRETS:-}" == "true" ]]; then
    args+=(--redact-secrets)
  fi
  if [[ "${THREESCALE_STRICT:-}" == "true" ]]; then
    args+=(--strict)
  fi
  if [[ -n "${THREESCALE_METRICS_SINCE:-}" ]]; then
    args+=(--metrics-since "${THREESCALE_METRICS_SINCE}")
  fi
  if [[ -n "${THREESCALE_METRICS_UNTIL:-}" ]]; then
    args+=(--metrics-until "${THREESCALE_METRICS_UNTIL}")
  fi
  if [[ -n "${THREESCALE_METRICS_GRANULARITY:-}" ]]; then
    args+=(--metrics-granularity "${THREESCALE_METRICS_GRANULARITY}")
  fi
  if [[ -n "${THREESCALE_CONCURRENCY:-}" ]]; then
    args+=(--concurrency "${THREESCALE_CONCURRENCY}")
  fi
  if [[ "${THREESCALE_VERBOSE:-}" == "true" ]]; then
    args+=(--verbose)
  fi

  echo "==> Running threescale-export to ${OUTPUT_DIR} (uid=$(id -u))"
  /usr/local/bin/threescale-export "${args[@]}"
}

if [[ "${SKIP_EXPORT}" != "true" ]]; then
  if [[ "${FORCE_EXPORT}" == "true" ]] || [[ ! -f "${OUTPUT_DIR}/manifest.json" ]]; then
    run_export
  else
    echo "==> ${OUTPUT_DIR}/manifest.json exists; skipping export (set FORCE_EXPORT=true to re-run)"
  fi
else
  echo "==> SKIP_EXPORT=true; serving existing data only"
fi

echo "==> Building download index and archive"
/usr/local/bin/generate-index.sh "${OUTPUT_DIR}" "${WWW_DIR}"

echo "==> Export ready — HTTP server on :${PORT} (uid=$(id -u))"
echo "    Landing page: /"
echo "    Browse files: /data/"
echo "    Full archive: /export.tar.gz"
exec nginx -g 'daemon off;'
