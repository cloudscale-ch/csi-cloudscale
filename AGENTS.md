# AGENTS.md — CSI-cloudscale-specific guidance for AI agents

This file is a supplement for AI agents working on csi-cloudscale, the
Container Storage Interface driver for cloudscale.ch. For general project
information, see [`README.md`](README.md). For contribution guidelines, see
[`.github/CONTRIBUTING.md`](.github/CONTRIBUTING.md).

## Related docs

- [`README.md`](README.md) — project overview, installation, volume parameters
- [`.github/CONTRIBUTING.md`](.github/CONTRIBUTING.md) — PR flow, required local checks
- [`CHANGELOG.md`](CHANGELOG.md) — version history

## What to run after a change

| You touched                                     | Run                           |
|-------------------------------------------------|-------------------------------|
| `*.go`                                          | `make lint-fix && make test`  |
| `Makefile`, `.golangci.yml`                     | `make lint`                   |
| `charts/csi-cloudscale/`                        | `make helm-template`          |

## Project structure

| Directory/File              | Purpose                                           |
|-----------------------------|---------------------------------------------------|
| `cmd/cloudscale-csi-plugin/` | Binary entry point and Dockerfile                |
| `driver/`                    | CSI driver implementation (Controller, Node, etc.) |
| `charts/csi-cloudscale/`     | Helm chart for deployment                        |
| `deploy/kubernetes/releases/`| Pre-rendered Kubernetes manifests                |
| `examples/kubernetes/`       | Example StorageClasses and PVCs                  |
| `test/kubernetes/`           | Integration tests                                |

## Key components

- **`driver/controller.go`** — CSI Controller service implementation.
  Handles all [CSI-defined API methods](https://github.com/container-storage-interface/spec). All non-read operations are guarded by
  `volumeLocks` to prevent concurrent modifications of the same volume.

- **`driver/node.go`** — CSI Node service implementation.

- **`driver/luks_util.go`** — LUKS encryption utilities.
  Provides functions for creating, opening, closing, and resizing LUKS
  encrypted volumes using `cryptsetup`.

- **`driver/mounter.go`** — Volume mounting logic.
  Wraps mount operations, filesystem creation (ext4, xfs), and resize
  operations.

- **`driver/identity.go`** — CSI Identity service.
  Reports driver name and capabilities.

- **`driver/volumelocks.go`** — Per-volume mutual exclusion.
  Simple string-keyed mutex map to prevent concurrent operations on the
  same volume.

## Testing conventions

- Unit tests live next to the code they exercise (`*_test.go` in `driver/`).
- Tests use [testify](https://github.com/stretchr/testify) for assertions.
- The fake cloudscale client is implemented in `driver/driver_test.go`.
- Run `make test` to execute all unit tests with race detection.
- Integration tests require a real cloudscale.ch account and are run via
  `make test-integration`. Do not run integration tests as an agent. Always ask the user to run them for you.

## Release process

See [`docs/releasing.md`](docs/releasing.md) for the full release process.

In brief:
1. Update `VERSION` file and run `make bump-version NEW_VERSION=vX.Y.Z`
2. Optionally bump chart version: `make bump-chart-version NEW_CHART_VERSION=1.X.Y`
3. Commit and push tag (`vX.Y.Z`)
4. GitHub Actions runs `release-chart.yml` to publish the Helm chart

## Dependency management

- Go dependencies: `go mod tidy` (run automatically in CI if `go.mod` / `go.sum`
  drift is detected).
- Tool binaries (`golangci-lint`, `govulncheck`) are installed on demand to
  `./bin/$(GOOS)-$(GOARCH)/`.
- Kubernetes module versions are pinned via `replace` directives in `go.mod`.
  Update them with `make update-k8s NEW_KUBERNETES_VERSION='1.34.0'`.
