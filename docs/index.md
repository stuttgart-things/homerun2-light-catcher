# Homerun2 Light Catcher

Redis Streams consumer microservice that triggers WLED light effects based on configurable YAML profiles. Visual alerting for the homerun2 ecosystem.

## Quick Start

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_ADDR` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | *(empty)* | Redis password |
| `REDIS_STREAM` | `messages` | Redis stream to consume |
| `CONSUMER_GROUP` | `homerun2-light-catcher` | Consumer group name |
| `CONSUMER_NAME` | hostname | Consumer name |
| `PROFILE_PATH` | `profile.yaml` | Path to WLED effect profile YAML |
| `HEALTH_PORT` | `8080` | Health endpoint port |
| `LOG_FORMAT` | `json` | Log format: `json` or `text` |
| `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `MOCK_WLED` | *(empty)* | Set to any value to start embedded WLED mock |
| `MOCK_WLED_PORT` | `9090` | Port for embedded WLED mock |

## Running Modes

The light-catcher can run with a real WLED device or with a mock for development/testing.

### Production — with real WLED device

Profile endpoints point to real WLED hardware. The light-catcher dashboard at `/` shows triggered events.

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
# Dashboard at http://localhost:8080
```

### Development — with embedded mock

Set `MOCK_WLED=true` to start the mock inside the same process. Profile endpoints point to `http://localhost:9090`.

```bash
MOCK_WLED=true \
MOCK_WLED_PORT=9090 \
PROFILE_PATH=tests/profile.yaml \
LOG_FORMAT=text \
REDIS_ADDR=localhost \
go run .
# Light-catcher dashboard at http://localhost:8080
# Mock dashboard at http://localhost:9090
```

### Kubernetes — with standalone mock

In Kubernetes, the mock runs as a separate deployment (`homerun2-wled-mock`). Profile endpoints point to the mock service DNS. Both services are exposed via HTTPRoute with their own dashboards.

```yaml
# profile endpoints point to mock service
endpoint: http://homerun2-wled-mock.homerun2.svc.cluster.local
```

Deploy via [Flux app](https://github.com/stuttgart-things/flux/tree/main/apps/homerun2) with `light-catcher` and `wled-mock` components.

### WLED Mock Standalone

Run the mock without Redis for UI testing:

```bash
go run ./cmd/wled-mock/
# Dashboard at http://localhost:8080
```

## Dashboards

Both services serve HTMX dashboards with the HOMERUN² design.

| Dashboard | Shows |
|-----------|-------|
| **Light Catcher** (`/`) | Event timeline with severity badges, system, effect, color |
| **WLED Mock** (`/`) | WLED state, segment visualization, event timeline with trigger context (severity, system, effect from light-catcher) |

## Endpoints

### Light Catcher

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Light event dashboard |
| GET | `/api/events` | JSON event list |
| GET | `/health` | Health check with build info |
| GET | `/healthz` | Health check (alias) |

### WLED Mock

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | WLED state dashboard |
| POST | `/json/state` | WLED JSON API (receives effects) |
| GET | `/json/state` | Current WLED state |
| GET | `/json/state/events` | State + event timeline |
| GET | `/api/state` | API state with request count |
| POST | `/api/reset` | Reset mock state |
| GET | `/healthz` | Health check |

## Container Images

Both images are built with [ko](https://ko.build) and published on every release.

| Image | Description |
|-------|-------------|
| `ghcr.io/stuttgart-things/homerun2-light-catcher:<tag>` | Main light-catcher service |
| `ghcr.io/stuttgart-things/homerun2-wled-mock:<tag>` | Standalone WLED mock server |

## Architecture

```
Redis Stream --> RedisCatcher --+--> LogHandler (structured slog)
                                +--> LightHandler
                                       |
                                 +-----+
                                 v
                           Profile YAML
                           (system + severity -> effect)
                                 |
                                 v
                           WLED HTTP API        Dashboard (/)
                           (real or mock)       event timeline
```

### Components

| Component | Description |
|-----------|-------------|
| `internal/catcher/` | Redis Stream consumer with Catcher interface |
| `internal/dashboard/` | Light event tracker and HTMX dashboard |
| `internal/profile/` | YAML profile loading and effect matching |
| `internal/wled/` | WLED HTTP client (send effects, turn off) |
| `internal/mock/` | WLED mock server with HTML dashboard |
| `internal/config/` | Env var loading, slog setup |
| `internal/handlers/` | Health endpoint |
| `cmd/wled-mock/` | Standalone WLED mock binary |

### Tech Stack

- **Language**: Go 1.25+
- **Consumer**: Redis Streams via `redisqueue` (consumer groups)
- **Library**: `homerun-library/v2` for shared types, Redis JSON, helpers
- **WLED**: HTTP client for WLED JSON API (`/json/state`)
- **Build**: ko (no Dockerfile), two images per release
- **CI**: Dagger modules, Taskfile, GitHub Actions
- **Deploy**: KCL manifests, Kustomize OCI, Flux app
