# homerun2-light-catcher

Event-driven WLED light controller for the homerun2 ecosystem. Consumes Redis Stream messages and triggers LED effects based on configurable YAML profiles.

[![Build & Test](https://github.com/stuttgart-things/homerun2-light-catcher/actions/workflows/build-test.yaml/badge.svg)](https://github.com/stuttgart-things/homerun2-light-catcher/actions/workflows/build-test.yaml)

## How It Works

The light-catcher connects to a Redis Stream (default: `messages`) via a consumer group, resolves full message payloads from Redis JSON, and matches each message's `system` + `severity` against a YAML effect profile. When a match is found, it sends the corresponding LED effect to a [WLED](https://kno.wled.ge/) device via its JSON API.

```
omni-pitcher → Redis Stream → light-catcher → profile match → WLED device
                                                                  │
                                                            lights flash!
```

### Effect Profile

Effects are configured in a YAML file mapping `(system, severity)` combinations to WLED effects:

```yaml
effects:
  error-git:
    systems: [gitlab, github]
    severity: [ERROR]
    fx: Blurz
    duration: 3          # auto-off after 3 seconds
    color: sunset        # color palette
    endpoint: http://wled:8080

  info:
    systems: ["*"]       # wildcard — matches any system
    severity: [INFO]
    fx: DJ Light
    duration: 3
    color: ocean
    endpoint: http://wled:8080
```

Available effects: Solid, Blink, Breathe, Wipe, Scan, Twinkle, Fireworks, Rainbow, Candle, Chase, Dynamic, Chase Rainbow, Aurora, Blurz, DJ Light

Color palettes: `sunset`, `beach`, `forest`, `ocean` — or single colors: `red`, `yellow`, `green`, `blue`, `white`

## Dashboards

Both the light-catcher and the WLED mock serve dashboards with the HOMERUN² design (Press Start 2P header, purple gradient, stuttgart-things footer).

| Dashboard | URL | Shows |
|-----------|-----|-------|
| **Light Catcher** | `http://localhost:8080/` | Light event timeline with severity, system, effect, color |
| **WLED Mock** | `http://localhost:9090/` (embedded) or `http://localhost:8080/` (standalone) | WLED state, segments, colors, event timeline with trigger context |

The light-catcher dashboard shows events as they are triggered. The mock dashboard shows what the WLED device receives, including severity/system/effect metadata from the light-catcher.

## Running Modes

### Production — with real WLED device

Profile endpoints point to real WLED hardware. No mock needed.

```yaml
effects:
  error:
    systems: ["*"]
    severity: [ERROR]
    fx: Blurz
    duration: 3
    color: sunset
    endpoint: http://192.168.1.100  # real WLED device
```

```bash
REDIS_ADDR=redis-host PROFILE_PATH=profile.yaml go run .
# Light-catcher dashboard at http://localhost:8080
```

### Development — with embedded mock

Set `MOCK_WLED=true` to start the mock server inside the light-catcher process. Profile endpoints point to the embedded mock.

```bash
MOCK_WLED=true MOCK_WLED_PORT=9090 \
PROFILE_PATH=tests/profile.yaml \
LOG_FORMAT=text REDIS_ADDR=localhost \
go run .
# Light-catcher dashboard at http://localhost:8080
# Mock dashboard at http://localhost:9090
```

### Kubernetes — with standalone mock

In Kubernetes, the mock runs as a separate deployment. Profile endpoints point to the mock's service DNS. Both are exposed via HTTPRoute with their own dashboards.

```yaml
# profile.yaml — endpoints point to mock service
effects:
  error:
    systems: ["*"]
    severity: [ERROR]
    fx: Blurz
    duration: 3
    color: sunset
    endpoint: http://homerun2-wled-mock.homerun2.svc.cluster.local
```

Deploy via Flux app (see [flux/apps/homerun2](https://github.com/stuttgart-things/flux/tree/main/apps/homerun2)):

| Service | Image | Dashboard |
|---------|-------|-----------|
| light-catcher | `ghcr.io/stuttgart-things/homerun2-light-catcher` | `https://light-catcher.<DOMAIN>` |
| wled-mock | `ghcr.io/stuttgart-things/homerun2-wled-mock` | `https://wled-mock.<DOMAIN>` |

### WLED Mock Standalone

Run the mock as a standalone binary for testing without Redis:

```bash
go run ./cmd/wled-mock/
# Dashboard at http://localhost:8080
```

## Container Images

Both images are built with [ko](https://ko.build) on top of `cgr.dev/chainguard/static` and published to GitHub Container Registry on every release.

| Image | Description |
|-------|-------------|
| `ghcr.io/stuttgart-things/homerun2-light-catcher:<tag>` | Main light-catcher service |
| `ghcr.io/stuttgart-things/homerun2-wled-mock:<tag>` | Standalone WLED mock server |

```bash
docker pull ghcr.io/stuttgart-things/homerun2-light-catcher:<tag>
docker pull ghcr.io/stuttgart-things/homerun2-wled-mock:<tag>
```

## Deployment

<details>
<summary><b>Run locally (with Redis + embedded mock)</b></summary>

```bash
# Start Redis (via Dagger)
task run-redis-as-service

# Run the light-catcher with embedded mock
task run-with-mock
```

</details>

<details>
<summary><b>Deploy Redis (prerequisite)</b></summary>

```bash
helmfile apply -f \
  git::https://github.com/stuttgart-things/helm.git@database/redis-stack.yaml.gotmpl \
  --state-values-set storageClass=openebs-hostpath \
  --state-values-set password="<REPLACE>" \
  --state-values-set namespace=homerun2
```

</details>

## Development

<details>
<summary><b>Project structure</b></summary>

```
main.go                    # Entrypoint, signal handling, handler composition
cmd/wled-mock/             # Standalone WLED mock binary
internal/
  banner/                  # Animated startup banner (Bubble Tea)
  catcher/                 # Catcher interface (Redis consumer, handlers, mock)
  config/                  # Env-based config loading, slog setup
  dashboard/               # Light event tracker + HTMX dashboard
  handlers/                # Health endpoint
  mock/                    # WLED mock server with HTML dashboard
  models/                  # CaughtMessage struct
  profile/                 # YAML profile loading, effect matching, color palettes
  wled/                    # WLED HTTP client
dagger/                    # CI functions (Lint, Build, BuildMockImage, Test, Scan)
kcl/                       # KCL deployment manifests (light-catcher)
kcl-wled-mock/             # KCL deployment manifests (WLED mock)
tests/                     # Test data (profiles, deploy config)
```

</details>

<details>
<summary><b>Configuration reference</b></summary>

| Variable | Description | Default |
|----------|-------------|---------|
| `REDIS_ADDR` | Redis server address | `localhost` |
| `REDIS_PORT` | Redis server port | `6379` |
| `REDIS_PASSWORD` | Redis password | (empty) |
| `REDIS_STREAM` | Redis stream to consume from | `messages` |
| `CONSUMER_GROUP` | Consumer group name | `homerun2-light-catcher` |
| `CONSUMER_NAME` | Consumer name within the group | hostname |
| `PROFILE_PATH` | Path to WLED effect profile YAML | `profile.yaml` |
| `HEALTH_PORT` | Health endpoint port | `8080` |
| `LOG_FORMAT` | Log format: `json` or `text` | `json` |
| `LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` | `info` |
| `MOCK_WLED` | Set to any value to start embedded WLED mock | (empty) |
| `MOCK_WLED_PORT` | Port for embedded WLED mock | `9090` |

</details>

## Testing

<details>
<summary><b>Unit tests</b></summary>

Unit tests run without Redis (22 tests):

```bash
go test ./internal/... ./cmd/... .
```

</details>

<details>
<summary><b>Integration tests (Dagger + Redis)</b></summary>

Builds the light-catcher, starts Redis, sends a test message:

```bash
task build-test-binary
```

</details>

<details>
<summary><b>Lint</b></summary>

```bash
task lint
```

</details>

<details>
<summary><b>Build and scan container image</b></summary>

```bash
task build-scan-image-ko
```

</details>

## Kubernetes Deployment (KCL)

<details>
<summary><b>Render manifests</b></summary>

The `kcl/` directory contains KCL modules that generate Kubernetes manifests (Namespace, ServiceAccount, ConfigMap, Secret, Deployment, Service, HTTPRoute).

```bash
# Render manifests (non-interactive, uses defaults)
task render-manifests-quick
```

</details>

<details>
<summary><b>Deploy to cluster via KCL</b></summary>

```bash
# Push kustomize base as OCI artifact (requires GITHUB_USER + GITHUB_TOKEN)
task push-kustomize-base

# Deploy to cluster
task deploy-kcl
```

</details>

<details>
<summary><b>Deploy profile</b></summary>

Edit `tests/kcl-deploy-profile.yaml` to customize the deployment:

```yaml
config.image: ghcr.io/stuttgart-things/homerun2-light-catcher:latest
config.namespace: homerun2
config.redisAddr: redis-stack.homerun2.svc.cluster.local
config.redisPort: "6379"
config.redisStream: messages
config.consumerGroup: homerun2-light-catcher
config.redisPassword: changeme
config.healthPort: "8080"
```

</details>

## Links

- [Releases](https://github.com/stuttgart-things/homerun2-light-catcher/releases)
- [Light Catcher Image](https://github.com/stuttgart-things/homerun2-light-catcher/pkgs/container/homerun2-light-catcher)
- [WLED Mock Image](https://github.com/orgs/stuttgart-things/packages/container/package/homerun2-wled-mock)
- [Flux App](https://github.com/stuttgart-things/flux/tree/main/apps/homerun2) (Kubernetes deployment)
- [homerun2-omni-pitcher](https://github.com/stuttgart-things/homerun2-omni-pitcher) (producer)
- [homerun2-core-catcher](https://github.com/stuttgart-things/homerun2-core-catcher) (sibling consumer)
- [homerun-library](https://github.com/stuttgart-things/homerun-library) (shared library)
- [WLED Project](https://kno.wled.ge/)

## License

Apache 2.0
