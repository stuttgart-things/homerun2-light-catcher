# CI/CD

## GitHub Actions Workflows

| Workflow | Trigger | Description |
|----------|---------|-------------|
| **CI - Dagger Build & Test** | push/PR to main | Lint + build + integration test with Redis |
| **Build, Push & Scan Container Image** | push/PR to main | ko image build, push, and Trivy scan |
| **Run Repository Linting** | push/PR to main | Repository-wide linting |
| **Release** | after image build succeeds, or manual | semantic-release, image staging, kustomize OCI push |

## Dagger Functions

The CI module in `dagger/main.go` provides:

| Function | Description |
|----------|-------------|
| `Lint` | Run golangci-lint on source code |
| `Build` | Compile Go binary |
| `BuildImage` | Build container image with ko |
| `ScanImage` | Scan image for vulnerabilities with Trivy |
| `BuildAndTestBinary` | Build binary and run integration tests with Redis service |

## Taskfile Commands

| Task | Description |
|------|-------------|
| `task lint` | Run Go lint via Dagger |
| `task build-test-binary` | Build and test binary with Redis via Dagger |
| `task build-output-binary` | Build project binary |
| `task build-scan-image-ko` | Build, push and scan container image |
| `task run-wled-mock` | Run WLED mock server locally |
| `task run-with-mock` | Run light-catcher with embedded WLED mock |
| `task trigger-release` | Trigger Release workflow on GitHub Actions |
| `task render-manifests-quick` | Render Kubernetes manifests |
| `task push-kustomize-base` | Push kustomize base as OCI artifact |
| `task release-local` | Full release pipeline (local, interactive) |
| `task release-github` | Full release pipeline (CI, non-interactive) |

## Release Process

Releases are managed by [semantic-release](https://semantic-release.gitbook.io/) with conventional commits:

- `feat:` triggers a minor version bump (e.g., 0.1.0 -> 0.2.0)
- `fix:` triggers a patch version bump (e.g., 0.1.0 -> 0.1.1)

The release pipeline:

1. Determines next version from commit history
2. Generates changelog
3. Creates GitHub release
4. Stages container image with version tag
5. Pushes Kustomize base as OCI artifact
