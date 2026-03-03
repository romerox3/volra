# Story 3.7: Deploy Command Pipeline & Output

Status: done

## Story

**As a** developer,
**I want** to run `volra deploy` and see a structured summary of what was deployed,
**So that** I know where to access my agent and dashboards.

## Acceptance Criteria (BDD)

```gherkin
Given a valid Agentfile and .env file exist
When I run `volra deploy`
Then it: loads Agentfile, generates all artifacts, runs docker compose up, verifies health
  And prints structured output listing:
    - Service status with ports (agent:{port}, Prometheus:9090, Grafana:3001)
    - Summary URLs: Agent URL + Grafana dashboard URL
    - Exact command to stop services

Given a valid Agentfile with env vars but no .env file
When I run `volra deploy`
Then it reports UserError E305 with fix: "Create .env file from .env.example"

Given the deploy command
When I inspect cmd/volra/deploy.go
Then Cobra RunE is 3-4 lines delegating to deploy.Run()
```

## Tasks / Subtasks

- [x] **Task 1: Implement deploy.Run()** — Full 7-step pipeline in deploy.go
- [x] **Task 2: Create cmd/volra/deploy.go** — Cobra subcommand (4-line RunE)
- [x] **Task 3: Create deploy_test.go** — 7 tests covering pipeline steps
- [x] **Task 4: Verify lint + test pass** — 198 total tests, 0 lint issues

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Completion Notes List

1. **deploy.Run()** pipeline: Load Agentfile → Validate .env → BuildContext → Generate artifacts → Orchestrate → WaitForHealth → Summary.
2. **7 artifacts generated**: Dockerfile (auto mode only), docker-compose.yml, prometheus.yml, alert_rules.yml, 4 Grafana files.
3. **E305 error** when Agentfile declares env vars but .env is missing — with fix pointing to .env.example.
4. **Dockerfile skipped** in custom mode (af.Dockerfile == DockerfileModeCustom).
5. **Summary output** via Presenter.Result(): Agent URL, Grafana URL, Prometheus URL, Stop command.
6. **cmd/volra/deploy.go** follows exact same pattern as init.go — 4-line RunE delegating to deploy.Run().

### Change Log

| # | Change | Reason |
|---|--------|--------|
| 1 | Removed unnecessary fmt.Sprintf on static strings | staticcheck S1039 lint error |

### File List

| File | Action | Purpose |
|------|--------|---------|
| `internal/deploy/deploy.go` | Modified | Full pipeline Run() function |
| `internal/deploy/deploy_test.go` | Created | 7 tests for pipeline |
| `cmd/volra/deploy.go` | Created | Cobra deploy subcommand |
