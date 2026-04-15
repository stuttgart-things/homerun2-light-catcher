# CLAUDE.md

## Project

homerun2-light-catcher — Go microservice that consumes messages from Redis Streams and triggers WLED light effects based on configurable YAML profiles. Visual alerting for the homerun2 ecosystem.

## Tech Stack

- **Language**: Go 1.25+
- **Consumer**: Redis Streams via `redisqueue` (consumer groups)
- **Library**: `homerun-library/v3` for shared types, Redis JSON, helpers
- **WLED**: HTTP client for WLED JSON API (`/json/state`)
- **Build**: ko (`.ko.yaml`), no Dockerfile
- **CI**: Dagger modules (`dagger/main.go`), Taskfile
- **Deploy**: KCL manifests (`kcl/`), Kustomize, Kubernetes
- **Infra**: GitHub Actions, semantic-release

## Git Workflow

**Branch-per-issue with PR and merge.** Every change gets its own branch, PR, and merge to main.

### Branch naming

- `fix/<issue-number>-<short-description>` for bugs
- `feat/<issue-number>-<short-description>` for features
- `test/<issue-number>-<short-description>` for test-only changes
- `chore/<issue-number>-<short-description>` for infra/CI changes

### Commit messages

- Use conventional commits: `fix:`, `feat:`, `test:`, `chore:`, `docs:`
- End with `Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>` when Claude authored
- Include `Closes #<issue-number>` to auto-close issues

## Code Conventions

- No Dockerfile — use ko for image builds
- Config via environment variables, loaded once at startup
- Tests: `go test ./...` — unit tests must not require Redis; integration tests run via Dagger with Redis service
- Catcher interface pattern: pluggable backends (Redis consumer)
- Pluggable message handlers: LogHandler, LightHandler
- Logging: `log/slog` (structured JSON/text), NOT pterm

## Architecture

```
Redis Stream ──► RedisCatcher ──┬──► LogHandler (structured slog)
                                └──► LightHandler
                                          │
                                    ┌─────┘
                                    ▼
                              Profile YAML
                              (system + severity → effect)
                                    │
                                    ▼
                              WLED HTTP API
                              (or WLED Mock)
```

### Components

| Component | Description |
|-----------|-------------|
| `internal/catcher/` | Redis Stream consumer with Catcher interface |
| `internal/profile/` | YAML profile loading + effect matching |
| `internal/wled/` | WLED HTTP client (send effects, turn off) |
| `internal/mock/` | WLED mock server with HTML dashboard |
| `internal/config/` | Env var loading, slog setup |
| `internal/banner/` | Animated TUI startup banner |
| `internal/handlers/` | Health endpoint |
| `cmd/wled-mock/` | Standalone WLED mock binary |

## Key Paths

- `main.go` — entrypoint, signal handling, handler composition
- `internal/catcher/catcher.go` — RedisCatcher with JSON.GET payload resolution
- `internal/catcher/handlers.go` — LogHandler, LightHandler, timestamp validation
- `internal/profile/profile.go` — YAML loading, effect matching, color palettes
- `internal/wled/wled.go` — WLED HTTP client
- `internal/mock/mock.go` — WLED mock server with dashboard
- `dagger/main.go` — CI functions (Lint, Build, BuildImage, ScanImage, BuildAndTestBinary)
- `kcl/` — KCL deployment manifests
- `tests/profile.yaml` — test effect profile
- `Taskfile.yaml` — task runner for build/test/deploy/release

## Environment Variables

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
| `LOG_FORMAT` | `json` | Log format: json or text |
| `LOG_LEVEL` | `info` | Log level: debug, info, warn, error |
| `MOCK_WLED` | *(empty)* | Set to any value to start embedded WLED mock |
| `MOCK_WLED_PORT` | `9090` | Port for embedded WLED mock |

## Testing

```bash
# Unit tests (no Redis needed)
go test ./...

# Run WLED mock standalone
go run ./cmd/wled-mock/

# Run with embedded mock (local dev)
MOCK_WLED=true PROFILE_PATH=tests/profile.yaml LOG_FORMAT=text go run .

# Lint
task lint

# Build + test via Dagger
task build-test-binary

# Build + scan image
task build-scan-image-ko
```

## Reference Projects

- `homerun2-core-catcher` — sibling consumer service (same patterns)
- `homerun2-omni-pitcher` — sibling producer service
- `homerun-light-catcher` (GitLab, old) — original service being replaced
