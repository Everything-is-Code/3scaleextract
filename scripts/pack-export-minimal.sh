#!/usr/bin/env bash
# Pack internal/visualize/testdata/export-minimal into testdata/export-minimal-1.0.tar.gz
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE="${ROOT}/internal/visualize/testdata/export-minimal"
SOURCE_PARENT="${ROOT}/internal/visualize/testdata"
SCHEMA_VERSION="1.0"
ARTIFACT="export-minimal-${SCHEMA_VERSION}.tar.gz"
OUT_DIR="${ROOT}/testdata"
OUT_TAR="${OUT_DIR}/${ARTIFACT}"
OUT_SHA="${OUT_DIR}/${ARTIFACT}.sha256"

usage() {
	echo "Usage: $0 [--check]" >&2
	exit 1
}

check_mode=false
if [[ "${1:-}" == "--check" ]]; then
	check_mode=true
elif [[ -n "${1:-}" ]]; then
	usage
fi

if [[ ! -f "${SOURCE}/manifest.json" ]]; then
	echo "missing source fixture: ${SOURCE}/manifest.json" >&2
	exit 1
fi

manifest_schema="$(grep -o '"schema_version"[[:space:]]*:[[:space:]]*"[^"]*"' "${SOURCE}/manifest.json" | sed 's/.*"\([^"]*\)"$/\1/')"
if [[ "${manifest_schema}" != "${SCHEMA_VERSION}" ]]; then
	echo "manifest schema_version ${manifest_schema} != ${SCHEMA_VERSION}" >&2
	exit 1
fi

pack_to() {
	local dest="$1"
	mkdir -p "$(dirname "${dest}")"
	# Reproducible archive: fixed mtime/owner and sorted names (CI vs local).
	tar --sort=name \
		--mtime='UTC 2020-01-01' \
		--owner=0 --group=0 --numeric-owner \
		-czf "${dest}" -C "${SOURCE_PARENT}" export-minimal
}

if [[ "${check_mode}" == true ]]; then
	if [[ ! -f "${OUT_TAR}" || ! -f "${OUT_SHA}" ]]; then
		echo "committed tarball missing; run ./scripts/pack-export-minimal.sh" >&2
		exit 1
	fi

	tmp="$(mktemp "${TMPDIR:-/tmp}/export-minimal-check.XXXXXX.tar.gz")"
	trap 'rm -f "${tmp}"' EXIT
	pack_to "${tmp}"

	expected="$(sha256sum "${OUT_TAR}" | awk '{print $1}')"
	actual="$(sha256sum "${tmp}" | awk '{print $1}')"
	if [[ "${expected}" != "${actual}" ]]; then
		echo "export-minimal tarball is stale; run ./scripts/pack-export-minimal.sh" >&2
		exit 1
	fi

	echo "export-minimal tarball is up to date"
	exit 0
fi

mkdir -p "${OUT_DIR}"
pack_to "${OUT_TAR}"
( cd "${OUT_DIR}" && sha256sum "${ARTIFACT}" > "${ARTIFACT}.sha256" )
echo "wrote ${OUT_TAR}"
echo "wrote ${OUT_SHA}"
