# Story 1.2: Output System — Presenter, Error Types, Error Catalog

Status: done

## Story

**As a** Volra developer,
**I want** a Presenter interface with mode detection and structured error/warning types,
**So that** all commands produce consistent, actionable output following NFR22/NFR23.

## Acceptance Criteria (BDD)

```gherkin
Given the output package exists
When I inspect the Presenter interface
Then it has 4 methods: Progress(msg string), Result(msg string), Error(err error), Warn(w *UserWarning)
  And Progress, Error, Warn write to stderr
  And Result writes to stdout

Given the output package exists
When I call DetectMode()
Then it returns ColorMode when stdout is a TTY and NO_COLOR is unset
  And it returns NoColorMode when NO_COLOR is set
  And it returns PlainMode when TERM=dumb or columns < 60

Given a UserError{Code: "E101", What: "Docker is not installed", Fix: "Install Docker Desktop from https://docker.com"}
When Presenter.Error() receives it (via errors.As type assertion)
Then it renders: "[E101] Docker is not installed\n  → Install Docker Desktop from https://docker.com"

Given a generic error (not UserError)
When Presenter.Error() receives it
Then it renders with "[INTERNAL]" prefix

Given a UserWarning{What: "Port not detected", Assumed: "8000", Override: "Set port in Agentfile"}
When Presenter.Warn() renders it
Then it shows: what happened + what was assumed + how to override (NFR23)

Given the error catalog
When I inspect error codes
Then codes E101-E106 exist for doctor failures
  And codes E201-E206 exist for setup failures
  And codes E301-E306 exist for deploy failures
  And codes E401-E402 exist for status failures
  And codes E501-E502 exist for shared failures

Given a test needs to verify output
When using MockPresenter from testutil
Then all 4 methods record their calls for assertion
```

## Tasks / Subtasks

