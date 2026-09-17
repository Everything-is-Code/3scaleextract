#!/usr/bin/env bash
# Build and optionally push threescale-export to GitHub Packages (ghcr.io).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${IMAGE:-ghcr.io/everything-is-code/threescale-export}"
VERSION="${VERSION:-dev}"
RUNTIME="${CONTAINER_RUNTIME:-podman}"
PUSH="${PUSH:-0}"

cd "${ROOT}"

echo "==> Build ${IMAGE}:${VERSION}"
"${RUNTIME}" build \
  -f deploy/Dockerfile \
  --build-arg "VERSION=${VERSION}" \
  -t "${IMAGE}:${VERSION}" \
  -t "${IMAGE}:latest" \
  .

echo "==> Image ready:"
"${RUNTIME}" images "${IMAGE}"

if [[ "${PUSH}" == "1" ]]; then
  echo "==> Pushing to ghcr.io (login first: echo TOKEN | ${RUNTIME} login ghcr.io -u USER --password-stdin)"
  "${RUNTIME}" push "${IMAGE}:${VERSION}"
  "${RUNTIME}" push "${IMAGE}:latest"
  echo "==> Published: https://github.com/Everything-is-Code/3scaleextract/packages"
fi
