# Deployment

## Container Image

Images are built with [ko](https://ko.build/) (no Dockerfile). The image is published to:

```
ghcr.io/stuttgart-things/homerun2-light-catcher
```

## Kubernetes Manifests

KCL templates in `kcl/` generate Kubernetes manifests. Render them with Dagger:

```bash
task render-manifests-quick
```

Or with custom parameters:

```bash
dagger call -m github.com/stuttgart-things/dagger/kcl@v0.82.0 run \
  --source "kcl" \
  --parameters-file="tests/kcl-deploy-profile.yaml" \
  export --path="/tmp/rendered-manifests.yaml"
```

## Kustomize OCI

Release pipelines push a Kustomize base as an OCI artifact:

```
ghcr.io/stuttgart-things/homerun2-light-catcher-kustomize:<version>
```

Push manually:

```bash
task push-kustomize-base
```

## Flux Deployment

Deploy as part of the homerun2 stack using Flux:

```bash
dagger call -m github.com/stuttgart-things/blueprints/kubernetes-deployment@v1.68.0 \
  deploy --kubeconfig /path/to/kubeconfig --namespace homerun2
```

## Development

### Run with Redis (via Dagger)

```bash
# Start Redis as a Dagger service
task run-redis-as-service

# In another terminal, run light-catcher with mock WLED
task run-with-mock
```

### Redis CLI

```bash
task run-redis-cli
```

### Send a test message

```bash
redis-cli XADD messages '*' messageID test-msg-001
```
