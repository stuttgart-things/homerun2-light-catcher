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

## WLED Mock

A built-in mock server with a live HTML dashboard for development — no real WLED hardware needed.

<details>
<summary><b>Screenshot</b></summary>

The dashboard shows:
- ON/OFF status with glow effect
- Segment cards with effect name, speed, intensity
- Color blocks that light up when active
- Event timeline with timestamps

</details>

<details>
<summary><b>Run standalone</b></summary>

```bash
go run ./cmd/wled-mock/
# Dashboard at http://localhost:8080
```

</details>

<details>
<summary><b>Run embedded with light-catcher</b></summary>

```bash
MOCK_WLED=true MOCK_WLED_PORT=9090 \
PROFILE_PATH=tests/profile.yaml \
LOG_FORMAT=text REDIS_ADDR=localhost \
go run .
# Mock dashboard at http://localhost:9090
```

</details>

## Deployment

<details>
<summary><b>Run locally</b></summary>

```bash
# Start Redis (via Dagger)
task run-redis-as-service

# Run the light-catcher with embedded mock
MOCK_WLED=true PROFILE_PATH=tests/profile.yaml \
REDIS_ADDR=localhost LOG_FORMAT=text go run .
```

</details>

<details>
<summary><b>Container image (ko / ghcr.io)</b></summary>

The container image is built with [ko](https://ko.build) on top of `cgr.dev/chainguard/static` and published to GitHub Container Registry.

```bash
# Pull the image
docker pull ghcr.io/stuttgart-things/homerun2-light-catcher:<tag>

# Run with Docker
docker run \
  -e REDIS_ADDR=redis -e REDIS_PORT=6379 \
  -e REDIS_STREAM=messages \
  -e PROFILE_PATH=/config/profile.yaml \
  -v ./tests/profile.yaml:/config/profile.yaml:ro \
  ghcr.io/stuttgart-things/homerun2-light-catcher:<tag>
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
  handlers/                # Health endpoint
  mock/                    # WLED mock server with HTML dashboard
  models/                  # CaughtMessage struct
  profile/                 # YAML profile loading, effect matching, color palettes
  wled/                    # WLED HTTP client
dagger/                    # CI functions (Lint, Build, Test, Scan)
kcl/                       # KCL deployment manifests (Kubernetes)
tests/                     # Test data (profiles, deploy config, integration messages)
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
- [Container Images](https://github.com/stuttgart-things/homerun2-light-catcher/pkgs/container/homerun2-light-catcher)
- [homerun2-omni-pitcher](https://github.com/stuttgart-things/homerun2-omni-pitcher) (producer)
- [homerun2-core-catcher](https://github.com/stuttgart-things/homerun2-core-catcher) (sibling consumer)
- [homerun-library](https://github.com/stuttgart-things/homerun-library) (shared library)
- [WLED Project](https://kno.wled.ge/)

## License

Apache 2.0
