---
stepsCompleted: [step-01-init, step-02-context, step-03-starter, step-04-decisions, step-05-patterns, step-06-structure, step-07-validation, step-08-complete]
lastStep: 8
status: 'complete'
completedAt: '2026-03-02'
inputDocuments:
  - '_bmad-output/planning-artifacts/prd.md'
  - '_bmad-output/planning-artifacts/product-brief-MegaCenter-2026-03-02.md'
  - '_bmad-output/brainstorming/brainstorming-session-2026-03-02.md'
  - 'guide.md'
workflowType: 'architecture'
executionMode: 'GENERATE'
project_name: 'MegaCenter'
user_name: 'Antonio'
date: '2026-03-02'
---

# Architecture Decision Document

_This document builds collaboratively through step-by-step discovery. Sections are appended as we work through each architectural decision together._

## Project Context Analysis

_5 rounds of Party Mode validation. 30 adjustments proposed, consolidated to 18 net effective changes. Key convergence: Go-idiomatic package structure aligned 1:1 with CLI commands, resource constraint as primary architecture driver._

### Input Documents Processed

| Document | Status | Architectural Signals |
|----------|--------|----------------------|
| PRD (prd.md) | Primary input — 41 FRs, 23 NFRs | All architectural signals extracted below |
| Product Brief | Subsumed by PRD | Vision and scope fully captured in PRD |
| Brainstorming Session | Subsumed by PRD | 135 ideas and 10 clusters distilled into PRD scope decisions — no additional architectural signals |
| guide.md (Framework Research) | Context only | Informs detection strategy (why LangGraph is first specific framework). No direct architectural implications beyond PRD's FR5 |

### Requirements Overview

**41 Functional Requirements** mapped to CLI commands:

| Command / Area | FRs | Scope |
|----------------|-----|-------|
| **doctor** | FR1-FR3 | Environment pre-flight checks, fix suggestions, exit codes |
| **init** | FR4-FR15 | Project scanning, framework/port/health detection, Agentfile generation, .env.example/.gitignore management |
| **deploy** | FR16-FR26, FR31-FR37 | Stack generation (Dockerfile, docker-compose, Prometheus, Grafana), deployment execution, health verification, error reporting, dashboard provisioning, probe metrics config, anonymous access |
| **status** | FR27-FR30 | Health check, service state reporting, Docker daemon detection. Read-only — does NOT own deployment lifecycle |
| **Installation** | FR38-FR41 | Build/release pipeline — not a runtime module (see Build & Distribution Artifacts) |

**Ownership clarification (Party Mode R1):** FR25 (re-deploy for updates) belongs to deploy — Generator owns the full deployment lifecycle. Status is strictly a read-only query of current state.

### NFR Architecture Drivers

**23 NFRs** — architecture-shaping implications:

| NFR | Driver | Architectural Implication |
|-----|--------|--------------------------|
| NFR1 | CLI < 5s | Go single binary, no startup overhead, no runtime deps |
| NFR2-3 | Deploy timing | Docker layer caching strategy, parallel service startup |
| NFR5 | 15s scrape interval | Probe config is static, generated at deploy time |
| NFR7 | 95% reliability | Defensive handling at every external boundary (see Integration Points) |
| NFR8 | Cross-platform | Template-based generation, no platform-conditional logic in artifacts |
| NFR9 | Data survives rebuilds | Named Docker volumes for Prometheus data (see Lifecycle Separation) |
| NFR10 | Artifacts functional | Template validation as first-class architectural concern — golden file testing pipeline |
| NFR12 | Zero-dep binary | Go cross-compilation, CGO_ENABLED=0 |
| NFR14 | Multi-arch Docker | Dockerfile must produce functional images on AMD64 + ARM64 — base image selection strategy needed |
| NFR15 | Terminal compat | Output abstraction layer (Presenter), NO_COLOR/TERM=dumb support, 60 col minimum |
| NFR16 | No secrets in artifacts | env_file reference in docker-compose, not inline values |
| NFR20 | No telemetry | No analytics SDK, no outbound network calls from CLI |
| NFR21 | < 500MB RAM | Container sizing strategy needed — Prometheus retention flags, Grafana plugin policy |
| NFR22-23 | Error/warning format | Structured error types with "what happened + fix" message templates |

### Scale & Complexity Assessment

| Dimension | Assessment |
|-----------|-----------|
| **Type** | CLI tool + infrastructure generator |
| **Complexity** | Medium — 4 commands, file generation, Docker orchestration |
| **State model** | Zero CLI state — project directory is the persistent context. The Agentfile persists between init and deploy. `.megacenter/` is regenerated each deploy. Prometheus volumes persist across rebuilds (NFR9) |
| **Component count** | 7 Go packages (4 command + 2 shared + 1 wiring — see below) |
| **External dependencies** | Docker Engine, Docker Compose V2, Go standard library |
| **Generated artifact types** | 5 (Dockerfile, docker-compose.yml, prometheus.yml, Grafana dashboard JSON ×2) |
| **Highest-risk component** | deploy/ — largest package, Dockerfile generation flagged as highest complexity/fragility in PRD |

### Go Package Structure (Preliminary)

_Converged through 5 rounds of Party Mode. Principle: 1 package per CLI command, shared packages for cross-cutting models and output. Go idioms over abstractions — no custom frameworks._

| Package | Type | FRs | Responsibility |
|---------|------|-----|---------------|
| `cmd/` | Wiring | — | main.go + subcommand parsing (flag parsing, calls domain packages). No business logic |
| `pkg/doctor/` | Command | FR1-FR3 | Environment checks, fix suggestions, exit codes |
| `pkg/init/` | Command | FR4-FR15 | Scan project → build Agentfile model → write Agentfile + .env.example + .gitignore. Pipeline: scan → structure → persist |
| `pkg/deploy/` | Command | FR16-FR26, FR31-FR37 | Read Agentfile → generate artifacts → execute docker compose → verify health. Internal structure by file: `dockerfile.go`, `compose.go`, `prometheus.go`, `grafana.go`, `orchestrate.go`, `healthcheck.go` |
| `pkg/status/` | Command | FR27-FR30 | Query Docker for container states, health check, daemon detection. Read-only |
| `pkg/agentfile/` | Shared | Cross-cutting | Parse, validate, apply defaults, schema version check. Used by init (write), deploy (read), status (read) |
| `pkg/output/` | Shared | Cross-cutting | Unified Presenter — the ONLY point of emission to stdout. Formatting, colors, NO_COLOR/TERM=dumb, progress indicators, structured errors per NFR22-23. No other package writes directly to stdout |

**deploy/ complexity note:** This is the largest package with 6 internal files covering 4 types of artifact generation + orchestration + health verification. Internal file discipline documented here; detailed design in subsequent architecture steps.

### Technical Constraints

| # | Constraint | Source | Architecture Impact |
|---|-----------|--------|-------------------|
| **0** | **1 developer, 8 weeks** | PRD: Resource Reality | Every architectural decision passes the filter: "can one person implement, test, and maintain this?" Go idioms over abstractions. No custom frameworks. Patterns implementable with stdlib |
| 1 | Go single binary | PRD: Installation | Cross-compile for 3 targets (macOS ARM64, Ubuntu AMD64, Debian AMD64), embed templates |
| 2 | 3 containers per project | PRD: Scoping | Docker Compose as sole orchestrator |
| 3 | Probe-based metrics only | PRD: FR31, Scoping | No sidecar, no gateway, Prometheus scrapes health_path directly |
| 4 | Idempotent regeneration | PRD: Design Constraints | `.megacenter/` always fully regenerated, no merge. Generated artifacts are ephemeral |
| 5 | Python-only detection | PRD: Scoping | Detector hardcoded to Python patterns (v0.1) |
| 6 | Framework: generic + LangGraph | PRD: FR5 | Detection strategy extensible but only 2 implementations shipped |
| 7 | No plugins, no hooks | PRD: Design Constraints | Override = edit generated files. No user-facing extension points |
| 8 | v0.2 migration path | PRD: Scoping | v0.1 independent stacks → v0.2 shared infrastructure. Architecture must not block this transition |
| 9 | Topology variation point | Party Mode R2 | Templates must support standalone (v0.1) vs shared (v0.2) topology. v0.1 implements standalone only, but the template structure must accommodate the switch without rewrite |
| 10 | Artifact vs data lifecycle | Party Mode R4 | Generated artifacts (`.megacenter/`) are ephemeral — regenerated every deploy. Runtime data (Prometheus volumes) is persistent — survives rebuilds. Two distinct lifecycle domains that must not be conflated |
| 11 | Go idioms over abstractions | Party Mode R4 | No pipeline frameworks, no DI containers, no custom middleware chains. Functions calling functions. `if err != nil` is the error strategy. Standard library first, external deps only when unavoidable |

### External Integration Points

_Every external boundary is a potential failure point. NFR7 (95% reliability) requires defensive handling at each._

| Boundary | Packages | Interaction Pattern | Failure Mode |
|----------|----------|-------------------|-------------|
| **Docker Engine API** | doctor (check), deploy (orchestrate), status (query) | 3 distinct patterns: availability check, compose execution, container state query | Not installed, not running, permission denied, version incompatible |
| **Docker Compose CLI** | deploy | Shell-out to `docker compose` | Command not found, compose file invalid, build failure, OOM, timeout |
| **Filesystem** | init (write config), deploy (write artifacts, read Agentfile) | Read project files, write generated artifacts, modify .gitignore | Permission denied, disk full, read-only, path not found |
| **Network (health probes)** | deploy (verify), status (check) | HTTP GET to agent health_path | Connection refused, timeout, non-200 response, DNS resolution (localhost) |

### Testing Surface Areas

_Identified for downstream architecture step (testing strategy). Not designed here, but cataloged to prevent backtracking._

| Surface | Approach Hint | Packages Affected |
|---------|--------------|-------------------|
| **Template output validation** | Golden file tests — expected output vs actual for known Agentfile inputs | deploy/ |
| **Project detection accuracy** | Fixture projects (real directory structures) as test inputs | init/ |
| **Docker integration** | Integration tests with real Docker vs mocked Docker client | deploy/, status/, doctor/ |
| **Output formatting** | Snapshot tests for terminal output (with/without color, narrow terminals) | output/ |
| **Agentfile parsing** | Unit tests for parse, validate, defaults, edge cases (missing fields, bad YAML) | agentfile/ |

### Build & Distribution Artifacts

_Not runtime components, but project artifacts that require design, maintenance, and testing._

| Artifact | Description | Notes |
|----------|-------------|-------|
| **Go binary ×3** | macOS ARM64, Ubuntu AMD64, Debian AMD64 | Cross-compiled with CGO_ENABLED=0. CI/CD pipeline (deferred to architecture) |
| **Install script** | `curl \| sh` — detects OS/arch, downloads correct binary, places in PATH | Bash script, lives in repo, tested on 3 targets |
| **SHA256 checksums** | Per-binary integrity verification | [SHOULD] — on cut list, generated by CI alongside binaries |

### Cross-Cutting Concerns

| Concern | Description | Implementation |
|---------|-------------|---------------|
| **User Communication** | All user-facing output: progress, success, errors, warnings | `pkg/output/` Presenter — single point of emission. Enforces NFR15 (NO_COLOR, TERM=dumb, 60 cols) and NFR22-23 (error/warning format) |
| **Agentfile Lifecycle** | Parse, validate, defaults, schema version | `pkg/agentfile/` shared package — consumed by init (write), deploy (read), status (read) |
| **Docker Interaction** | 3 patterns: check (doctor), orchestrate (deploy), query (status) | Utility functions per pattern, not a generic abstraction. Mockable at boundary for testing |
| **File System Operations** | Write generated artifacts, manage .gitignore, .env.example | Contained within init/ and deploy/ — no shared FS abstraction needed |

