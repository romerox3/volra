# Contributing to Volra

## Prerequisites

- Go 1.25+
- Docker (for E2E and integration tests)
- golangci-lint v2.10+
- Python 3.10+ (for volra-observe development)

## Build

```sh
make build          # Build binary for current platform
make build-all      # Cross-compile for darwin/arm64, linux/amd64, linux/arm64
```

## Test

```sh
make test               # Unit tests (no Docker required)
make e2e                # E2E tests Phase 1+2 (no Docker required)
make e2e-deploy         # E2E tests Phase 3+4 (Docker required)
make test-integration   # Integration tests (Docker required)
make lint               # Run linters
```

For the volra-observe Python package:

```sh
cd volra-observe
pip install -e ".[dev]"
pytest tests/ -v
```

## Project Structure

```
cmd/volra/              # CLI entry point and Cobra commands
internal/
  agentfile/            # Agentfile parsing and validation
  deploy/               # Deploy command (templates, generation, orchestration)
  docker/               # DockerRunner interface
  doctor/               # Doctor command (environment checks)
  mcp/                  # MCP server (JSON-RPC protocol, tool handlers)
  output/               # Presenter (color, plain, JSON), errors, warnings
  setup/                # Init command (project scanning, Agentfile creation)
  status/               # Status command (health reporting)
  templates/            # Embedded quickstart templates (basic, rag, conversational)
  testutil/             # Test helpers (mocks, golden files)
volra-observe/          # Python package: framework-agnostic LLM observability
docs/                   # Additional documentation (MCP integration guide)
examples/               # Example agents for E2E testing
tests/e2e/              # End-to-end test suite
```

## Submitting Changes

1. Fork and create a feature branch from `main`
2. Write tests for new functionality
3. Ensure `make test` and `make lint` pass
4. Submit a pull request with a clear description
