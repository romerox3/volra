# Story 4.2: Docker Daemon Detection & Status CLI Wiring

Status: done

## Story

**As a** developer,
**I want** the status command to detect if Docker has restarted and report that agents need redeployment,
**So that** I understand why my agent is not responding after a Docker restart.

## Tasks / Subtasks

- [x] **Task 1: Implement daemon restart detection** — allStopped() checks for all exited/dead/created containers
- [x] **Task 2: Create cmd/volra/status.go** — Cobra subcommand (4-line RunE)
- [x] **Task 3: Add daemon detection tests** — TestRun_AllStopped_DaemonRestart + TestAllStopped_* unit tests
- [x] **Task 4: Verify lint + test pass** — 212 total tests, 0 lint issues

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Completion Notes List

1. **allStopped()** detects daemon restart — returns true when all containers are exited/dead/created.
2. **Warning** (not error) emitted: "All containers are stopped — Docker may have restarted" with fix: "Run: volra deploy".
3. **cmd/volra/status.go** follows the same pattern as init.go and deploy.go — 4-line RunE.
4. Implemented as part of Story 4.1 in the same status.Run() pipeline (step 4).

### File List

| File | Action | Purpose |
|------|--------|---------|
| `internal/status/status.go` | Modified (with 4.1) | allStopped() + daemon restart warning |
| `cmd/volra/status.go` | Created | Cobra status subcommand |