### Command Execution Model

_Each CLI command follows a conceptual pipeline: stages execute in sequence, data flows forward, errors halt the pipeline and report which stage failed. This is a documentation pattern, NOT a framework — implemented as sequential function calls in Go._

| Command | Pipeline Stages |
|---------|----------------|
| **doctor** | Check Docker → Check Compose → Check ports → Check Python → Check disk → Report |
| **init** | Validate directory → Scan project → Detect framework/port/health → Build Agentfile model → Write Agentfile → Write .env.example → Update .gitignore → Report |
| **deploy** | Read Agentfile → Validate → Generate Dockerfile → Generate docker-compose.yml → Generate prometheus.yml → Generate Grafana dashboards → Execute docker compose up → Health check → Report |
| **status** | Read Agentfile → Query Docker containers → Check health endpoint → Report |

## Starter Template Evaluation

_3 rounds of Party Mode. 14 adjustments consolidated. Key decisions: Cobra (without Viper), internal/ over pkg/, static dashboards with Grafana native variables, Makefile over GoReleaser._

### Technology Domain

**Go CLI tool + infrastructure generator.** No web framework starters apply. Decisions center on: CLI framework, template strategy, project structure, dependency management, and build tooling.

### Options Evaluated

| Option | Description | Verdict |
|--------|-------------|---------|
| **A: Cobra + go:embed + Standard Layout** | Industry-standard CLI framework, stdlib templates, idiomatic Go structure | **Selected** |
| B: Minimal (pflag + manual routing) | Zero framework, POSIX flags standalone | Rejected — saves 1 dep but costs ~2-3 days of boilerplate. Violates constraint #0 |
| C: stdlib flag + os.Args | Pure standard library | Rejected — stdlib `flag` does not support POSIX `--double-dash` flags (PRD requirement) |

### Selected Stack: Cobra + go:embed + internal/ Layout

**Rationale:** Constraint #0 (1 dev, 8 weeks) is the deciding factor. Cobra eliminates boilerplate for subcommand routing, help text, flag parsing, and shell completion. For 4 commands it's "just enough framework" — neither over- nor under-engineering.

#### CLI Framework: Cobra (without Viper)

**Why Cobra:**
1. **PersistentPreRun** (primary justification) — shared pre-checks (e.g., Docker running?) before deploy/status without code duplication
2. Subcommand routing for 4 commands with automatic help generation
3. POSIX/GNU flag parsing via built-in pflag
4. Shell completion for bash/zsh/fish/PowerShell (free, not critical for 4 commands but nice)
5. `--version` flag automatic

**Explicitly NOT included: Viper.** MegaCenter has no config file system. The Agentfile is parsed directly with yaml.v3 in `internal/agentfile/`. Viper would add ~15 transitive dependencies for zero benefit. Any tutorial suggesting Viper integration should be ignored.

#### Template & Static Asset Strategy

Two categories of embeddable files, with distinct processing strategies:

| Category | Directory | Processing | Contents |
|----------|-----------|-----------|----------|
| **Templates** (processed) | `templates/` | `text/template` stdlib with `template.ParseFS` | Dockerfile.tmpl, docker-compose.yml.tmpl, prometheus.yml.tmpl, datasource.yml.tmpl |
| **Static assets** (copied) | `static/` | `go:embed` → copy to output, no processing | Grafana dashboard JSONs, Grafana provisioning (dashboard provider) |

**Why dashboards are static, not templates:**
- Grafana dashboard JSON is deeply nested (200-400 lines). Go templates (`{{ }}`) inside JSON (`{}`) creates an escaping nightmare
- Grafana provides native template variables (`$agent_name`) — panel queries reference these variables, defined once in the dashboard JSON
- The only dynamic value MegaCenter needs to inject is the Prometheus datasource URL, which is handled via the datasource provisioning YAML (a template), not the dashboard JSON itself
- Static dashboards only need `json.Valid()` tests, not golden file testing — simpler to maintain

**For files needing minimal injection** (not full template logic): `strings.NewReplacer()` with `${PLACEHOLDER}` convention. One call, explicit, debuggable.

#### Project Structure

```
megacenter/
├── cmd/
│   └── megacenter/
│       └── main.go                    # Entry point, Cobra root command, subcommand registration
├── internal/
│   ├── doctor/                        # FR1-FR3: environment checks
│   │   └── testdata/                  # Fixtures: mock system states
│   ├── setup/                         # FR4-FR15: scan, detect, write Agentfile
│   │   └── testdata/                  # Fixtures: sample Python projects
│   ├── deploy/                        # FR16-FR26, FR31-FR37: generate + orchestrate
│   │   ├── dockerfile.go              # Dockerfile generation (highest risk)
│   │   ├── compose.go                 # docker-compose.yml generation
│   │   ├── prometheus.go              # prometheus.yml generation
│   │   ├── grafana.go                 # Dashboard + provisioning copy/generation
│   │   ├── orchestrate.go             # docker compose up, lifecycle management
│   │   ├── healthcheck.go             # Post-deploy health verification
│   │   └── testdata/                  # Golden files: expected generated artifacts
│   ├── status/                        # FR27-FR30: read-only health query
│   ├── agentfile/                     # Shared: parse, validate, defaults, schema version
│   │   └── testdata/                  # Edge cases: missing fields, bad YAML, version mismatch
│   └── output/                        # Shared: Presenter (stdout, colors, errors, progress)
├── templates/                         # go:embed — processed with text/template
│   ├── Dockerfile.tmpl
│   ├── docker-compose.yml.tmpl
│   ├── prometheus.yml.tmpl
│   └── grafana/
│       └── datasource.yml.tmpl        # Prometheus datasource provisioning
├── static/                            # go:embed — copied without processing
│   └── grafana/
│       ├── dashboards/
│       │   ├── overview.json          # Agent Overview dashboard (static)
│       │   └── detail.json            # Agent Detail dashboard (static)
│       └── provisioning/
│           └── dashboards.yml         # Dashboard provider config (static)
├── Makefile                           # Build, test, lint, cross-compile, checksums
├── .github/
│   └── workflows/
│       └── release.yml                # Cross-compile + GitHub Release + SHA256
├── .golangci.yml                      # Linter configuration
├── go.mod
└── go.sum
```

**Key structure decisions:**
- **`internal/` not `pkg/`**: MegaCenter is a CLI tool, not a library. `internal/` is enforced by the Go compiler — no external package can import these. Aligned with constraint #7 (no plugins, no hooks)
- **`internal/setup/` not `internal/init/`**: Avoids semantic collision with Go's special `func init()`. The CLI command is `megacenter init`, the package is `setup` — different names, same responsibility
- **`testdata/` per package**: Go convention. Each package owns its test fixtures. `go build` ignores `testdata/` directories
- **`deploy/` internal file structure**: 6 files for the largest package. Not sub-packages — Go idiom is flat packages with clear file boundaries

#### Docker Interaction Strategy

**No Docker SDK.** All Docker interaction via `os/exec.Command("docker", ...)` shell-out.

**Rationale:** The Docker Go SDK (`github.com/docker/docker/client`) would add 20+ transitive dependencies for 3 operations: check if running, compose up, query container status. `os/exec` is stdlib, zero deps, and sufficient for MegaCenter's needs.

**Testability pattern:** A minimal `DockerRunner` interface enables mocking without Docker:

```go
type DockerRunner interface {
    Run(ctx context.Context, args ...string) (string, error)
}
```

Production implementation wraps `os/exec`. Test implementation returns canned responses. Go-idiomatic dependency injection — one interface, two implementations, ~20 lines. Not a framework, just testable code.

#### Dependencies

| # | Dependency | Purpose | Scope |
|---|-----------|---------|-------|
| 1 | `github.com/spf13/cobra` | CLI framework (includes pflag for POSIX flags) | Runtime |
| 2 | `gopkg.in/yaml.v3` | Agentfile parsing (YAML) | Runtime |
| 3 | `github.com/stretchr/testify` | Test assertions, require, mock | Dev-only |
| — | `text/template`, `embed`, `os/exec`, `net/http`, `encoding/json` | Templates, embedding, Docker shell-out, health checks, JSON | Stdlib (free) |

**3 external dependencies. Everything else is Go standard library.** This is a deliberate constraint aligned with #0 (resource reality) and #11 (Go idioms over abstractions).

#### Build & Release Tooling

| Tool | Purpose | v0.1 | Future |
|------|---------|------|--------|
| **Makefile** | `build`, `test`, `lint`, `build-all`, `checksums` | Yes | Stays |
| **GitHub Actions** | CI (test + lint) + Release (cross-compile, upload binaries, SHA256) | Yes | Stays |
| **golangci-lint** | Aggregated linter | Yes | Stays |
| **GoReleaser** | Automated releases, homebrew, changelog | No — overkill for 3 binaries | v0.2 when homebrew needed |

**Cross-compile is 3 lines of bash:**
```bash
GOOS=darwin GOARCH=arm64 go build -o dist/megacenter-darwin-arm64 ./cmd/megacenter
GOOS=linux GOARCH=amd64 go build -o dist/megacenter-linux-amd64 ./cmd/megacenter
# + sha256sum dist/megacenter-* > dist/checksums.txt
```

**Makefile over GoReleaser for v0.1:** GoReleaser solves homebrew taps, Docker images, snapcraft, changelogs — none of which exist in v0.1. A Makefile with 5 targets is simpler, educational, and constraint #0 compliant. GoReleaser is a v0.2 upgrade when distribution channels expand.

### Initialization Command

```bash
mkdir megacenter && cd megacenter
go mod init github.com/antonioromero/megacenter
go get github.com/spf13/cobra@latest
go get gopkg.in/yaml.v3@latest
go get github.com/stretchr/testify@latest
```

**Note:** Project initialization using these commands should be the first implementation story.

## Core Architectural Decisions

### Category 1: Agentfile Schema & Validation

_2 rounds of Party Mode. 10 adjustments consolidated. Key decisions: Parse-Don't-Validate pattern, custom Go types with UnmarshalYAML, DNS label naming, two-level validation, re-deploy lifecycle._

#### Schema Definition

```yaml
# Agentfile — generated by megacenter init
version: 1

name: my-agent
framework: generic          # generic | langgraph
port: 8000
health_path: /health
env:
  - OPENAI_API_KEY
  - DATABASE_URL
dockerfile: auto            # auto | custom
```

**7 fields. YAML format.** Parsed with `gopkg.in/yaml.v3`.

| Field | Type | Required | Default | Validation |
|-------|------|----------|---------|-----------|
| `version` | integer | Yes | Error if missing | Must equal `1` in v0.1. If > supported → error with upgrade instruction. If < current → backward compat |
| `name` | string | Yes | Derived from directory (`filepath.Base`, sanitized) | DNS label: `^[a-z][a-z0-9-]*[a-z0-9]$`, 2-63 chars. Used as Docker Compose project name |
| `framework` | enum | Yes | `generic` | One of: `generic`, `langgraph`. Extensible in v0.2 |
| `port` | integer | Yes | `8000` | Range: 1-65535 |
| `health_path` | string | Yes | `/health` | Must start with `/` |
| `env` | list of strings | No | `[]` | Each entry non-empty, no duplicates. Names only — values live in `.env` |
| `dockerfile` | enum | Yes | `auto` | `auto` = MegaCenter generates. `custom` = uses `./Dockerfile` (must exist, error if not found) |

**Strict mode:** `yaml.Decoder.KnownFields(true)` — unknown fields cause parse error. Prevents silent typos (e.g., `heath_path` instead of `health_path`). Field validation is version-relative: version 1 accepts these 7 fields, version 2 will accept its expanded set.

