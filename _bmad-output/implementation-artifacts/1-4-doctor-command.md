# Story 1.4: Doctor Command

Status: done

## Story

**As a** developer setting up Volra,
**I want** to run `volra doctor` to validate all prerequisites,
**So that** I know my system is ready before attempting to deploy an agent.

## Acceptance Criteria (BDD)

```gherkin
Given Docker is installed and running
When I run `volra doctor`
Then it checks: Docker installed, Docker running, Compose V2 available, Python >= 3.10 present, sufficient disk space (1GB)
  And each check shows pass/fail with a specific fix suggestion on failure
  And Volra version is reported in the output
  And exit code is 0

Given Docker is not installed
When I run `volra doctor`
Then the Docker check fails with error E101
  And exit code is 1

Given Docker is installed but not running
When I run `volra doctor`
Then the Docker-running check fails with error E102

Given Compose V2 is not available
When I run `volra doctor`
Then the check fails with error E103

Given Python < 3.10 or not found
When I run `volra doctor`
Then the check fails with error E105

Given disk space < 1GB available
When I run `volra doctor`
Then the check fails with error E106

Given ports 9090 and 3001 are in use
When I run `volra doctor`
Then it warns about port conflicts (warning, not failure)
```

## Tasks / Subtasks

- [x] **Task 1: Implement doctor.Run()** — orchestrator + 6 checks via SystemInfo interface
- [x] **Task 2: Unit tests** — 20 tests: all pass/fail combos, version parsing, multi-failure
- [x] **Task 3: Verify lint + test pass** — 51 total tests, 0 lint issues

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Completion Notes List

1. **SystemInfo interface** created for testability of non-Docker checks (Python, disk, ports). Architecture signature `Run(ctx, p, r)` extended to `Run(ctx, version, p, r, sys)` — `version` string for FR41, `sys` for mockable system checks.
2. Python detection uses `os/exec` directly (not DockerRunner) — architecture story notes allow "os/exec for Python (via DockerRunner pattern or direct)".
3. Disk space uses `syscall.Statfs` (stdlib, no external deps).
4. Port check uses `net.DialTimeout` with 500ms timeout — connection success = in use, refusal/timeout = free.
5. All 6 checks run independently — reports all results even when some fail.
6. Port warnings (E106-equivalent) use `Warn()` not `Error()` — do not cause exit code 1.
7. `conn.Close()` error explicitly ignored with `_ =` to satisfy errcheck.
8. 9 Python version parsing subtests cover edge cases (3.8, 3.9, 3.10, 3.11, 3.12, 4.0, garbage, empty).

### Change Log

| # | Change | Reason |
|---|--------|--------|
| 1 | Added `SystemInfo` interface | Testability for Python/disk/port checks without real system |
| 2 | Added `version` parameter to Run() | FR41 requires version reporting in doctor output |
| 3 | `_ = conn.Close()` | errcheck linter compliance |

### File List

| File | Action | Purpose |
|------|--------|---------|
| `internal/doctor/doctor.go` | Replaced stub | Full doctor implementation: Run() + 6 checks + SystemInfo |
| `internal/doctor/doctor_test.go` | Created | 20 unit tests with mocks |
