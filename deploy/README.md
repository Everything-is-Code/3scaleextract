# threescale-export on OpenShift

Export a 3scale tenant in-cluster, then download the result from a small web UI.

## Image (GitHub Packages)

Published automatically to:

```text
ghcr.io/everything-is-code/threescale-export:latest
```

Workflow: [`.github/workflows/container.yml`](../.github/workflows/container.yml)

**Repository secrets required for CI build** (Settings → Secrets → Actions):

| Secret | Purpose |
|--------|---------|
| `REDHAT_REGISTRY_USERNAME` | Pull `registry.redhat.io/3scale-amp2/toolbox-rhel9` during image build |
| `REDHAT_REGISTRY_PASSWORD` | Red Hat registry service account token |

### Local build + push to [GitHub Packages](https://github.com/Everything-is-Code/3scaleextract/packages)

```powershell
# 1. Red Hat registry (toolbox base) — https://access.redhat.com/RegistryAuthentication
$env:REDHAT_REGISTRY_USER = 'your-service-account'
$env:REDHAT_REGISTRY_TOKEN = 'your-token'

# 2. Build and push (uses `gh auth token` for ghcr.io if GHCR_TOKEN unset)
.\deploy\build-and-push.ps1
```

Image published as:

```text
ghcr.io/everything-is-code/threescale-export:latest
```

Manual push:

```powershell
podman login registry.redhat.io
podman build -f deploy/Dockerfile -t ghcr.io/everything-is-code/threescale-export:latest .
gh auth token | podman login ghcr.io -u pcastelo --password-stdin
podman push ghcr.io/everything-is-code/threescale-export:latest
```

After the first successful run, set the package visibility to **public** under GitHub → Packages (or add an `imagePullSecrets` entry in the manifest).

## Deploy (copy / paste)

1. Edit credentials in [`openshift.yaml`](openshift.yaml) — only the `Secret` block:

```yaml
stringData:
  THREESCALE_ADMIN_URL: "https://tenant-admin.apps.example.com"
  THREESCALE_ACCESS_TOKEN: "your-pat"
```

2. Apply:

```bash
oc apply -f deploy/openshift.yaml
```

3. Wait and open the route:

```bash
oc logs -f deployment/threescale-export -n threescale-export
oc get route threescale-export -n threescale-export
```

| URL | Content |
|-----|---------|
| `/` | Landing page |
| `/export.tar.gz` | Full archive |
| `/data/` | Browse files |

## Optional env vars (Deployment)

Edit `env:` in `openshift.yaml` or patch live:

```bash
oc set env deployment/threescale-export -n threescale-export \
  THREESCALE_INCLUDE_METRICS=true \
  THREESCALE_INSECURE_TLS=true
```

| Variable | Default | Notes |
|----------|---------|-------|
| `THREESCALE_INCLUDE_METRICS` | `false` | Enterprise + PAT Analytics scope |
| `THREESCALE_INSECURE_TLS` | `false` | Self-signed lab certs |
| `FORCE_EXPORT` | `false` | Re-run export on restart |
| `SKIP_EXPORT` | `false` | Serve existing PVC only |

## Private GHCR package

```bash
oc create secret docker-registry ghcr-pull \
  --docker-server=ghcr.io \
  --docker-username=YOUR_GITHUB_USER \
  --docker-password=YOUR_GITHUB_PAT \
  -n threescale-export
```

Uncomment `imagePullSecrets` in `openshift.yaml`, then re-apply.

## Image design

The **shipped image is the Red Hat toolbox** (`toolbox-rhel9:3scale2.16`) with extra layers:

| Layer | Role |
|-------|------|
| `FROM toolbox-rhel9` | Base image — `/opt/toolbox/bin/3scale` already inside |
| `threescale-export` | Go CLI copied in at build time |
| nginx | Serves `/export.tar.gz` and `/data/` after export |

**No container-in-container.** `threescale-export` calls `/opt/toolbox/bin/3scale product export …` directly (`THREESCALE_TOOLBOX_BINARY`). Docker/podman are not installed or used in the running pod.

A separate `go-toolset` stage only compiles the Go binary during `podman build`; it is not in the final image.

- **OpenShift non-root**: `USER 1001`, writable paths use `chgrp 0` + `g+rwX`

Local smoke test with an OpenShift-like UID:

```bash
podman run --rm -u 1000680000:0 -p 8080:8080 \
  -e THREESCALE_ADMIN_URL=... -e THREESCALE_ACCESS_TOKEN=... \
  ghcr.io/everything-is-code/threescale-export:latest
```
