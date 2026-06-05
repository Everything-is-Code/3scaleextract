#!/usr/bin/env bash
# Load local 3scale credentials from .env (gitignored).
set -a
# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/../.env"
set +a
