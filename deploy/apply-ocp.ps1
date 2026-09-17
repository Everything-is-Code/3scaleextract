# Deploy threescale-export to OpenShift (workshop-friendly).
# Pushes local image to the cluster registry — no GHCR required.
#
# Prereq: oc logged in to the target cluster
#   oc login https://api.cluster-4vhhf.dyn.redhatworkshops.io:6443 --token=... --insecure-skip-tls-verify=true

$ErrorActionPreference = 'Stop'
$Root = Resolve-Path (Join-Path $PSScriptRoot '..')
$Ns = if ($env:NAMESPACE) { $env:NAMESPACE } else { 'threescale-export' }
$LocalImage = if ($env:LOCAL_IMAGE) { $env:LOCAL_IMAGE } else { 'ghcr.io/everything-is-code/threescale-export:latest' }

if (-not $env:THREESCALE_ADMIN_URL -or -not $env:THREESCALE_ACCESS_TOKEN) {
    Write-Error 'Set THREESCALE_ADMIN_URL and THREESCALE_ACCESS_TOKEN'
}

$user = oc whoami 2>$null
if (-not $user) {
    Write-Error @"
Not logged in to OpenShift. Run:
  oc login https://api.cluster-4vhhf.dyn.redhatworkshops.io:6443 --token=YOUR_OCP_TOKEN --insecure-skip-tls-verify=true
Token: OpenShift Console -> admin -> Copy login command
"@
}
Write-Host "==> Cluster: $user @ $(oc config view --minify -o jsonpath='{.clusters[0].cluster.server}')"

$nsExists = oc get namespace $Ns -o name 2>$null
if (-not $nsExists) {
    oc new-project $Ns --display-name='3scale Export'
} else {
    oc project $Ns
}

$registry = (oc registry info)
$remote = "$registry/$Ns/threescale-export:latest"
Write-Host "==> Pushing $LocalImage -> $remote"
oc registry login
podman tag $LocalImage $remote
podman push $remote

$manifest = Join-Path $env:TEMP "threescale-export-ocp.yaml"
$adminUrl = $env:THREESCALE_ADMIN_URL
$token = $env:THREESCALE_ACCESS_TOKEN
$insecure = if ($env:THREESCALE_INSECURE_TLS) { $env:THREESCALE_INSECURE_TLS } else { 'true' }

(Get-Content (Join-Path $Root 'deploy\openshift.yaml') -Raw) `
    -replace 'THREESCALE_ADMIN_URL: "https://TENANT-admin.apps.example.com"', "THREESCALE_ADMIN_URL: `"$adminUrl`"" `
    -replace 'THREESCALE_ACCESS_TOKEN: "replace-with-personal-access-token"', "THREESCALE_ACCESS_TOKEN: `"$token`"" `
    -replace 'ghcr.io/everything-is-code/threescale-export:latest', $remote `
    -replace '(?m)- name: THREESCALE_INSECURE_TLS\s+value: "false"', "- name: THREESCALE_INSECURE_TLS`n              value: `"$insecure`"" |
    Set-Content $manifest -Encoding utf8

Write-Host "==> Applying $manifest"
oc apply -f $manifest

Write-Host "==> Waiting for rollout"
oc rollout status deployment/threescale-export -n $Ns --timeout=30m
$routeHost = oc get route threescale-export -n $Ns -o jsonpath='{.spec.host}' 2>$null
Write-Host "==> Route: https://$routeHost/"
Write-Host "==> Logs: oc logs -f deployment/threescale-export -n $Ns"
