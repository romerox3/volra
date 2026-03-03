# Story 3.5: Deploy Orchestration

Status: done

## Story

**As a** developer,
**I want** the deploy command to execute `docker compose up` after generating artifacts,
**So that** all services start with a single command.

## Acceptance Criteria (BDD)

```gherkin
Given all artifacts are generated in .volra/
When the deploy orchestrator runs
Then it executes: docker compose -f .volra/docker-compose.yml -p {name} up -d --build
  And the agent container is rebuilt
  And monitoring containers are NOT rebuilt if unchanged

Given Docker is not running
When the deploy orchestrator attempts docker compose
Then it fails with UserError E301: "Docker is not running" + fix

Given the build fails (e.g., pip install error)
When docker compose up --build fails
Then it reports UserError E302: "Agent build failed" with the build log excerpt + fix
```

## Tasks / Subtasks

- [x] **Task 1: Create orchestrate.go** — Orchestrate() calling docker compose up via DockerRunner
- [x] **Task 2: Implement error mapping** — classifyComposeError parses output for E301/E302 patterns
- [x] **Task 3: Create orchestrate_test.go** — 7 tests with MockDockerRunner
- [x] **Task 4: Verify lint + test pass** — 185 total tests, 0 lint issues

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Completion Notes List

1. **Orchestrate()** takes ctx, DockerRunner, name, dir — builds compose command with -f, -p, up -d --build.
2. **classifyComposeError()** maps docker output to UserError: E301 (docker not running), E302 (build failed).
3. **extractExcerpt()** returns last N lines of build output for error messages.
4. **Unknown errors** fall through to generic `fmt.Errorf` wrap (no UserError).
5. Uses `docker.DockerRunner` interface for full testability with MockDockerRunner.

### File List

| File | Action | Purpose |
|------|--------|---------|
| `internal/deploy/orchestrate.go` | Created | Orchestrate + classifyComposeError + extractExcerpt |
| `internal/deploy/orchestrate_test.go` | Created | 7 tests (success, docker not running, build failed, unknown error, command args, excerpt) |
