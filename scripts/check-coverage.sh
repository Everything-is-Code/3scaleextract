#!/usr/bin/env bash
# Fail if Go statement coverage in a coverprofile is below MIN_COVERAGE (default 80).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROFILE="${1:-${ROOT}/coverage.out}"
MIN_COVERAGE="${MIN_COVERAGE:-80}"

usage() {
	echo "Usage: $0 [coverage.out]" >&2
	echo "  MIN_COVERAGE=80 (default) minimum total statement coverage percentage" >&2
	exit 1
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	usage
fi

if [[ ! -f "${PROFILE}" ]]; then
	echo "coverage profile not found: ${PROFILE}" >&2
	exit 1
fi

line="$(go tool cover -func="${PROFILE}" | awk '/^total:/ {print; exit}')"
if [[ -z "${line}" ]]; then
	echo "could not parse total coverage from ${PROFILE}" >&2
	exit 1
fi

pct="$(echo "${line}" | awk '{gsub(/%/,"",$3); print $3}')"
echo "${line}"

awk -v pct="${pct}" -v min="${MIN_COVERAGE}" 'BEGIN {
	if (pct + 0 >= min + 0) exit 0
	printf "coverage %.1f%% is below minimum %.1f%%\n", pct, min > "/dev/stderr"
	exit 1
}'