- [x] **Task 1: Create Mode detection** (AC: #2)
  - [x] `internal/output/mode.go` — Mode type, constants (ModeColor, ModeNoColor, ModePlain), DetectMode()
  - [x] `internal/output/mode_test.go` — 6 tests covering NO_COLOR, TERM=dumb, narrow terminal, default, precedence

- [x] **Task 2: Create error/warning types** (AC: #3, #4, #5)
  - [x] `internal/output/errors.go` — UserError struct (Code, What, Fix), UserWarning struct (What, Assumed, Override)
  - [x] UserError implements `error` interface
  - [x] `internal/output/errors_test.go` — 4 tests: Error() output, interface check, errors.As(), warning fields

- [x] **Task 3: Create error catalog** (AC: #6)
  - [x] `internal/output/catalog.go` — all error code constants
  - [x] E101-E106 (doctor), E201-E206 (setup), E301-E306 (deploy), E401-E402 (status), E501-E502 (shared)

- [x] **Task 4: Create Presenter interface + implementations** (AC: #1, #3, #4, #5)
  - [x] `internal/output/presenter.go` — Presenter interface + colorPresenter, noColorPresenter, plainPresenter
  - [x] Progress/Error/Warn → stderr, Result → stdout
  - [x] Error() uses errors.As() for UserError detection
  - [x] `internal/output/presenter_test.go` — 11 tests covering all modes, routing, formatting

- [x] **Task 5: Create MockPresenter in testutil** (AC: #7)
  - [x] `internal/testutil/mock_presenter.go` — implements Presenter, records all calls
  - [x] `internal/testutil/mock_presenter_test.go` — compile-time interface check + call recording test

- [x] **Task 6: Verify lint + test pass**
  - [x] `make test` — 22 tests passing
  - [x] `make lint` — 0 issues

## Dev Notes

### CRITICAL: Project Rename — MegaCenter → Volra

All planning artifacts reference "MegaCenter". Use "Volra" / "volra" in all implementation code.

### Architecture Compliance

**Presenter Interface (4 methods):**

```go
type Presenter interface {
    Progress(msg string)    // → stderr
    Result(msg string)      // → stdout
    Error(err error)        // → stderr, handles UserError via errors.As
    Warn(w *UserWarning)    // → stderr
}
```

**Mode Detection:**

```go
type Mode int
const (
    ModeColor   Mode = iota  // Default: ANSI colors + emoji
    ModeNoColor              // NO_COLOR env set: no colors, keep emoji
    ModePlain                // TERM=dumb or columns < 60
)

func DetectMode() Mode
```

**UserError:**

```go
type UserError struct {
    Code string  // E1xx=doctor, E2xx=setup, E3xx=deploy, E4xx=status, E5xx=shared
    What string  // What happened
    Fix  string  // How to fix it
}

func (e *UserError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.What)
}
```

**Rendering Format:**
- UserError: `[E101] Docker is not installed\n  → Install Docker Desktop from https://docker.com`
- Generic error: `[INTERNAL] {error.Error()}`

**UserWarning:**

```go
type UserWarning struct {
    What     string
    Assumed  string
    Override string
}
```

### Error Catalog

| Code | What | Fix |
|------|------|-----|
| E101 | Docker is not installed | Install Docker: https://docs.docker.com/get-docker/ |
| E102 | Docker is not running | Start Docker Desktop and try again |
| E103 | Docker Compose V2 not available | Update Docker Desktop or install docker-compose-plugin |
| E104 | Port {port} already in use | Stop the process using port {port} or change port in Agentfile |
| E105 | Python < 3.10 or not found | Install Python >= 3.10 from https://www.python.org/downloads/ |
| E106 | Insufficient disk space | Free up disk space. Volra needs at least 1GB. |
| E201 | No Python project detected | Volra requires requirements.txt or pyproject.toml |
| E202 | No entry point found | Create main.py or specify in Agentfile |
| E203 | Agentfile already exists | Use --force to overwrite |
| E204 | Reserved | Reserved |
| E205 | Reserved | Reserved |
| E206 | Reserved | Reserved |
| E301 | Docker is not running | Start Docker and try again |
| E302 | Docker build failed | Check Dockerfile and dependencies |
| E303 | Health check failed after {timeout}s | Check agent starts on {port}{health_path} |
| E304 | Agent container OOM killed | Increase Docker memory limit or optimize agent memory usage |
| E305 | .env not found | Copy .env.example to .env and fill in values |
| E306 | Reserved | Reserved |
| E401 | No deployment found | Run volra deploy first |
| E402 | Docker not running | Start Docker and try again |
| E501 | Invalid Agentfile field | Field-specific fix |
| E502 | Unsupported Agentfile version | Update Volra |

### File Organization

```
internal/output/
├── mode.go            # Mode type, DetectMode()
├── mode_test.go       # Detection tests
├── errors.go          # UserError, UserWarning structs
├── errors_test.go     # Error formatting tests
├── catalog.go         # Error code constants
├── presenter.go       # Presenter interface + 3 implementations
└── presenter_test.go  # Output routing tests
```

### MockPresenter (in testutil)

```go
type MockPresenter struct {
    ProgressCalls []string
    ResultCalls   []string
    ErrorCalls    []error
    WarnCalls     []*output.UserWarning
}
```

### Enforcement Guidelines

1. **No `fmt.Print*` outside `output/` package**
2. **No `os.Exit()` outside `cmd/`**
3. **No `log.*` calls anywhere**
4. **3 external deps only:** cobra, yaml.v3, testify — use stdlib for everything else

### Mode Behavior Table

| Condition | Mode | Error prefix | Check pass | Check fail |
|-----------|------|-------------|------------|------------|
| Default TTY | ModeColor | `❌` (red) | `✓` (green) | `✗` (red) |
| NO_COLOR set | ModeNoColor | `❌` | `✓` | `✗` |
| TERM=dumb / cols < 60 | ModePlain | `ERROR:` | `PASS:` | `FAIL:` |

### What This Story Does NOT Include

- No format helpers (FormatCheck, FormatService, FormatURL) — those come in later stories
- No actual command wiring — Presenter is injected in Story 1.5
- No signal handling / Ctrl+C — deferred
- No DockerRunner (Story 1.3)
- No MockDockerRunner (Story 1.3)
- No golden file testing (Story 1.3)

### References

- [architecture.md — Presenter Pattern]
- [architecture.md — Error Types]
- [architecture.md — Enforcement Guidelines]
- [epics.md — Story 1.2]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Completion Notes List

1. Removed old `internal/output/output.go` stub — replaced by 4 production files
2. **errcheck linter** required wrapping all `fmt.Fprintf`/`fmt.Fprintln` calls. Created internal `writef`/`writeln` helpers to cleanly discard write errors (non-actionable for terminal I/O)
3. Missing `fmt` import in errors_test.go caught during first test run — fixed immediately
4. Mode detection uses `COLUMNS` env var (not `os.Stdout.Fd()` stat) for portability in tests
5. Plain mode uses ASCII arrows `->` instead of Unicode `→` for dumb terminal compatibility
6. Error catalog uses string constants (not constructors) — construction happens at call sites in domain packages
7. 22 tests total: 6 mode, 4 error types, 11 presenter, 1 mock

### Change Log

| # | Change | Reason |
|---|--------|--------|
| 1 | Internal `writef`/`writeln` helpers | Satisfy errcheck linter for terminal writes |
| 2 | Plain mode uses `->` not `→` | ASCII compatibility for TERM=dumb |
| 3 | `WARNING:` prefix in plain mode | Replaces emoji for non-Unicode terminals |

### File List

| File | Action | Purpose |
|------|--------|---------|
| `internal/output/output.go` | Deleted | Replaced by production files |
| `internal/output/mode.go` | Created | Mode type, DetectMode() |
| `internal/output/mode_test.go` | Created | 6 mode detection tests |
| `internal/output/errors.go` | Created | UserError, UserWarning structs |
| `internal/output/errors_test.go` | Created | 4 error/warning type tests |
| `internal/output/catalog.go` | Created | 22 error code constants (E101-E502) |
| `internal/output/presenter.go` | Created | Presenter interface + 3 implementations |
| `internal/output/presenter_test.go` | Created | 11 presenter tests (routing, formatting, modes) |
| `internal/testutil/mock_presenter.go` | Created | MockPresenter for test assertions |
| `internal/testutil/mock_presenter_test.go` | Created | Interface compliance + call recording test |
