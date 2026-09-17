# Build threescale-export and push to GitHub Packages (ghcr.io).
# Usage:
#   $env:REDHAT_REGISTRY_USER = '...'
#   $env:REDHAT_REGISTRY_TOKEN = '...'
#   $env:GHCR_TOKEN = 'ghp_...'    # PAT with write:packages
#   $env:GHCR_USER = 'pcastelo'
#   .\deploy\build-and-push.ps1
#
# Skip push: $env:PUSH = '0'

$ErrorActionPreference = 'Stop'
$Root = Resolve-Path (Join-Path $PSScriptRoot '..')

$Image = if ($env:IMAGE) { $env:IMAGE } else { 'ghcr.io/everything-is-code/threescale-export' }
$Version = if ($env:VERSION) { $env:VERSION } else { 'dev' }
$Push = if ($env:PUSH) { $env:PUSH } else { '1' }

function Require-Env($Name) {
    if (-not $env:$Name) {
        Write-Error "Missing env var: $Name"
    }
}

Write-Host "==> Logging into registry.redhat.io (toolbox base image)"
Require-Env 'REDHAT_REGISTRY_USER'
Require-Env 'REDHAT_REGISTRY_TOKEN'
$rhUser = $env:REDHAT_REGISTRY_USER
if ($env:REDHAT_REGISTRY_USERNAME) { $rhUser = $env:REDHAT_REGISTRY_USERNAME }
$rhPass = $env:REDHAT_REGISTRY_TOKEN
if ($env:REDHAT_REGISTRY_PASSWORD) { $rhPass = $env:REDHAT_REGISTRY_PASSWORD }
$rhPass | podman login registry.redhat.io -u $rhUser --password-stdin

Write-Host "==> Building $Image:$Version"
podman build `
    -f (Join-Path $Root 'deploy\Dockerfile') `
    --build-arg "VERSION=$Version" `
    -t "${Image}:${Version}" `
    -t "${Image}:latest" `
    $Root

podman images $Image

Write-Host "==> Non-root smoke test (OpenShift-like UID)"
podman run --rm --entrypoint /bin/bash -u 1000680000:0 "${Image}:latest" -c 'test "$(id -u)" -ne 0 && nginx -t'

if ($Push -eq '1') {
    Write-Host "==> Logging into ghcr.io"
    $ghUser = $env:GHCR_USER
    $ghToken = $env:GHCR_TOKEN
    if (-not $ghToken) {
        $ghToken = (gh auth token)
        $ghUser = (gh api user -q .login)
    }
    if (-not $ghUser -or -not $ghToken) {
        Write-Error "Set GHCR_USER + GHCR_TOKEN or run 'gh auth login'"
    }
    $ghToken | podman login ghcr.io -u $ghUser --password-stdin

    Write-Host "==> Pushing to GitHub Packages"
    podman push "${Image}:${Version}"
    podman push "${Image}:latest"
    Write-Host "==> Done: https://github.com/Everything-is-Code/3scaleextract/packages"
}
