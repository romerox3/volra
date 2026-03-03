# Story 4.1: Status Health Reporting

Status: done

## Story

**As a** developer with a deployed agent,
**I want** to run `volra status` to see the health of my agent and supporting services,
**So that** I can quickly diagnose issues without opening multiple tools.

## Tasks / Subtasks

- [x] **Task 1: Implement status.Run()** — Full pipeline: Load Agentfile → Docker check → Query containers → Probe health → Report
- [x] **Task 2: Container state querying** — docker compose ps --format json parsing
- [x] **Task 3: Health endpoint check** — Direct HTTP probe with 3s timeout
- [x] **Task 4: Create status_test.go** — 14 tests total (6 Run tests + 8 unit tests)
- [x] **Task 5: Verify lint + test pass** — 212 total tests, 0 lint issues

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Completion Notes List

1. **status.Run()** pipeline: Load Agentfile → docker info → docker compose ps → probe health → report.
2. **queryContainers()** parses docker compose ps --format json (one JSON object per line).
3. **probeHealth()** makes single HTTP GET with 3s timeout, returns bool.
4. **reportServices()** outputs Agent (with healthy/unhealthy label), Prometheus, Grafana states.
5. **findState()** matches services by suffix (e.g., "-agent", "-prometheus").
6. **E401** for missing Agentfile or no containers. **E402** for Docker not running.

### File List

| File | Action | Purpose |
|------|--------|---------|
| `internal/status/status.go` | Modified | Full status.Run() implementation |
| `internal/status/status_test.go` | Created | 14 tests |
