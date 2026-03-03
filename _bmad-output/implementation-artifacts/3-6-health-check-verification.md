# Story 3.6: Health Check Verification

Status: done

## Story

**As a** developer,
**I want** the deploy command to verify my agent is healthy after starting,
**So that** I know the deployment succeeded before seeing the summary.

## Acceptance Criteria (BDD)

```gherkin
Given docker compose up succeeded
When the health checker runs
Then it polls GET http://localhost:{port}{health_path}
  And it retries every 2 seconds
  And it has a default timeout of 60 seconds

Given the agent responds with HTTP 200 within timeout
When the health checker evaluates
Then it reports healthy status

Given the agent does not respond within timeout
When the health checker times out
Then it reports UserError E303: "Health check timed out after 60s" + fix
  And the fix suggests checking agent logs: docker logs {name}-agent
```

## Tasks / Subtasks

- [x] **Task 1: Create healthcheck.go** — WaitForHealth() with polling loop, 2s retry, 60s timeout
- [x] **Task 2: Create healthcheck_test.go** — 6 tests with httptest
- [x] **Task 3: Verify lint + test pass** — 191 total tests, 0 lint issues

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Completion Notes List

1. **WaitForHealth()** takes ctx, port, healthPath, name, Presenter — polls with net/http.
2. **Polling loop** uses context.WithDeadline (60s) and 2s retry via time.After.
3. **HTTP client** has 5s per-request timeout to avoid hanging on slow responses.
4. **Presenter.Progress()** called on each iteration ("Waiting for agent health...").
5. **Timeout error** returns UserError E303 with fix suggesting `docker logs {name}-agent`.
6. **Tests** use httptest.NewServer — immediate success, eventual success (3 attempts), timeout, connection refused, correct path, progress called.

### File List

| File | Action | Purpose |
|------|--------|---------|
| `internal/deploy/healthcheck.go` | Created | WaitForHealth polling function |
| `internal/deploy/healthcheck_test.go` | Created | 6 tests with httptest |
