# Story 3.1: Dockerfile Generation

Status: done

## Story

**As a** Volra developer,
**I want** the deploy command to generate a Dockerfile from a template when mode is "auto",
**So that** the agent can be containerized without manual Docker knowledge.

## Acceptance Criteria (BDD)

```gherkin
Given an Agentfile with dockerfile: auto
When the Dockerfile generator runs
Then it produces a Dockerfile using the embedded template
  And the base image is python:{version}-slim
  And it handles requirements.txt vs pyproject.toml install strategies
  And the CMD uses the detected entry_point
  And the EXPOSE uses the configured port

Given an Agentfile with dockerfile: custom
When the Dockerfile generator runs
Then it skips generation (uses existing project Dockerfile)

Given a project with requirements.txt
When the Dockerfile generator runs
Then it COPY requirements.txt first for layer caching, then pip install, then COPY source

Given a project with only pyproject.toml
When the Dockerfile generator runs
Then it COPY all source, then pip install .
```

## Tasks / Subtasks

- [x] **Task 1: Create context.go** — TemplateContext struct + BuildContext() + detectPythonVersion + detectDeployEntryPoint
- [x] **Task 2: Create constants.go** — shared constants (JobHealth, JobMetrics, DatasourceName, NetworkName, OutputDir)
- [x] **Task 3: Create embed.go** — go:embed for templates/*.tmpl
- [x] **Task 4: Create templates/Dockerfile.tmpl** — two-branch template (requirements.txt vs pyproject.toml)
- [x] **Task 5: Create dockerfile.go** — GenerateDockerfile() + RenderDockerfile()
- [x] **Task 6: Create tests + golden files** — 12 tests, 2 golden files
- [x] **Task 7: Verify lint + test pass** — 147 total tests, 0 lint issues

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Completion Notes List

1. **TemplateContext** embeds Agentfile + PythonVersion, EntryPoint, HasRequirements for deploy-time metadata.
2. **BuildContext()** re-detects entry point and Python version at deploy time (stateless).
3. **detectPythonVersion** parses `requires-python` from pyproject.toml, strips quotes, extracts major.minor. Fallback "3.11".
4. **Dockerfile template** has two branches: requirements.txt (layer-cached) vs pyproject.toml (COPY all + pip install .).
5. **RenderDockerfile()** returns string for testability; **GenerateDockerfile()** writes to .volra/Dockerfile.
6. **staticFS** removed (unused until Prometheus/Grafana stories).
7. **Golden files** generated with UPDATE_GOLDEN=1 for requirements and pyproject variants.

### Change Log

| # | Change | Reason |
|---|--------|--------|
| 1 | Strip quotes from requires-python value | pyproject.toml uses `">=3.12"` with quotes |
| 2 | Removed staticFS from embed.go | Unused lint error; will be added in Story 3.3/3.4 |

### File List

| File | Action | Purpose |
|------|--------|---------|
| `internal/deploy/constants.go` | Created | Shared constants |
| `internal/deploy/context.go` | Created | TemplateContext + BuildContext + detection |
| `internal/deploy/embed.go` | Created | go:embed for templates |
| `internal/deploy/templates/Dockerfile.tmpl` | Created | Dockerfile template |
| `internal/deploy/dockerfile.go` | Created | GenerateDockerfile + RenderDockerfile |
| `internal/deploy/dockerfile_test.go` | Created | 12 tests + 2 golden files |
| `internal/deploy/testdata/golden/*.golden` | Created | Golden files |
