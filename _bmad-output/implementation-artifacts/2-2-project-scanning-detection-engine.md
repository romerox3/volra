# Story 2.2: Project Scanning & Detection Engine

Status: done

## Story

**As a** developer running `volra init`,
**I want** the system to auto-detect my agent's framework, entry point, port, health endpoint, and environment variables,
**So that** I get a pre-populated Agentfile with minimal manual editing.

## Acceptance Criteria (BDD)

```gherkin
Given a Python project with `langgraph` in requirements.txt
When the framework detector runs
Then it returns LangGraph (FR5)

Given a Python project with no known framework in dependencies
When the framework detector runs
Then it returns Generic (FR5)

Given a project with main.py, app.py, and server.py
When the entry point detector runs
Then it returns main.py (priority order: main.py > app.py > server.py) (FR6)

Given a project with only server.py
When the entry point detector runs
Then it returns server.py (FR6)

Given entry point code containing `uvicorn.run(app, port=8080)`
When the port detector runs
Then it returns 8080 (FR7)

Given entry point code with no port pattern
When the port detector runs
Then it returns 8000 as default and emits a warning (FR10)

Given entry point code with `@app.get("/health")`
When the health endpoint detector runs
Then it returns /health (FR8)

Given agent code with `os.environ["OPENAI_API_KEY"]` and `os.getenv("MODEL_NAME")`
When the env var detector runs
Then it returns ["OPENAI_API_KEY", "MODEL_NAME"] (FR9)

Given detection is partial (e.g., port not found)
When the scan completes
Then safe defaults are used and a UserWarning is emitted with the specific Agentfile field to override (FR10)
```

## Tasks / Subtasks

- [x] **Task 1: Create scan.go** — ScanResult type + ScanProject() orchestrator
- [x] **Task 2: Create detect_framework.go** — requirements.txt + pyproject.toml scanning
- [x] **Task 3: Create detect_entry.go** — Entry point priority detection (main.py > app.py > server.py)
- [x] **Task 4: Create detect_port.go** — Regex patterns on entry point code (uvicorn, flask, app.run)
- [x] **Task 5: Create detect_health.go** — Route decorator scanning (@app.get, @router.get)
- [x] **Task 6: Create detect_env.go** — os.environ/os.getenv pattern scanning
- [x] **Task 7: Create test fixtures** — 6 fixture projects (12 files)
- [x] **Task 8: Create tests** — 32 tests (framework, entry, port, health, env, scan integration, package name)
- [x] **Task 9: Verify lint + test pass** — 108 total tests, 0 lint issues

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Completion Notes List

1. **5 detectors** implemented as separate files: framework, entry point, port, health path, env vars.
2. **Framework detection** scans requirements.txt (line-by-line with package name extraction) and pyproject.toml (content scan for "langgraph").
3. **Entry point detection** uses priority order: main.py > app.py > server.py with warning on fallback.
4. **Port detection** uses 3 regex patterns: uvicorn.run, .run(port=), --port flag.
5. **Health detection** uses 2 regex patterns: @app.get/@app.route and @router.get/@router.route for health-related paths.
6. **Env var detection** scans all .py files with 3 patterns: os.environ[], os.environ.get(), os.getenv(). Returns sorted, deduplicated list.
7. **pyproject.toml scanning** simplified to content-based search after section-based parsing failed for `[project]` key-style dependencies.
8. **readFileContents** helper shared between scan.go (port/health detection) and detect_env.go.

### Change Log

| # | Change | Reason |
|---|--------|--------|
| 1 | Simplified pyproject.toml scanner | Section-based parsing didn't handle `dependencies = [...]` under `[project]` |

### File List

| File | Action | Purpose |
|------|--------|---------|
| `internal/setup/scan.go` | Created | ScanResult type + ScanProject() orchestrator |
| `internal/setup/detect_framework.go` | Created | Framework detection (requirements.txt + pyproject.toml) |
| `internal/setup/detect_entry.go` | Created | Entry point priority detection |
| `internal/setup/detect_port.go` | Created | Port regex extraction |
| `internal/setup/detect_health.go` | Created | Health endpoint detection |
| `internal/setup/detect_env.go` | Created | Env var scanning + readFileContents helper |
| `internal/setup/scan_test.go` | Created | 32 tests |
| `internal/setup/testdata/fixtures/` | Created | 6 fixture projects (12 files) |

## Developer Context

### ScanResult Type

```go
type ScanResult struct {
    Framework   agentfile.Framework
    EntryPoint  string
    Port        int
    HealthPath  string
    EnvVars     []string
    Warnings    []*output.UserWarning
}
```

### Detection Defaults & Warnings

| Detection | Default | Warning When |
|-----------|---------|-------------|
| Framework | generic | No known framework found |
| Entry Point | main.py | No main.py/app.py/server.py found |
| Port | 8000 | No port pattern in entry point code |
| Health Path | /health | No route decorator patterns found |
| Env Vars | [] (empty) | — (no warning for empty) |

### Detector Files

```
internal/setup/
├── scan.go               # ScanResult + ScanProject()
├── detect_framework.go   # detectFramework(dir)
├── detect_entry.go       # detectEntryPoint(dir)
├── detect_port.go        # detectPort(entryPointCode)
├── detect_health.go      # detectHealthPath(entryPointCode)
├── detect_env.go         # detectEnvVars(dir)
└── testdata/fixtures/
    ├── fastapi_project/  # main.py + requirements.txt (fastapi, uvicorn)
    ├── langgraph_project/ # agent.py + requirements.txt (langgraph)
    ├── pyproject_project/ # main.py + pyproject.toml
    ├── generic_project/  # main.py + requirements.txt (no framework)
    ├── multi_entry/      # main.py + app.py + server.py
    └── empty_project/    # Nothing detectable
```

### Regex Patterns

**Port detection:**
- `uvicorn\.run\(.*port\s*=\s*(\d+)`
- `\.run\(.*port\s*=\s*(\d+)`
- `--port\s+(\d+)` (CLI-style)

**Health endpoint detection:**
- `@app\.(get|route)\(\s*["'](/[^"']*health[^"']*)["']`
- `@router\.(get|route)\(\s*["'](/[^"']*health[^"']*)["']`

**Env var detection:**
- `os\.environ\[["']([A-Z_][A-Z0-9_]*)["']\]`
- `os\.environ\.get\(["']([A-Z_][A-Z0-9_]*)["']`
- `os\.getenv\(["']([A-Z_][A-Z0-9_]*)["']`

### Framework Detection Logic

1. Read requirements.txt → check for `langgraph` (case-insensitive line match)
2. Read pyproject.toml → check [project.dependencies] for `langgraph`
3. If found → LangGraph; otherwise → Generic

### Package Dependencies

- `internal/output` (for UserWarning)
- `internal/agentfile` (for Framework type)
- Go stdlib: `os`, `path/filepath`, `regexp`, `strings`, `bufio`, `strconv`
