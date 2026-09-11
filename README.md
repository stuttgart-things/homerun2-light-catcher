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

### Where a new instance starts reading

A light effect is a signal about *now*, so the light-catcher never replays a stream's history:

- **New consumer group** (first start, a new instance, a newly added stream, or after the group was deleted): the group is created at `$` and only messages pitched **after** it exists are handled. Set `CONSUMER_START_ID=0` to replay the whole stream instead, or a stream ID to start after that entry.
- **Existing consumer group** (a restart): the group keeps its position, so messages pitched while the catcher was down are still delivered. `CONSUMER_START_ID` has no effect on an existing group.

Messages are handled **one at a time, in stream order**, up to and including the WLED call. A burst (e.g. a point and the match-winning point pitched milliseconds apart) reaches the light in the order it happened. A WLED device can only show one effect at a time, so nothing is gained by handling messages concurrently. One slow or unreachable device does delay the messages behind it, up to the 10 s HTTP timeout per message.

### Effect Profile

Effects are configured in a YAML file mapping `(system, severity)` combinations — optionally narrowed by message `tags` — to WLED effects:

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

**Rules are evaluated in the order they appear in the profile, and the first match wins.** Put a specific rule above any wildcard rule it overlaps with — a `*` rule declared first shadows everything below it for the same severity. `effects` stays a map keyed by rule name; only its order in the file matters.

When an effect has a `duration`, the light is turned off after that many seconds — unless a newer effect has been sent to the same endpoint in the meantime. The newer effect's own `duration` then decides when the light goes off.

### Matching on tags

A rule can also require message tags. `tags` is optional; a rule without it matches on `systems` and `severity` alone, exactly as before.

```yaml
effects:
  tabletennis-match:
    systems: [tabletennis]
    severity: [success]
    tags: [transition=match_won]
    fx: Fireworks
    duration: 10
    color: forest
  tabletennis-point-a:
    systems: [tabletennis]
    severity: [info]
    tags: [transition=point, side=a]
    fx: Solid
    duration: 1
    color: blue
  info:                   # declared after the tag rules, catches the rest
    systems: ["*"]
    severity: [info]
    fx: DJ Light
    color: ocean
```

- **All listed tags must be present (AND).** `[transition=point, side=a]` matches only messages carrying both.
- **Each entry matches one whole element** of the message's comma-separated `tags` field — `side=a` matches `match=36c17b30,set=2,transition=point,side=a` but not `side=ab`. Whitespace around elements is ignored; the comparison is case-sensitive.
- Declare tag rules **above** the wildcard rules they overlap with (first match wins).

> **Not the same as the notification-catcher.** homerun2-notification-catcher's `tags_contain` is a *substring* match that *ORs* the entries in its list. Here, entries are *whole elements* and *all* of them must match. Don't copy the rule shape between the two catchers unchanged.

Both dashboards show the tags a triggered rule matched on in their event timeline.

Available effects: Solid, Blink, Breathe, Wipe, Scan, Twinkle, Fireworks, Rainbow, Candle, Chase, Dynamic, Chase Rainbow, Aurora, Blurz, DJ Light

Color palettes: `sunset`, `beach`, `forest`, `ocean` — or single colors: `red`, `yellow`, `green`, `blue`, `white`

## Dashboards

Both the light-catcher and the WLED mock serve dashboards with the HOMERUN² design (Press Start 2P header, purple gradient, stuttgart-things footer).

| Dashboard | URL | Shows |
|-----------|-----|-------|
| **Light Catcher** | `http://localhost:8080/` | Light event timeline with severity, system, effect, color, matched tags |
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
| `CONSUMER_START_ID` | Where a **newly created** consumer group starts: `$` (only new messages), `0` (whole stream), or a stream ID. Ignored for an existing group | `$` |
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

The `kcl/` directory contains KCL modules that generate Kubernetes manifests (ServiceAccount, ConfigMap, Secret, Deployment, Service, HTTPRoute). No `Namespace` is emitted — the consuming Argo CD Application creates it via `CreateNamespace=true`.

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
- [homerun2-notification-catcher](https://github.com/stuttgart-things/homerun2-notification-catcher) (MS Teams / webhook consumer)
- [homerun-library](https://github.com/stuttgart-things/homerun-library) (shared library)
- [WLED Project](https://kno.wled.ge/)

## License

Apache 2.0