#### Go Type Design: Parse-Don't-Validate

Custom types with `UnmarshalYAML` guarantee that a parsed Agentfile is always in a valid state. Consumers never need to re-validate enum fields.

```go
// Custom types with built-in validation
type Framework string

const (
    FrameworkGeneric   Framework = "generic"
    FrameworkLangGraph Framework = "langgraph"
)

func (f *Framework) UnmarshalYAML(value *yaml.Node) error {
    // Validates during parsing — invalid value never enters the struct
}

type DockerfileMode string

const (
    DockerfileModeAuto   DockerfileMode = "auto"
    DockerfileModeCustom DockerfileMode = "custom"
)

// Agentfile is the central configuration model
type Agentfile struct {
    Version    int            `yaml:"version"`
    Name       string         `yaml:"name"`
    Framework  Framework      `yaml:"framework"`
    Port       int            `yaml:"port"`
    HealthPath string         `yaml:"health_path"`
    Env        []string       `yaml:"env,omitempty"`
    Dockerfile DockerfileMode `yaml:"dockerfile"`
}
```

#### Two-Level Validation

| Level | Function | Scope | Failure Mode |
|-------|----------|-------|-------------|
| **Syntactic** | `Parse(reader)` | YAML syntax, known fields, field types, enum values via UnmarshalYAML, version compatibility | Hard error — cannot proceed |
| **Semantic** | `Validate(af)` | Port range, health_path starts with `/`, name is DNS label, no duplicate env vars, `dockerfile: custom` → `./Dockerfile` exists | Hard error with field-specific fix instruction |
| **Combined** | `Load(path)` | Calls Parse then Validate. Returns `*Agentfile, error` | Single entry point for all consumers |

**Version compatibility in Parse:**
- Agentfile `version` > CLI supported → `❌ Agentfile version 2 requires MegaCenter >= v0.2. You have v0.1.0. Update: curl -sSL https://...`
- Agentfile `version` < CLI current → accepted (backward compatible)

#### .env Lifecycle

**Generated by init:** `.env.example` with variable names and empty values, plus instructive header:

```
# MegaCenter — Environment Variables
# Copy this file to .env and fill in the values
# Values are injected into your agent container at deploy time

OPENAI_API_KEY=
DATABASE_URL=
```

Hint-style placeholders (e.g., `sk-your-key-here`) deferred to v0.2 — would require contextual detection beyond variable name scanning.

**Validated by deploy:** If Agentfile `env` is non-empty:
1. Check `.env` file exists → error if missing: `❌ Agentfile declares env vars but .env not found. Copy .env.example to .env and fill in values.`
2. Check each declared variable is present in `.env` → warning (not error) per missing var: `⚠️ OPENAI_API_KEY declared in Agentfile but not found in .env. Your agent may fail at runtime.`

Warning, not error — the variable may have a default in the application code.

#### Name as Docker Compose Project Name

`name` from Agentfile is used directly as Docker Compose project name (`docker compose -p {name}`). This means:

- Containers are named: `{name}-agent-1`, `{name}-prometheus-1`, `{name}-grafana-1`
- **Collision detection:** At deploy, if containers with the same project name already exist from a different directory → warning: `⚠️ A stack named '{name}' is already running. This will replace it. Use a different name in Agentfile to run both.`
- **Name change detection:** If `.megacenter/` exists with a previous deploy and the name has changed → warning: `⚠️ Previous stack '{old-name}' still running. Stop it with: docker compose -p {old-name} down`

#### Re-deploy Strategy (FR25)

Re-running `megacenter deploy` on an existing deployment:

