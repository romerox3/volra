# Contributing to Volra

## Prerequisites

- Go 1.25+
- Docker (for integration tests)
- golangci-lint v2.10+

## Build

```sh
make build          # Build binary for current platform
make build-all      # Cross-compile for darwin/arm64 and linux/amd64
```

## Test

```sh
make test               # Unit tests (no Docker required)
make test-integration   # Integration tests (Docker required)
```

## Lint

```sh
make lint
```

## Project Structure

```
cmd/volra/          # CLI entry point and Cobra commands
internal/
  agentfile/        # Agentfile parsing and validation
  deploy/           # Deploy command (templates, generation, orchestration)
  docker/           # DockerRunner interface
  doctor/           # Doctor command (environment checks)
  output/           # Presenter, errors, warnings
  setup/            # Init command (project scanning, Agentfile creation)
  status/           # Status command (health reporting)
  testutil/         # Test helpers (mocks, golden files)
```

## Submitting Changes

1. Fork and create a feature branch from `main`
2. Write tests for new functionality
3. Ensure `make test` and `make lint` pass
4. Submit a pull request with a clear description
