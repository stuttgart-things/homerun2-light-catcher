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

### Run Locally (with embedded WLED mock)

```bash
MOCK_WLED=true \
MOCK_WLED_PORT=9090 \
PROFILE_PATH=tests/profile.yaml \
LOG_FORMAT=text \
REDIS_ADDR=localhost \
go run .
```

### Run WLED Mock Standalone

```bash
go run ./cmd/wled-mock/
```

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check with build info |
| GET | `/healthz` | Health check (alias) |

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
                           WLED HTTP API
                           (or WLED Mock)
```

### Components

| Component | Description |
|-----------|-------------|
| `internal/catcher/` | Redis Stream consumer with Catcher interface |
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
- **Build**: ko (no Dockerfile)
- **CI**: Dagger modules, Taskfile
- **Deploy**: KCL manifests, Kustomize, Kubernetes