1. Regenerate `.megacenter/` completely (idempotent, constraint #4)
2. Execute `docker compose -p {name} up -d --build` — forces agent image rebuild
3. Prometheus and Grafana containers are NOT recreated if their config hasn't changed (Docker Compose native behavior)
4. Prometheus data volumes survive rebuild (NFR9 — named volumes)
5. Health check on the new agent container

### Category 2: Error Handling & Output Strategy

_3 rounds of Party Mode. 15 adjustments consolidated to 12 net decisions. Key: Presenter as sole output channel, stderr/stdout separation, fix-and-re-run philosophy, context propagation._

#### Failure Philosophy

**No cleanup on failure. No rollback. Fix and re-run.**

MegaCenter has zero CLI state (Step 2 constraint). Generated artifacts in `.megacenter/` are idempotent — the next deploy overwrites them. Docker Compose manages container lifecycle. If a deploy fails mid-way, the user fixes the issue and runs `megacenter deploy` again. This eliminates an entire category of error handling complexity (rollback logic, partial state cleanup, transaction-like semantics).

#### Error & Warning Types

```go
// internal/output/errors.go

type UserError struct {
    Code string  // E1xx=doctor, E2xx=setup, E3xx=deploy, E4xx=status, E5xx=agentfile
    What string  // What happened
    Fix  string  // How to fix it
}

func (e *UserError) Error() string {
    return fmt.Sprintf("❌ %s. %s", e.What, e.Fix)
}

type UserWarning struct {
    What     string  // What happened
    Assumed  string  // What was assumed (empty for non-default warnings)
    Override string  // How to override
}
```

**Agentfile errors** use a field-specific format: `❌ Invalid Agentfile: {field} — {problem}. {fix}`

Example: `❌ Invalid Agentfile: port — 'banana' is not a valid port number. Use an integer between 1 and 65535.`

**Error codes** assigned only to the current catalog (~15 errors). Convention: E1xx doctor, E2xx setup, E3xx deploy, E4xx status, E5xx agentfile. Codes live in the struct but are NOT printed in v0.1 — they prepare for `--json` output in v0.2.

#### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (user or internal — no distinction) |
| 130 | Cancelled (SIGINT / Ctrl+C) |

Internal errors (unexpected bugs) are marked with `[INTERNAL]` prefix in the message but use the same exit code 1. The prefix makes bug reports identifiable without exposing a separate exit code.

#### Stream Routing

| Content | Stream | Rationale |
|---------|--------|-----------|
| Progress (checkmarks, spinners, step-by-step) | stderr | Does not pollute piped output |
| Errors (`❌`) | stderr | Unix convention |
| Warnings (`⚠️`) | stderr | Unix convention |
| Result (summary, URLs, tables) | stdout | Capturable: `megacenter deploy > result.txt` |

This makes MegaCenter pipe-friendly from v0.1. When `--json` arrives in v0.2, JSON goes to stdout clean.

#### Presenter Interface

```go
// internal/output/presenter.go

type Presenter interface {
    Progress(msg string)        // → stderr, with ✓/⚠️/spinner (mode-aware)
    Result(msg string)          // → stdout, final output
    Error(err *UserError)       // → stderr, ❌ formatted + records for exit
    Warn(w *UserWarning)        // → stderr, ⚠️ formatted
}
```

**4 methods. Single responsibility: output routing + formatting. No layout logic.**

- The Presenter is the ONLY point of emission to terminal. No package calls `fmt.Println` directly
- Layout (tables, check lists, summaries) is built by the caller as a string, then passed to `Progress()` or `Result()`
- Injected into every command via constructor — testable with a `TestPresenter` that accumulates messages

#### Format Helpers

Pure functions in `internal/output/` that produce formatted strings without writing to any stream:

```go
func FormatCheck(name string, passed bool, detail string) string
// → "✓ Docker installed" or "✗ Docker Compose — not found"

func FormatService(name string, port int, status string) string
// → "✓ agent         (port 8000)    healthy"

func FormatURL(label string, url string) string
// → "Dashboard:      http://localhost:3001"
```

Format helpers are mode-aware — they receive the output mode to produce emoji or plain text variants.

#### Output Modes

Detected once at Presenter construction. Flows to all output and format helpers.

```go
func DetectMode() Mode  // Called once in main.go

type Mode int
const (
    ModeColor   Mode = iota  // Default: ANSI colors + emoji
    ModeNoColor              // NO_COLOR env set: no colors, keep emoji
    ModePlain                // TERM=dumb: no colors, no emoji, "ERROR:" instead of "❌"
)
```

| Condition | Mode | Error format | Check format |
|-----------|------|-------------|-------------|
| Default terminal | ModeColor | `❌ Docker not running...` (red) | `✓ Docker installed` (green) |
| `NO_COLOR` set | ModeNoColor | `❌ Docker not running...` | `✓ Docker installed` |
| `TERM=dumb` | ModePlain | `ERROR: Docker not running...` | `PASS: Docker installed` |

#### Cobra Integration

```go
// cmd/megacenter/main.go
rootCmd.SilenceErrors = true   // Cobra does NOT print errors
rootCmd.SilenceUsage = true    // Cobra does NOT print usage on error
```

Cobra handles ONLY subcommand routing. All error presentation goes through the Presenter. This prevents double-printing and ensures consistent formatting across all commands.

#### Context Propagation & Signal Handling

```go
// Root command setup
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()
```

- `context.Context` propagated from root command to all domain packages
- `DockerRunner.Run(ctx, args...)` uses `exec.CommandContext` — kills child process on cancellation
- **Ctrl+C behavior:** Signal handler prints `⚠️ Cancelled. Run megacenter deploy to retry.` via Presenter, then exits with code 130
- Prevents zombie Docker processes when user interrupts a long build

#### Command Summary Lines

Every command emits a final summary via `Result()` (stdout):

| Command | Success | Failure |
|---------|---------|---------|
| doctor | `✅ All checks passed` | `❌ {n} check(s) failed` |
| setup (init) | `✅ Generated Agentfile for {name}` | Error-specific message |
| deploy | `✅ Deploy complete ({duration})` | Error-specific message |
| status | `✅ {name} is healthy` | `❌ {name} is unhealthy` |

#### Error Catalog

| Code | Command | What | Fix |
|------|---------|------|-----|
| E101 | doctor | Docker is not installed | Install Docker: https://docs.docker.com/get-docker/ |
| E102 | doctor | Docker is not running | Start Docker Desktop and try again |
| E103 | doctor | Docker Compose V2 not available | Update Docker Desktop or install docker-compose-plugin |
| E104 | doctor | Port {port} already in use | Stop the process using port {port} or change port in Agentfile |
| E201 | setup | No Python project detected | MegaCenter requires requirements.txt or pyproject.toml |
| E202 | setup | No entry point found | Create main.py or specify in Agentfile |
| E203 | setup | Agentfile already exists | Use --force to overwrite |
| E301 | deploy | Docker is not running | Start Docker and try again |
| E302 | deploy | Docker build failed | Check Dockerfile and dependencies. Logs: docker logs {name}-agent-1 |
| E303 | deploy | Health check failed after {timeout}s | Check agent starts on {port}{health_path}. Logs: docker logs {name}-agent-1 |
| E304 | deploy | Agent container OOM killed | Increase Docker memory limit or optimize agent memory usage |
| E305 | deploy | .env not found | Copy .env.example to .env and fill in values |
| E401 | status | No deployment found | Run megacenter deploy first |
| E402 | status | Docker not running | Start Docker and try again |
| E501 | agentfile | Invalid field: {field} — {problem} | Field-specific fix instruction |
| E502 | agentfile | Unsupported version {n} | Requires MegaCenter >= v{required}. Update: curl -sSL ... |

### Category 3: Docker Compose & Dockerfile Generation

_4 rounds of Party Mode. 19 adjustments consolidated to 12 net decisions. Key: Prometheus 3.x, Grafana 12.x, no Docker HEALTHCHECK, .dockerignore generated, Dockerfile limited to 2 dep managers, TemplateContext struct._

#### Container Images (Pinned)

| Service | Image | Version | Notes |
|---------|-------|---------|-------|
| Prometheus | `prom/prometheus` | `v3.10.0` | Prometheus 3.x — breaking changes from 2.x don't affect basic scrape config |
| Grafana | `grafana/grafana-oss` | `12.4.0` | OSS image (lighter, no enterprise features). NOT `grafana/grafana` |

**Update strategy:** Versions are pinned in the Go binary's embedded templates. Updated with each MegaCenter release. No user-configurable version override in v0.1.

#### docker-compose.yml Template

```yaml
# .megacenter/docker-compose.yml — Generated by MegaCenter. Do not edit.
# Regenerate with: megacenter deploy

name: {{.Name}}

services:
  agent:
    build:
      context: ..
      dockerfile: .megacenter/Dockerfile
    container_name: {{.Name}}-agent
    restart: unless-stopped
    ports:
      - "{{.Port}}:{{.Port}}"
    {{- if .Env}}
    env_file:
      - ../.env
    {{- end}}
    networks:
      - megacenter

  prometheus:
    image: prom/prometheus:v3.10.0
    container_name: {{.Name}}-prometheus
    restart: unless-stopped
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - ./alert_rules.yml:/etc/prometheus/alert_rules.yml:ro
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.retention.time=7d'
      - '--storage.tsdb.retention.size=500MB'
    networks:
      - megacenter

  grafana:
    image: grafana/grafana-oss:12.4.0
    container_name: {{.Name}}-grafana
    restart: unless-stopped
    ports:
      - "3001:3000"
    environment:
      - GF_AUTH_ANONYMOUS_ENABLED=true
      - GF_AUTH_ANONYMOUS_ORG_ROLE=Viewer
      - GF_AUTH_DISABLE_LOGIN_FORM=true
      - GF_DASHBOARDS_DEFAULT_HOME_DASHBOARD_PATH=/var/lib/grafana/dashboards/overview.json
    volumes:
      - ./grafana/dashboards:/var/lib/grafana/dashboards:ro
      - ./grafana/provisioning:/etc/grafana/provisioning:ro
    networks:
      - megacenter

volumes:
  prometheus-data:

networks:
  megacenter:
    driver: bridge
```

#### Compose Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Network** | Dedicated bridge `megacenter` | Isolation. Services reference each other by service name (`http://agent:{{.Port}}`) |
| **Project name** | `name: {{.Name}}` top-level (mandatory) | Without it, Compose uses directory name (`.megacenter`) — all projects would collide |
| **Agent build context** | `..` (parent directory) | Dockerfile is in `.megacenter/`, agent code is in project root |
| **Agent ports** | `{{.Port}}:{{.Port}}` | Expose agent port for direct host access |
| **Prometheus port** | `9090:9090` | Standard Prometheus port |
| **Grafana port** | `3001:3000` | External 3001 avoids conflict with apps on 3000. Internal Grafana stays on 3000 |
| **Prometheus volume** | Named volume `prometheus-data` | Persists data across re-deploys (NFR9). 7d retention + 500MB size limit (NFR21) |
| **Grafana volumes** | Bind mounts, read-only | Dashboards and provisioning regenerated each deploy — no persistence needed |
| **Grafana anonymous** | Enabled, Viewer role | FR36: no login screen. Localhost-only acceptable for v0.1 (NFR19) |
| **Grafana default dashboard** | Path-based via env var | FR36: opens directly to Overview dashboard |
| **Container naming** | `{{.Name}}-{service}` | Consistent, predictable, matches Docker Compose project naming |
| **Restart policy** | `unless-stopped` | Survives host reboot. Stops with explicit `docker compose down` |
| **env_file** | Conditional — only if Agentfile `env` is non-empty | Prevents Docker Compose error when `.env` doesn't exist for agents with no env vars |
| **No Docker HEALTHCHECK** | Removed from compose | Redundant with Prometheus probes. Deploy uses its own HTTP polling from host (`deploy/healthcheck.go`). Avoids curl/wget dependency in agent container |

#### Port Conflict Strategy

No auto-remap in v0.1. Fixed ports = predictable setup.

| Service | Default Port | If occupied |
|---------|-------------|-------------|
| Agent | From Agentfile (`port`) | Error E104 in doctor |
| Prometheus | 9090 | Error E104 in doctor |
| Grafana | 3001 | Error E104 in doctor |

#### .dockerignore Generation

Generated by `megacenter deploy` in the project root (NOT in `.megacenter/`) — `.dockerignore` is read relative to build context, which is `..` (project root).

**Rules:**
- If `.dockerignore` already exists → respect it, do not modify
- If not exists → generate with minimal set:

```
.megacenter/
.git/
.env
__pycache__/
*.pyc
.venv/
venv/
node_modules/
.pytest_cache/
```

Protects against slow builds from large `venv/` or `node_modules/` directories in the build context.

#### Dockerfile Template (auto mode)

The highest-risk generated artifact. Template is deliberately simple — two branches, three variables.

```dockerfile
# Generated by MegaCenter. Override with dockerfile: custom in Agentfile.
FROM python:{{.PythonVersion}}-slim

WORKDIR /app

{{if .HasRequirements -}}
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .
{{- else -}}
COPY . .
RUN pip install --no-cache-dir .
{{- end}}

EXPOSE {{.Port}}

CMD ["python", "{{.EntryPoint}}"]
```

**Two branches only:**

| Detected | Install Strategy | Layer Caching |
|----------|-----------------|---------------|
| `requirements.txt` exists | `COPY requirements.txt` → `pip install -r` → `COPY . .` | Optimized — deps cached until requirements.txt changes |
| `pyproject.toml` only | `COPY . .` → `pip install .` | Not optimized — any file change re-installs. Pragmatic for v0.1, `dockerfile: custom` for optimization |

**Explicitly NOT supported in auto mode:** Poetry, uv, conda, pipenv. If detected, init emits warning: `⚠️ Poetry detected. Run 'poetry export -f requirements.txt -o requirements.txt' for optimal Docker builds, or use dockerfile: custom in Agentfile.`

**ARM64 note:** `python:X.Y-slim` is multi-arch (Docker pulls correct variant automatically). If build fails due to C extension compilation on ARM64, error E302 includes hint: `Tip: Some Python packages lack ARM64 wheels. Try dockerfile: custom with --platform linux/amd64.`

#### Template Context

The Agentfile has 7 fields. The Dockerfile template needs 3 additional metadata fields detected at deploy time:

```go
// internal/deploy/context.go
type TemplateContext struct {
    agentfile.Agentfile          // Embedded — all 7 fields
    PythonVersion   string       // Detected from pyproject.toml/runtime.txt, fallback "3.11"
    EntryPoint      string       // Detected from scanner (main.py/app.py/server.py), fallback "main.py"
    HasRequirements bool         // requirements.txt exists in project root
}
```

**Why not store metadata in Agentfile?** Between `init` and `deploy`, the user may change files (rename app.py → main.py, add requirements.txt). Deploy always re-scans for fresh metadata. Stateless — aligned with zero-CLI-state philosophy.

**Why not add fields to Agentfile?** PRD scope rule: "The Agentfile has 6 fields — adding a 7th requires removing one or explicit justification." (Already used that budget for `version`.) PythonVersion and EntryPoint are build-time metadata, not user-facing configuration.

#### dockerfile: custom Path

When Agentfile has `dockerfile: custom`:
1. Deploy skips Dockerfile generation entirely
2. Deploy reads `./Dockerfile` from project root (validated in Agentfile semantic validation — error if not found)
3. Compose template uses `dockerfile: ../Dockerfile` instead of `.megacenter/Dockerfile`
4. All other generation proceeds normally (compose, prometheus, grafana)

This is the escape hatch for: Poetry/uv projects, multi-stage builds, non-standard project structures, ARM64-specific builds, any case where auto-generated Dockerfile is insufficient.

### Category 4: Prometheus & Grafana Configuration

_2 rounds of Party Mode. 10 adjustments consolidated to 8 net decisions. Critical bug fixes: corrected PromQL metrics (scrape_duration_seconds, not probe_duration_seconds), alert timing aligned with NFR11, HasMetricsEndpoint detection eliminated._

#### Shared Constants

Values that appear across multiple generated artifacts. Defined once in `internal/deploy/constants.go`:

```go
const (
    JobHealth      = "agent-health"    // prometheus.yml, alert_rules.yml, dashboards
    JobMetrics     = "agent-metrics"   // prometheus.yml
    DatasourceName = "Prometheus"       // datasource.yml, dashboards
    NetworkName    = "megacenter"       // docker-compose.yml
)
```

#### prometheus.yml Template

```yaml
# .megacenter/prometheus.yml — Generated by MegaCenter. Do not edit.
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - /etc/prometheus/alert_rules.yml

scrape_configs:
  - job_name: 'agent-health'
    metrics_path: {{.HealthPath}}
    scrape_interval: 15s
    scrape_timeout: 10s
    static_configs:
      - targets: ['agent:{{.Port}}']
        labels:
          agent: '{{.Name}}'

  - job_name: 'agent-metrics'
    metrics_path: /metrics
    scrape_interval: 15s
    scrape_timeout: 10s
    static_configs:
      - targets: ['agent:{{.Port}}']
        labels:
          agent: '{{.Name}}'
```

**Key decisions:**

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Scrape target | `agent:{{.Port}}` | Docker DNS resolution within bridge network |
| Two jobs always | `agent-health` + `agent-metrics` | No runtime detection of `/metrics`. If agent doesn't expose `/metrics`, Prometheus handles the failed scrapes gracefully (`up{job="agent-metrics"} == 0`). Simpler than deploy-time detection |
| scrape_timeout | 10s (NFR5) | Health endpoint not responding in 10s = unhealthy |
| scrape_interval | 15s (NFR5) | Default probe frequency |

**Prometheus automatically generates these metrics per scrape target (no agent instrumentation needed):**
- `up` → 1 if scrape succeeded (HTTP 200), 0 if failed
- `scrape_duration_seconds` → total time for the scrape request
- `scrape_samples_scraped` → number of metrics samples collected

**NOT available without agent instrumentation:** `probe_duration_seconds` (blackbox exporter only), `process_start_time_seconds` (requires client library).

#### alert_rules.yml (Static)

Lives in `static/`, copied without template processing. Uses Prometheus alerting template syntax `{{ $labels }}` which would conflict with Go template delimiters.

```yaml
# .megacenter/alert_rules.yml — Generated by MegaCenter.
groups:
  - name: agent-health
    rules:
      - alert: AgentDown
        expr: up{job="agent-health"} == 0
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: "Agent {{ $labels.agent }} is down"
          description: "Health check failing for more than 30 seconds."
```

**Alert timing (aligned with NFR11 — detection within 1 minute):**
- Worst case: agent goes down right after a successful probe
- +15s: next probe fires, fails → `up == 0`
- +30s: `for: 30s` satisfied → alert transitions to FIRING
- Total worst case: **~45 seconds** from failure to alert FIRING state

**No Alertmanager** — this alert rule evaluates in Prometheus and manifests as a visual indicator (red status) in the Grafana Overview dashboard. Notification delivery (Slack, email) is a v0.2 feature with Alertmanager.

#### Grafana Provisioning (All Static)

**Datasource** (`static/grafana/provisioning/datasources/datasource.yml`):

```yaml
apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: false
```

Static — URL is always `http://prometheus:9090` (Docker service name within bridge network). No variables needed.

**Dashboard provider** (`static/grafana/provisioning/dashboards/dashboards.yml`):

```yaml
apiVersion: 1
providers:
  - name: 'MegaCenter'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: true
    editable: false
    options:
      path: /var/lib/grafana/dashboards
```

#### Grafana Dashboards (Static JSON)

Dashboards are static JSON files with hardcoded job names. No Grafana template variables in v0.1 — one dashboard = one agent = one job. Template variables (dropdown selectors) come in v0.2 for multi-agent unified dashboards.

**Overview Dashboard** (`static/grafana/dashboards/overview.json`):

| Panel | Type | PromQL | Notes |
|-------|------|--------|-------|
| Agent Status | Stat | `up{job="agent-health"}` | Value mapping: 1 → "Healthy" (green), 0 → "Unhealthy" (red). FR34 visual alert indicator |
| Uptime (1h) | Gauge | `avg_over_time(up{job="agent-health"}[1h]) * 100` | Percentage of successful probes in last hour |
| Probe Latency | Gauge | `scrape_duration_seconds{job="agent-health"}` | Current probe response time. Labeled "Probe Latency" (FR37 honesty) |
| Last Seen | Stat | `timestamp(up{job="agent-health"} == 1)` | Timestamp of last successful probe. Freezes when agent is down — shows "last time it worked" |

**Detail Dashboard** (`static/grafana/dashboards/detail.json`):

| Panel | Type | PromQL | Notes |
|-------|------|--------|-------|
| Probe Latency Over Time | Time series | `scrape_duration_seconds{job="agent-health"}` | Line chart. Labeled "Probe Latency" |
| Health Status Timeline | State timeline | `up{job="agent-health"}` | Green (1) / Red (0) state chart over time |
| Uptime % (24h) | Gauge | `avg_over_time(up{job="agent-health"}[24h]) * 100` | 24-hour uptime percentage |
| Uptime % (7d) | Gauge | `avg_over_time(up{job="agent-health"}[7d]) * 100` | 7-day uptime percentage |

**Metrics honesty (FR37):** All panels clearly labeled as probe-based — "Probe Latency" not "Request Latency", "Health Checks" not "Request Count". Real user traffic metrics require agent-side instrumentation (v0.2).

**Bonus agent metrics (FR35):** If the agent exposes `/metrics`, Prometheus collects them via the always-present `agent-metrics` job. Users can explore these metrics in Grafana's Explore view and create custom panels. Pre-built bonus panels are not generated — manual dashboard editing for custom metrics is acceptable in v0.1.

#### Artifact Generation Summary

Final classification of all generated artifacts:

| Artifact | Category | Processing | Template Variables |
|----------|----------|-----------|-------------------|
| `Dockerfile` | Template | `text/template` | PythonVersion, Port, EntryPoint, HasRequirements |
| `docker-compose.yml` | Template | `text/template` | Name, Port, Env (conditional) |
| `prometheus.yml` | Template | `text/template` | Name, Port, HealthPath |
| `alert_rules.yml` | Static | Copy | None (uses Prometheus `{{ }}` syntax) |
| `grafana/dashboards/overview.json` | Static | Copy | None (hardcoded job names) |
| `grafana/dashboards/detail.json` | Static | Copy | None (hardcoded job names) |
| `grafana/provisioning/dashboards/dashboards.yml` | Static | Copy | None |
| `grafana/provisioning/datasources/datasource.yml` | Static | Copy | None |

**3 templates (processed) + 5 static files (copied) = 8 artifacts total.**

#### Updated TemplateContext

With `HasMetricsEndpoint` removed:

```go
type TemplateContext struct {
    agentfile.Agentfile          // Embedded — all 7 fields
    PythonVersion   string       // Detected, fallback "3.11"
    EntryPoint      string       // Detected, fallback "main.py"
    HasRequirements bool         // requirements.txt exists
}
```

3 metadata fields. Clean, no detection logic at deploy time beyond filesystem checks.

---

## Implementation Patterns & Consistency Rules

_Autonomous Party Mode — 3 rounds to convergence. 15 adjustments consolidated. All 6 agents confirmed no remaining conflict points. Key convergence: Cobra wiring-only pattern, exact Run() signatures, golden file testing with testutil helpers, build-tag integration test split._

### Go Naming Conventions

| Category | Convention | Example |
|----------|-----------|---------|
| Packages | Single lowercase word, aligned with CLI command | `doctor`, `setup`, `deploy`, `status`, `agentfile`, `output` |
| Files | Snake_case | `docker_runner.go`, `template_context.go` |
| Exported types | PascalCase, noun | `Agentfile`, `Presenter`, `DockerRunner`, `UserError` |
| Exported functions | PascalCase, verb-first | `Load()`, `Run()`, `DetectMode()` |
| Unexported | camelCase | `parseFramework()`, `renderTemplate()` |
| Constants | PascalCase (exported), camelCase (unexported) | `ExitDockerNotRunning`, `defaultPythonVersion` |
| Receiver letters | Single letter from type name | `a` = Agentfile, `p` = Presenter, `r` = DockerRunner, `tc` = TemplateContext |
| Test files | `*_test.go` co-located | `agentfile_test.go` next to `agentfile.go` |

**Constraint #11 in action:** Go idioms over abstractions. No interfaces unless we have 2+ implementations (Presenter and DockerRunner qualify — production and mock).

### Error Handling Patterns

```go
// EXPECTED errors (user can fix) — use UserError
return &output.UserError{
    Code:    "E201",
    Message: "Docker is not running",
    Hint:    "Start Docker Desktop or run: sudo systemctl start docker",
}

// UNEXPECTED errors (bugs, system failures) — wrap with context
return fmt.Errorf("reading Agentfile: %w", err)
```

**Rules:**
1. `UserError` for anything the user can act on — always include `Hint`
2. `fmt.Errorf` with `%w` for unexpected errors — preserve the chain
3. Never `log.Fatal()` or `os.Exit()` inside packages — only `cmd/` decides exit codes
4. Never `panic()` — return errors
5. Cobra's `SilenceErrors: true` on root command — we handle display via Presenter
6. Summary line before exit on failure: `p.Error(err)` renders code + message + hint

**Exit codes:**

| Code | Meaning | When |
|------|---------|------|
| 0 | Success | Command completed |
| 1 | General error | Unexpected failure |
| 2 | Agentfile error | Parse or validation failure |
| 3 | Docker error | Docker not running, compose failed |
| 4 | Network error | Port conflict, connectivity |

**Ctrl+C handling:** All `Run()` functions accept `context.Context`. Cobra provides a context that cancels on SIGINT. Long operations (Docker commands) pass the context to `exec.CommandContext`. No cleanup routines — fix-and-re-run philosophy.

### Testing Patterns

#### Test Split: Unit vs Integration

```makefile
test:              ## Run unit tests (no Docker required)
	go test ./internal/... -v -count=1

test-integration:  ## Run integration tests (Docker required)
	go test ./internal/... -v -count=1 -tags=integration

lint:              ## Run linters
	golangci-lint run ./...
```

- **Unit tests** (`*_test.go`): No build tag. Test pure logic — Agentfile parsing, template rendering, error formatting. Use mocks for DockerRunner and Presenter.
- **Integration tests** (`*_integration_test.go`): `//go:build integration` tag. Require Docker. Test actual `docker compose up/down`, generated artifacts against real Docker.

#### Golden File Testing

For template output verification — rendered Dockerfile, docker-compose.yml, prometheus.yml:

```go
// internal/testutil/golden.go
func AssertGolden(t *testing.T, got string, goldenFile string) {
    t.Helper()
    if os.Getenv("UPDATE_GOLDEN") == "1" {
        os.WriteFile(goldenFile, []byte(got), 0644)
        return
    }
    expected, err := os.ReadFile(goldenFile)
    require.NoError(t, err, "golden file %s not found — run with UPDATE_GOLDEN=1", goldenFile)
    assert.Equal(t, string(expected), got)
}
```

Usage: `UPDATE_GOLDEN=1 go test ./internal/deploy/...` regenerates golden files. CI runs without the flag — fails if output drifts.

Golden files live next to tests: `internal/deploy/testdata/*.golden`

#### Fixture Projects

End-to-end validation of the full generation pipeline:

```
internal/deploy/testdata/
├── fixtures/
│   ├── minimal/           # Only required fields
│   │   ├── Agentfile
│   │   └── expected.yaml  # Expected generated file list + checksums
│   ├── full/              # All optional fields populated
│   │   ├── Agentfile
│   │   ├── .env
│   │   └── expected.yaml
│   └── pyproject/         # pyproject.toml instead of requirements.txt
│       ├── Agentfile
│       ├── pyproject.toml
│       └── expected.yaml
```

`expected.yaml` lists generated files and their SHA-256 checksums. Test loads fixture, runs generation, compares output against expectations.

#### Cross-Artifact Consistency Test

A dedicated test verifies that shared constants (image versions, job names, port numbers) are consistent across all templates and static files:

```go
// internal/deploy/consistency_test.go
func TestArtifactConsistency(t *testing.T) {
    // Verify prometheus.yml job name matches alert_rules.yml group name
    // Verify docker-compose.yml Prometheus image matches prometheus.yml expectations
    // Verify Grafana dashboard PromQL queries reference correct job names
    // Verify all port numbers are consistent across artifacts
}
```

This prevents drift between templates and static files as they evolve independently.

#### Test Infrastructure (`internal/testutil/`)

```go
// internal/testutil/mocks.go

// MockPresenter records all output calls for assertion
type MockPresenter struct {
    ProgressCalls []string
    ResultCalls   []string
    ErrorCalls    []error
    WarnCalls     []string
}

func (m *MockPresenter) Progress(msg string) { m.ProgressCalls = append(m.ProgressCalls, msg) }
func (m *MockPresenter) Result(msg string)   { m.ResultCalls = append(m.ResultCalls, msg) }
func (m *MockPresenter) Error(err error)     { m.ErrorCalls = append(m.ErrorCalls, err) }
func (m *MockPresenter) Warn(msg string)     { m.WarnCalls = append(m.WarnCalls, msg) }

// MockDockerRunner returns preconfigured responses
type MockDockerRunner struct {
    Responses map[string]struct {
        Output string
        Err    error
    }
}

func (m *MockDockerRunner) Run(ctx context.Context, args ...string) (string, error) {
    key := strings.Join(args, " ")
    if r, ok := m.Responses[key]; ok {
        return r.Output, r.Err
    }
    return "", fmt.Errorf("unexpected docker call: %s", key)
}
```

### Output Patterns

**All user-facing output goes through Presenter. Never `fmt.Print*` directly.**

```go
// Correct
p.Progress("Generating Dockerfile...")
p.Result("Stack deployed successfully")
p.Warn("Port 8080 is commonly used — ensure no conflicts")
p.Error(err)  // Renders UserError with code + hint, or generic error

// Wrong
fmt.Println("Generating Dockerfile...")  // NEVER
log.Printf("Error: %v", err)            // NEVER
```

**Stream separation:**
- `Progress()` → stderr (ephemeral status, safe to pipe)
- `Result()` → stdout (final output, machine-parseable)
- `Error()` → stderr
- `Warn()` → stderr

**Presenter modes** (`DetectMode()` checks `$TERM`, `$NO_COLOR`, piping):
- `ModeColor` — ANSI colors, spinners (interactive terminal)
- `ModeNoColor` — Structured text, no ANSI (NO_COLOR=1 or dumb terminal)
- `ModePlain` — Bare text (piped to file or another process)

### Docker Interaction Patterns

**All Docker operations go through the `DockerRunner` interface:**

```go
// internal/docker/runner.go
type DockerRunner interface {
    Run(ctx context.Context, args ...string) (string, error)
}

// ExecRunner implements DockerRunner via os/exec
type ExecRunner struct{}

func (e *ExecRunner) Run(ctx context.Context, args ...string) (string, error) {
    cmd := exec.CommandContext(ctx, "docker", args...)
    out, err := cmd.CombinedOutput()
    return string(out), err
}

func NewExecRunner() *ExecRunner {
    return &ExecRunner{}
}
```

**No Docker SDK.** All Docker interaction is `docker` CLI via `os/exec`. This keeps the dependency count at 3 (cobra, yaml.v3, testify) and avoids CGO complications.

**Common Docker calls:**

| Operation | Command |
|-----------|---------|
| Check Docker running | `docker info` |
| Check Compose plugin | `docker compose version` |
| Deploy stack | `docker compose -p {name} up -d` |
| Stop stack | `docker compose -p {name} down` |
| Stack status | `docker compose -p {name} ps --format json` |
| Container logs | `docker compose -p {name} logs --tail 20` |

### File I/O Patterns

```go
// Directory as parameter — never assume cwd
func Run(ctx context.Context, dir string, p output.Presenter, r docker.DockerRunner) error {
    agentfilePath := filepath.Join(dir, "Agentfile")
    // ...
}

// Write files directly — no atomic writes, no temp files
os.WriteFile(filepath.Join(dir, "docker-compose.yml"), rendered, 0644)
os.MkdirAll(filepath.Join(dir, "grafana", "provisioning", "datasources"), 0755)

// Check file existence
if _, err := os.Stat(filepath.Join(dir, "requirements.txt")); err == nil {
    tc.HasRequirements = true
}
```

**No atomic writes.** Generated artifacts are disposable — if a write fails mid-way, the user runs `megacenter deploy` again. Fix-and-re-run philosophy.

**Respect existing files:** `deploy` generates `.dockerignore` only if one doesn't already exist. Uses `os.Stat` to check before writing.

### Template Patterns

```go
// Template field names match Go struct fields exactly
// TemplateContext.Name, TemplateContext.Port, TemplateContext.HealthPath
const dockerComposeTemplate = `# Generated by MegaCenter — do not edit manually
services:
  {{.Name}}:
    build: .
    ports:
      - "{{.Port}}:{{.Port}}"
{{- if .Env}}
    env_file:
      - .env
{{- end}}
`
```

**Rules:**
1. Generated file comment on first line: `# Generated by MegaCenter — do not edit manually`
2. Template field names = Go struct field names (no mapping layer)
3. Use `{{- }}` and `{{ -}}` for whitespace control
4. Templates are `go:embed` strings, not external files
5. `text/template` only — never `html/template` (we generate YAML/Dockerfile, not HTML)

### Cobra Wiring Pattern

```go
// cmd/megacenter/deploy.go — WIRING ONLY, zero business logic
var deployCmd = &cobra.Command{
    Use:   "deploy",
    Short: "Deploy agent with monitoring stack",
    RunE: func(cmd *cobra.Command, args []string) error {
        p := output.NewPresenter(output.DetectMode())
        runner := docker.NewExecRunner()
        return deploy.Run(cmd.Context(), projectDir, p, runner)
    },
}
```

**Rules:**
1. `RunE` only — never `Run` (we need error returns)
2. Body is 3-4 lines: create dependencies, call `Run()`, return error
3. Zero business logic in `cmd/` — all logic lives in `internal/`
4. Root command sets `SilenceErrors: true` and `SilenceUsage: true`
5. `projectDir` resolved once in root `PersistentPreRunE` (default: cwd)

### Command Entry Points

Exact signatures for all 4 command `Run()` functions:

```go
// internal/doctor/doctor.go
func Run(ctx context.Context, p output.Presenter, r docker.DockerRunner) error

// internal/setup/setup.go
func Run(ctx context.Context, dir string, p output.Presenter) error

// internal/deploy/deploy.go
func Run(ctx context.Context, dir string, p output.Presenter, r docker.DockerRunner) error

// internal/status/status.go
func Run(ctx context.Context, dir string, p output.Presenter, r docker.DockerRunner) error
```

**Why `doctor` has no `dir`:** `doctor` checks system prerequisites (Docker, Compose) — it doesn't operate on a project directory. The other 3 commands require a project directory containing an Agentfile.

**Why `setup` has no `DockerRunner`:** `setup` (née `init`) creates an Agentfile interactively — no Docker interaction needed.

### Enforcement Guidelines

**Mandatory rules for implementation — violations are build-breaking:**

1. **No `fmt.Print*` outside `output/` package** — all user-facing text through Presenter
2. **No `os.Exit()` outside `cmd/`** — packages return errors, Cobra handles exit
3. **No `log.*` calls anywhere** — use Presenter for user output, return errors for failures
4. **No Docker SDK imports** — all Docker interaction via `DockerRunner` interface
5. **No external dependencies beyond cobra, yaml.v3, testify** — stdlib for everything else
6. **No `interface{}` / `any` in public APIs** — concrete types with validation
7. **All `Run()` functions accept `context.Context` as first parameter** — Ctrl+C propagation
8. **All generated files start with `# Generated by MegaCenter`** — user knows what's safe to delete

---

## Project Structure & Boundaries

_Autonomous Party Mode — 3 rounds to convergence. 11 adjustments consolidated. Key decisions: templates/static inside internal/deploy/ (sole consumer), install.sh added (FR39), runner_test.go eliminated (trivial glue), CI = go build + unit tests only, integration tests local-only for v0.1._

### Requirements → Package Mapping

| FR Category | Package | FRs | Key Files |
|-------------|---------|-----|-----------|
| Environment Checks | `internal/doctor/` | FR1-FR3 | `doctor.go`, `checks.go` |
| Project Scanning & Detection | `internal/setup/` | FR4-FR15 | `setup.go`, `scanner.go`, `detector.go` |
| Artifact Generation | `internal/deploy/` | FR16-FR23, FR31-FR37 | `dockerfile.go`, `compose.go`, `prometheus.go`, `grafana.go` |
| Deployment Execution | `internal/deploy/` | FR24-FR26 | `orchestrate.go`, `healthcheck.go` |
| Status Query | `internal/status/` | FR27-FR30 | `status.go` |
| Agentfile Lifecycle | `internal/agentfile/` | Cross-cutting | `agentfile.go`, `framework.go`, `dockerfile_mode.go` |
| User Output | `internal/output/` | NFR15, NFR22-23 | `presenter.go`, `mode.go`, `errors.go`, `format.go` |
| Docker Abstraction | `internal/docker/` | Cross-cutting | `runner.go` |
| Test Infrastructure | `internal/testutil/` | Cross-cutting | `golden.go`, `mocks.go` |
| CLI Wiring | `cmd/megacenter/` | — | `main.go`, `doctor.go`, `deploy.go`, `init.go`, `status.go` |
| Build & Release | Root / `.github/` | FR38-FR41 | `Makefile`, `install.sh`, `ci.yml`, `release.yml` |

### Complete Project Directory Structure

_This is the **definitive** project tree — supersedes the preliminary tree in Starter Template Evaluation (Step 3)._

```
megacenter/
├── cmd/
│   └── megacenter/
│       ├── main.go                          # Root Cobra command, PersistentPreRunE, SilenceErrors/SilenceUsage
│       ├── doctor.go                        # doctor subcommand wiring (3 lines in RunE)
│       ├── deploy.go                        # deploy subcommand wiring
│       ├── init.go                          # init subcommand wiring → calls setup.Run()
│       │                                    # (init.go has no special Go compiler meaning; only func init() does)
│       └── status.go                        # status subcommand wiring
│
├── internal/
│   ├── agentfile/
│   │   ├── agentfile.go                     # Agentfile struct, Load(), Parse(), Validate()
│   │   ├── agentfile_test.go                # Parse/validate unit tests
│   │   ├── framework.go                     # Framework custom type + UnmarshalYAML
│   │   ├── dockerfile_mode.go               # DockerfileMode type + UnmarshalYAML
│   │   └── testdata/
│   │       ├── valid_minimal.yaml           # Only required fields (version, name)
│   │       ├── valid_full.yaml              # All 7 fields populated
│   │       ├── valid_custom_dockerfile.yaml # dockerfile: custom
│   │       ├── invalid_version.yaml         # version: "99.0"
│   │       ├── invalid_framework.yaml       # framework: "django" (not supported)
│   │       ├── invalid_port.yaml            # port: 99999
│   │       ├── invalid_name_dns.yaml        # name: "My Agent!" (not DNS label)
│   │       ├── missing_required.yaml        # No version field
│   │       └── bad_yaml.yaml                # Malformed YAML
│   │
│   ├── deploy/
│   │   ├── deploy.go                        # Run() entry point — orchestration sequence
│   │   ├── dockerfile.go                    # Dockerfile template rendering
│   │   ├── compose.go                       # docker-compose.yml template rendering
│   │   ├── prometheus.go                    # prometheus.yml template rendering
│   │   ├── grafana.go                       # Static file copy (dashboards + provisioning)
│   │   ├── orchestrate.go                   # docker compose up/down, project naming
│   │   ├── healthcheck.go                   # Post-deploy HTTP health verification
│   │   ├── embed.go                         # //go:embed directives for templates/ and static/
│   │   ├── context.go                       # TemplateContext struct + detection helpers
│   │   ├── deploy_test.go                   # Run() orchestration unit tests (mocked)
│   │   ├── dockerfile_test.go               # Golden file tests for Dockerfile rendering
│   │   ├── compose_test.go                  # Golden file tests for docker-compose.yml
│   │   ├── prometheus_test.go               # Golden file tests for prometheus.yml
│   │   ├── grafana_test.go                  # Static file copy verification
│   │   ├── healthcheck_test.go              # Health check with mock HTTP
│   │   ├── consistency_test.go              # Cross-artifact consistency (job names, ports, versions)
│   │   ├── deploy_integration_test.go       # //go:build integration — real Docker
│   │   ├── templates/                       # go:embed — processed with text/template
│   │   │   ├── Dockerfile.tmpl              # 2 branches: requirements.txt vs pyproject.toml
│   │   │   ├── docker-compose.yml.tmpl      # 3 services: agent + prometheus + grafana
│   │   │   └── prometheus.yml.tmpl          # scrape configs: agent-health + agent-metrics
│   │   ├── static/                          # go:embed — copied without processing
│   │   │   ├── alert_rules.yml              # AgentDown alert (Prometheus {{ }} syntax, not Go template)
│   │   │   └── grafana/
│   │   │       ├── dashboards/
│   │   │       │   ├── overview.json        # 4 panels: Status, Uptime, Latency, Last Seen
│   │   │       │   └── detail.json          # 4 panels: Latency timeline, Health timeline, 24h/7d uptime
│   │   │       └── provisioning/
│   │   │           ├── dashboards/
│   │   │           │   └── dashboards.yml   # Dashboard file provider config
│   │   │           └── datasources/
│   │   │               └── datasource.yml   # Static: http://prometheus:9090
│   │   └── testdata/
│   │       ├── fixtures/
│   │       │   ├── minimal/                 # Only required Agentfile fields
│   │       │   │   ├── Agentfile
│   │       │   │   ├── requirements.txt
│   │       │   │   ├── main.py
│   │       │   │   └── expected.yaml
│   │       │   ├── full/                    # All fields + .env
│   │       │   │   ├── Agentfile
│   │       │   │   ├── requirements.txt
│   │       │   │   ├── main.py
│   │       │   │   ├── .env
│   │       │   │   └── expected.yaml
│   │       │   └── pyproject/               # pyproject.toml dependency path
│   │       │       ├── Agentfile
│   │       │       ├── pyproject.toml
│   │       │       ├── main.py
│   │       │       └── expected.yaml
│   │       └── golden/
│   │           ├── dockerfile_minimal.golden
│   │           ├── dockerfile_full.golden
│   │           ├── dockerfile_pyproject.golden
│   │           ├── compose_minimal.golden
│   │           ├── compose_full.golden
│   │           └── prometheus.golden
│   │
│   ├── doctor/
│   │   ├── doctor.go                        # Run() — sequential check pipeline
│   │   ├── checks.go                        # checkDocker(), checkCompose(), checkPorts()
│   │   ├── doctor_test.go                   # Unit tests with MockDockerRunner
│   │   └── doctor_integration_test.go       # //go:build integration
│   │
│   ├── docker/
│   │   └── runner.go                        # DockerRunner interface + ExecRunner impl (~15 lines)
│   │                                        # No unit test — trivial glue code. Tested indirectly via integration tests.
│   │
│   ├── output/
│   │   ├── presenter.go                     # Presenter interface + colorPresenter/plainPresenter
│   │   ├── mode.go                          # DetectMode(), ModeColor/ModeNoColor/ModePlain
│   │   ├── errors.go                        # UserError, UserWarning, error catalog (E101-E502)
│   │   ├── format.go                        # Format helper functions (return strings)
│   │   ├── presenter_test.go                # Output capture tests
│   │   ├── mode_test.go                     # TERM/NO_COLOR detection tests
│   │   └── errors_test.go                   # Error formatting + catalog tests
│   │
│   ├── setup/
│   │   ├── setup.go                         # Run() entry point — interactive Agentfile creation
│   │   │                                    # Also handles FR14 (.gitignore) and FR15 (.env.example)
│   │   ├── scanner.go                       # Scan project dir for Python patterns
│   │   ├── detector.go                      # Detect framework, port, health_path, entry point
│   │   ├── setup_test.go                    # Run() with mock filesystem
│   │   ├── scanner_test.go                  # Scanner with fixture projects
│   │   ├── detector_test.go                 # Detection logic unit tests
│   │   └── testdata/
│   │       └── fixtures/
│   │           ├── fastapi_project/         # FastAPI detection: uvicorn, port 8000
│   │           │   ├── main.py
│   │           │   └── requirements.txt
│   │           ├── flask_project/           # Flask detection: flask run, port 5000
│   │           │   ├── app.py
│   │           │   └── requirements.txt
│   │           ├── langgraph_project/       # LangGraph detection: langgraph-specific patterns
│   │           │   ├── agent.py
│   │           │   └── requirements.txt
│   │           ├── pyproject_project/       # pyproject.toml instead of requirements.txt
│   │           │   ├── main.py
│   │           │   └── pyproject.toml
│   │           ├── generic_project/         # No framework detected — defaults applied
│   │           │   ├── main.py
│   │           │   └── requirements.txt
│   │           └── empty_project/           # Nothing to detect — error case
│   │
│   ├── status/
│   │   ├── status.go                        # Run() — read-only container + health query
│   │   ├── status_test.go                   # Unit tests with MockDockerRunner
│   │   └── status_integration_test.go       # //go:build integration
│   │
│   └── testutil/
│       ├── golden.go                        # AssertGolden() — UPDATE_GOLDEN=1 support
│       └── mocks.go                         # MockPresenter, MockDockerRunner
│
├── .github/
│   └── workflows/
│       ├── ci.yml                           # Push/PR: go build, go test, golangci-lint, shellcheck
│       └── release.yml                      # Tag v*: cross-compile 2 platforms, SHA256, GitHub Release
│
├── install.sh                               # curl | sh installer — FR39 (detects OS/arch, downloads binary)
├── .golangci.yml                            # Linter rules (gofmt, govet, errcheck, staticcheck)
├── .gitignore                               # /megacenter, /dist/, .megacenter/
├── go.mod                                   # module github.com/antonioromero/megacenter
├── go.sum
├── Makefile                                 # build, test, test-integration, lint, build-all, checksums
├── LICENSE                                  # MIT
└── README.md
```

**Totals: 9 packages, ~35 source files, ~18 test files, ~25 testdata files, 2 CI workflows.**

### Embed Strategy

```go
// internal/deploy/embed.go
package deploy

import "embed"

//go:embed templates/*.tmpl
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS
```

Paths relative to the package directory. `deploy` is the sole consumer of templates and static files — no `embed.FS` passed between packages. Full encapsulation.

### Generated Output Structure (User's Agent Project)

What MegaCenter generates in the user's project directory:

```
my-agent/                                    # User's project root
├── Agentfile                                # Created by `megacenter init`, persists across deploys
├── .env                                     # Optional — user creates/manages, never generated
├── .gitignore                               # Updated by `megacenter init` — adds .megacenter/ entry (FR14)
├── .dockerignore                            # Generated by `megacenter deploy` (if not exists)
│                                            # MUST exclude .megacenter/ from Docker build context
├── .megacenter/                             # Generated by `megacenter deploy` — FULLY REGENERATED each time
│   ├── Dockerfile                           # From Dockerfile.tmpl
│   ├── docker-compose.yml                   # From docker-compose.yml.tmpl
│   ├── prometheus.yml                       # From prometheus.yml.tmpl
│   ├── alert_rules.yml                      # Copied from static/
│   └── grafana/
│       ├── dashboards/
│       │   ├── overview.json                # Copied from static/
│       │   └── detail.json                  # Copied from static/
│       └── provisioning/
│           ├── dashboards/
│           │   └── dashboards.yml           # Copied from static/
│           └── datasources/
│               └── datasource.yml           # Copied from static/
├── main.py                                  # User's agent code (untouched by MegaCenter)
├── requirements.txt                         # User's dependencies (untouched)
└── ...
```

**Docker Compose invocation:** `docker compose -f .megacenter/docker-compose.yml -p {name} up -d`

**Build context resolution:** `docker-compose.yml` uses `build: { context: .., dockerfile: .megacenter/Dockerfile }`. The `..` resolves to the project root (relative to the compose file in `.megacenter/`), so `COPY requirements.txt .` in the Dockerfile accesses user source files correctly.

**`.dockerignore` contents** (generated by deploy if not exists):
```
.megacenter/
.git/
.env
__pycache__/
*.pyc
```

### Package Dependency Graph

```
cmd/megacenter/
    ├── imports → internal/doctor/
    ├── imports → internal/setup/
    ├── imports → internal/deploy/
    ├── imports → internal/status/
    ├── imports → internal/output/     (NewPresenter, DetectMode)
    └── imports → internal/docker/     (NewExecRunner)

internal/doctor/
    ├── imports → internal/output/     (Presenter)
    └── imports → internal/docker/     (DockerRunner)

internal/setup/
    ├── imports → internal/output/     (Presenter)
    └── imports → internal/agentfile/  (Agentfile struct — for writing)

internal/deploy/
    ├── imports → internal/output/     (Presenter)
    ├── imports → internal/docker/     (DockerRunner)
    └── imports → internal/agentfile/  (Load + Agentfile struct — for reading)

internal/status/
    ├── imports → internal/output/     (Presenter)
    ├── imports → internal/docker/     (DockerRunner)
    └── imports → internal/agentfile/  (Load — needs project name for compose -p)

internal/agentfile/  → imports NOTHING internal (leaf package)
internal/output/     → imports NOTHING internal (leaf package)
internal/docker/     → imports NOTHING internal (leaf package)
internal/testutil/   → imports internal/output/ + internal/docker/ (interfaces for mocks)
```

**3 dependency rules:**
1. **No circular dependencies** — the graph is a strict DAG
2. **Leaf packages** (`agentfile`, `output`, `docker`) import nothing internal — maximum reusability
3. **`cmd/` is the sole assembly point** — creates instances and passes them to `Run()`

### Data Flow: `megacenter deploy`

```
cmd/deploy.go          internal/agentfile/        internal/deploy/            Docker
     │                       │                          │                       │
     ├── NewPresenter()      │                          │                       │
     ├── NewExecRunner()     │                          │                       │
     └── deploy.Run() ──────►│                          │                       │
                              ├── Load(dir) ───────────►│                       │
                              │  Parse + Validate       │                       │
                              │◄── Agentfile struct ────┤                       │
                              │                         ├── buildContext()      │
                              │                         ├── renderDockerfile()  │
                              │                         ├── renderCompose()     │
                              │                         ├── renderPrometheus()  │
                              │                         ├── copyStaticFiles()   │
                              │                         ├── writeAll(.megacenter/)│
                              │                         ├── r.Run("compose",...) ──► docker compose up
                              │                         │◄─── output ──────────┤
                              │                         ├── healthCheck()        ──► HTTP GET /health
                              │                         │◄─── 200 OK ──────────┤
                              │                         └── p.Result("Done")   │
```

### CI/CD Pipeline Structure

**`ci.yml`** (push + PR):
1. `go build ./...` — catches compilation errors in untested files
2. `go test ./internal/... -v -count=1` — unit tests only (no Docker required)
3. `golangci-lint run ./...` — linting
4. `shellcheck install.sh` — bash script validation

**Integration tests are local-only for v0.1.** CI runs unit tests. `make test-integration` requires Docker and is executed manually by the developer before merge.

**`release.yml`** (on tag `v*`):
1. Cross-compile for `darwin/arm64` and `linux/amd64`
   _Note: Ubuntu AMD64 and Debian AMD64 share the same `linux/amd64` binary — 2 platform combinations produce the 3 PRD targets._
2. Generate SHA256 checksums
3. Create GitHub Release with binaries + checksums + install.sh

### Makefile

```makefile
BINARY    := megacenter
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS   := -s -w -X main.version=$(VERSION)
# Ubuntu AMD64 and Debian AMD64 share linux/amd64 binary (2 combos = 3 PRD targets)
PLATFORMS := darwin/arm64 linux/amd64

.PHONY: build test test-integration lint build-all checksums clean

build:               ## Build for current platform
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/megacenter/

test:                ## Run unit tests (no Docker required)
	go test ./internal/... -v -count=1

test-integration:    ## Run integration tests (Docker required, local only)
	go test ./internal/... -v -count=1 -tags=integration

lint:                ## Run linters
	golangci-lint run ./...

build-all:           ## Cross-compile for all platforms
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
		go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-$$os-$$arch ./cmd/megacenter/; \
	done

checksums:           ## Generate SHA256 checksums
	cd dist && shasum -a 256 $(BINARY)-* > checksums.txt

clean:               ## Remove build artifacts
	rm -rf $(BINARY) dist/
```

### Architectural Boundary Summary

| Boundary | Enforced By | Violation Detection |
|----------|-------------|-------------------|
| `internal/` visibility | Go compiler | Build fails if external package imports `internal/` |
| No `fmt.Print*` outside `output/` | Code review + linter | `golangci-lint` custom rule or grep in CI |
| No `os.Exit()` outside `cmd/` | Code review | Enforcement guideline #2 |
| No Docker SDK | `go.mod` inspection | Only cobra, yaml.v3, testify in dependencies |
| DAG dependency graph | Go compiler | Circular imports = build failure |
| Template/static encapsulation | `go:embed` in `deploy/` only | No other package references template/static paths |
| Generated files marked | Template content | First line: `# Generated by MegaCenter` |

### Corrections from Previous Steps

| Step | Original | Corrected in Step 6 | Reason |
|------|----------|-------------------|--------|
| Step 3 | `templates/grafana/datasource.yml.tmpl` | `static/grafana/provisioning/datasources/datasource.yml` | Step 4 determined URL is static (`http://prometheus:9090`) |
| Step 3 | `templates/` and `static/` at project root | Inside `internal/deploy/` | deploy is sole consumer; enables package-local `go:embed` |
| Step 2 | `pkg/` package prefix | `internal/` | Step 3 decision — enforced by Go compiler |
| Step 2 | `pkg/init/` | `internal/setup/` | Avoids `func init()` semantic collision |

---

## Architecture Validation Results

_Autonomous Party Mode — 3 rounds to convergence. 12 adjustments. 1 critical bug fixed in-place (alert_rules.yml mount). 3 inter-step inconsistencies corrected via errata. Full FR/NFR coverage verified._

### Coherence Validation ✅

**Decision Compatibility:**
- Go 1.22+ / Cobra / yaml.v3 / testify — no version conflicts
- Prometheus v3.10.0 queries from Grafana OSS 12.4.0 — compatible (standard PromQL)
- Docker Compose V2 invoked via os/exec DockerRunner — compatible
- `go:embed` + `text/template` + static file copy — no conflicts
- `internal/` layout with 3 external deps — minimal, coherent

**Pattern Consistency:**
- All user output through Presenter — enforced by enforcement guideline #1
- All Docker interaction through DockerRunner — enforced by guideline #4
- All errors returned (not thrown/logged) — Go idiomatic, enforced by guidelines #2-3
- Naming conventions consistent: DNS labels for agent names, snake_case files, PascalCase exports, single-letter receivers

**Structure Alignment:**
- 9 packages match 1:1 with the dependency DAG — Go compiler enforces no cycles
- templates/ and static/ inside `internal/deploy/` — sole consumer, package-local `go:embed`
- testdata/ per package — Go convention, `go build` ignores them
- Generated output in `.megacenter/` — isolated from user code, fully regenerated per deploy

### Requirements Coverage ✅

#### Functional Requirements: 41 FRs

| Category | FRs | Status | Architectural Support |
|----------|-----|--------|----------------------|
| Environment Checks | FR1-FR3 | ✅ 3/3 | `internal/doctor/` — checks.go, exit codes |
| Project Detection | FR4-FR11 | ✅ 8/8 | `internal/setup/` — scanner.go, detector.go |
| Project Config Files | FR12-FR15 | ✅ 4/4 | `internal/setup/` (FR12 .env.example, FR13 .gitignore) + `internal/agentfile/` (FR14 schema version, FR15 exists check) |
| Stack Generation | FR16-FR21 | ✅ 6/6 | `internal/deploy/` — dockerfile.go, compose.go, prometheus.go, grafana.go |
| Deployment Execution | FR22-FR26 | ✅ 5/5 | `internal/deploy/` — orchestrate.go, healthcheck.go |
| Dry-Run | FR17 [SHOULD] | ⏭️ Deferred | Deferred to v0.2. Architecturally trivial — deploy.Run() split into generate-only + execute. No structural changes needed |
| Health & Status | FR27-FR30 | ✅ 4/4 [SHOULD] | `internal/status/` — status.go |
| Probe Metrics | FR31-FR34 | ✅ 4/4 | prometheus.yml template + alert_rules.yml (mount bug fixed) + overview.json |
| Bonus Metrics | FR35 | ✅ 1/1 | agent-metrics job always generated in prometheus.yml |
| Dashboard UX | FR36-FR37 | ✅ 2/2 | Grafana anonymous env vars + honest panel labeling |
| Installation | FR38-FR41 | ✅ 4/4 | install.sh + Makefile + release.yml + Cobra --version |

**Coverage: 36/36 MUST ✅ · 4/5 SHOULD ✅ · 1/5 SHOULD deferred (FR17)**

#### Non-Functional Requirements: 23 NFRs

| Category | NFRs | Status | Architectural Support |
|----------|------|--------|----------------------|
| Performance | NFR1-NFR6 | ✅ 6/6 | Go binary (NFR1), Docker layer caching (NFR2-3), Grafana standard (NFR4), prometheus.yml 15s interval (NFR5), 4-command flow (NFR6) |
| Reliability | NFR7-NFR11 | ✅ 5/5 | Defensive error handling (NFR7), templates cross-platform (NFR8), named volumes (NFR9), golden file testing (NFR10), 15s scrape + 30s alert = 45s max (NFR11) |
| Compatibility | NFR12-NFR15 | ✅ 4/4 | CGO_ENABLED=0 (NFR12), Compose Specification (NFR13), python:slim multi-arch (NFR14), Presenter modes (NFR15) |
| Security | NFR16-NFR20 | ✅ 5/5 | env_file reference (NFR16), .gitignore via setup (NFR17), SHA256 in release (NFR18), anonymous access localhost-only (NFR19), no outbound calls (NFR20) |
| Resource | NFR21 | ✅ 1/1 | Prometheus retention flags (7d, 500MB) |
| DX | NFR22-NFR23 | ✅ 2/2 | UserError{What, Fix} (NFR22), UserWarning{What, Assumed, Override} (NFR23) |

**Coverage: 22/22 MUST ✅ · 1/1 SHOULD ✅ (NFR18)**

### Implementation Readiness ✅

| Dimension | Assessment | Evidence |
|-----------|-----------|----------|
| **Decision completeness** | High | All technology choices versioned. All patterns with code examples. All templates with full content |
| **Pattern completeness** | High | 8 pattern categories in Step 5. 8 enforcement guidelines. Receiver letters, Run() signatures, Makefile targets all specified |
| **Structure completeness** | High | Every file named and described. 9 packages, ~35 source files, ~18 test files, ~25 testdata files |
| **Test strategy** | Complete | 5 testing approaches: unit (mocked), golden file (template output), fixture projects (end-to-end), integration (real Docker), consistency (cross-artifact) |
| **Dependency graph** | Verified | DAG with 3 leaf packages, no cycles. Go compiler enforces |
| **Error catalog** | Complete | 15 errors (E101-E502) with What + Fix for each |

### Bug Fixes Applied In-Place

| Location | Bug | Fix | Impact |
|----------|-----|-----|--------|
| Category 3: docker-compose.yml template | `alert_rules.yml` not mounted into Prometheus container | Added `- ./alert_rules.yml:/etc/prometheus/alert_rules.yml:ro` to Prometheus volumes | FR20, FR34, NFR11 now functional |

### Corrections & Errata

_When a later step contradicts an earlier step, this table is the authoritative resolution. AI agents implementing from this document should consult this table first._

| Location | Original | Corrected | Reason |
|----------|----------|-----------|--------|
| Step 5: Exit Codes table | Codes {0, 1, 2, 3, 4} | **Codes {0, 1, 130}** only | Step 4 Category 2 is authoritative: "no distinction between user/internal errors in exit code" |
| Step 5: UserError example fields | `Code, Message, Hint` | **`Code, What, Fix`** | Step 4 Category 2 defines the struct |
| Step 5: Presenter.Error signature | Not explicitly reconciled | **`Error(err error)`** with `errors.As(*UserError)` type assertion | Must handle both UserError (formatted) and unexpected errors ([INTERNAL] prefix) |
| Step 6: setup.go inline comment | "FR14 (.gitignore), FR15 (.env.example)" | **"FR12 (.env.example), FR13 (.gitignore)"** | PRD numbering |
| Step 5: Template Rule #4 | "Templates are `go:embed` strings" | **"Templates are `.tmpl` files embedded via `go:embed`"** | Clarification — templates are files, not string constants |
| Step 3: datasource.yml.tmpl | Listed as template in templates/ | **Static file in static/** | Step 4 determined URL is always `http://prometheus:9090` |
| Step 6: deploy/ project tree | Missing `constants.go` | **Add `constants.go`** between context.go and deploy_test.go | Defined in Step 4 Category 4 (shared constants) |

### Additional Project Files (Party Mode additions)

Files added during validation that were missing from the Step 6 tree:

```
megacenter/
├── CONTRIBUTING.md                          # PR rules, conventional commits, tests required
├── .github/
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.md                    # Bug report template
│   │   └── feature_request.md               # Feature request template
│   └── workflows/
│       ├── ci.yml
│       └── release.yml
├── internal/
│   └── deploy/
│       ├── constants.go                     # JobHealth, JobMetrics, DatasourceName, NetworkName
│       └── ...
```

### Architecture Completeness Checklist

**✅ Requirements Analysis (Step 2)**
- [x] Project context thoroughly analyzed — 41 FRs, 23 NFRs mapped
- [x] Scale and complexity assessed — Medium (4 commands, 7+2 packages)
- [x] 11 technical constraints identified and numbered
- [x] 4 external integration points mapped with failure modes
- [x] Testing surface areas cataloged

**✅ Technology Selection (Step 3)**
- [x] CLI framework selected with rationale (Cobra, no Viper)
- [x] Template strategy defined (go:embed + text/template + static copy)
- [x] 3 external dependencies justified (cobra, yaml.v3, testify)
- [x] Build tooling selected (Makefile + GitHub Actions)

**✅ Architectural Decisions (Step 4)**
- [x] Agentfile schema: 7 fields, Parse-Don't-Validate, DNS label names
- [x] Error handling: UserError{What, Fix}, Presenter, {0, 1, 130} exit codes
- [x] Docker generation: 3 templates + 5 static files, TemplateContext struct
- [x] Monitoring: Prometheus v3.10.0 + Grafana OSS 12.4.0, probe-based metrics, 2 dashboards

**✅ Implementation Patterns (Step 5)**
- [x] Go naming conventions with receiver letters
- [x] Testing strategy: golden files, fixtures, mocks, integration split
- [x] Cobra wiring pattern: RunE = 3-4 lines, zero business logic in cmd/
- [x] Exact Run() signatures for all 4 commands
- [x] 8 enforcement guidelines for AI agents

**✅ Project Structure (Step 6)**
- [x] Complete directory tree with all files described
- [x] Package dependency DAG verified
- [x] Generated output structure (.megacenter/) documented
- [x] Embed strategy defined (package-local go:embed)
- [x] CI/CD pipeline structure specified
- [x] Makefile with all targets

**✅ Validation (Step 7)**
- [x] 1 critical bug found and fixed in-place (alert_rules.yml mount)
- [x] 6 inter-step inconsistencies identified and corrected via errata
- [x] 100% MUST FR coverage verified
- [x] 100% MUST NFR coverage verified
- [x] Implementation readiness confirmed: High

### Architecture Readiness Assessment

**Overall Status: ✅ READY FOR IMPLEMENTATION**

**Confidence Level: HIGH** — All critical decisions documented, all requirements covered, all inconsistencies resolved, one real bug caught and fixed.

**Key Strengths:**
1. Extreme simplicity — 3 external deps, 9 packages, zero state, fix-and-re-run philosophy
2. High testability — interfaces at every boundary, 5 testing strategies, golden files
3. Clear enforcement — 8 mandatory rules that AI agents can follow mechanically
4. Complete error catalog — every user-facing error has What + Fix
5. Probe-based monitoring — zero agent instrumentation needed for core value

**Areas for v0.2 Enhancement:**
1. FR17 dry-run mode — architecturally trivial split of deploy.Run()
2. Multi-agent unified dashboards — Grafana template variables
3. Alertmanager for notification delivery (Slack, email)
4. GoReleaser for expanded distribution channels
5. `--json` output flag — structured stdout for automation

### Implementation Handoff

**AI Agent Guidelines:**
1. Read this document top-to-bottom before starting implementation
2. **Consult the Corrections & Errata table** before implementing any pattern from Steps 3-6
3. Follow enforcement guidelines (Step 5) — violations are build-breaking
4. Use the exact Run() signatures, Presenter interface, and DockerRunner interface as specified
5. Test every template against golden files before considering it done
6. Refer to the error catalog for exact error messages and codes

**Recommended Implementation Order:**
1. Project scaffolding: `go mod init`, directory structure, Makefile
2. `internal/output/` — Presenter, modes, error types (leaf package, no deps)
3. `internal/agentfile/` — Parse, Validate, Load (leaf package, no deps)
4. `internal/docker/` — DockerRunner interface + ExecRunner (leaf package)
5. `internal/testutil/` — MockPresenter, MockDockerRunner, AssertGolden
6. `internal/doctor/` — simplest command, validates tooling
7. `internal/setup/` — second command, creates Agentfile
8. `internal/deploy/` — largest package, highest risk, leave for last
9. `internal/status/` — read-only, depends on deploy working
10. `cmd/megacenter/` — Cobra wiring, assembled last
