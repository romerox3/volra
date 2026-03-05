---
stepsCompleted: [step-01-init, step-02-context, step-03-starter, step-04-decisions, step-05-patterns, step-06-structure, step-07-validation, step-08-complete]
lastStep: 8
status: 'complete'
completedAt: '2026-03-02'
inputDocuments:
  - '_bmad-output/planning-artifacts/prd.md'
  - '_bmad-output/planning-artifacts/product-brief-Volra-2026-03-02.md'
  - '_bmad-output/brainstorming/brainstorming-session-2026-03-02.md'
  - 'guide.md'
workflowType: 'architecture'
executionMode: 'GENERATE'
project_name: 'Volra'
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
| **State model** | Zero CLI state — project directory is the persistent context. The Agentfile persists between init and deploy. `.volra/` is regenerated each deploy. Prometheus volumes persist across rebuilds (NFR9) |
| **Component count** | 11 Go packages (5 command + 4 shared + 1 wiring + 1 test) + 1 Python package (volra-observe) |
| **External dependencies** | Docker Engine, Docker Compose V2, Go standard library |
| **Generated artifact types** | 7+ (Dockerfile, docker-compose.yml, prometheus.yml, Grafana dashboard JSON ×3 (overview, detail, level2), per-service env files, alert_rules.yml) |
| **Highest-risk component** | deploy/ — largest package, Dockerfile generation flagged as highest complexity/fragility in PRD |

### Go Package Structure (Preliminary)

_Converged through 5 rounds of Party Mode. Principle: 1 package per CLI command, shared packages for cross-cutting models and output. Go idioms over abstractions — no custom frameworks._

| Package | Type | FRs | Responsibility |
|---------|------|-----|---------------|
| `cmd/volra/` | Wiring | — | main.go + Cobra subcommands (doctor, init, deploy, status, logs, quickstart, mcp). `--json` flag, `newPresenter()`/`flushPresenter()` helpers. No business logic |
| `internal/doctor/` | Command | FR1-FR3 | Environment checks, fix suggestions, exit codes. Level 2 metrics endpoint check |
| `internal/setup/` | Command | FR4-FR15 | Scan project → build Agentfile model → write Agentfile + .env.example + .gitignore. Pipeline: scan → structure → persist |
| `internal/deploy/` | Command | FR16-FR26, FR31-FR37 | Read Agentfile → generate artifacts → execute docker compose → verify health. Files: `dockerfile.go`, `compose.go`, `prometheus.go`, `grafana.go`, `orchestrate.go`, `healthcheck.go`, `service_defaults.go`, `envfiles.go`, `preflight.go` |
| `internal/status/` | Command | FR27-FR30 | Query Docker for container states, health check, daemon detection. Read-only |
| `internal/agentfile/` | Shared | Cross-cutting | Parse, validate, apply defaults, schema version check. Structs: Agentfile, Service, SecurityContext, ObservabilityConfig, BuildConfig, HealthcheckConfig, ResourceConfig, TmpfsMount |
| `internal/output/` | Shared | Cross-cutting | Unified Presenter (color, nocolor, plain, JSON). `--json` via `JSONPresenter` with deferred `Flush()`. Structured errors (UserError) and warnings (UserWarning) |
| `internal/mcp/` | Command | — | MCP server: JSON-RPC 2.0 protocol over stdio. 4 tools: `volra_deploy`, `volra_status`, `volra_logs`, `volra_doctor`. Direct protocol implementation (no SDK) |
| `internal/templates/` | Shared | — | Embedded quickstart templates (basic, rag, conversational) via `go:embed`. Template scaffolding with name substitution |
| `internal/docker/` | Shared | Cross-cutting | DockerRunner interface for shelling out to `docker` / `docker compose` |
| `internal/testutil/` | Test | — | Test helpers: mocks, golden file utilities |
| `volra-observe/` | External | — | Python package: framework-agnostic LLM observability. Auto-patches OpenAI/Anthropic SDKs. 5 Prometheus metrics |

**deploy/ complexity note:** This is the largest package with 6 internal files covering 4 types of artifact generation + orchestration + health verification. Internal file discipline documented here; detailed design in subsequent architecture steps.

### Technical Constraints

| # | Constraint | Source | Architecture Impact |
|---|-----------|--------|-------------------|
| **0** | **1 developer, 8 weeks** | PRD: Resource Reality | Every architectural decision passes the filter: "can one person implement, test, and maintain this?" Go idioms over abstractions. No custom frameworks. Patterns implementable with stdlib |
| 1 | Go single binary | PRD: Installation | Cross-compile for 3 targets (macOS ARM64, Ubuntu AMD64, Debian AMD64), embed templates |
| 2 | 3 containers per project | PRD: Scoping | Docker Compose as sole orchestrator |
| 3 | Probe-based metrics only | PRD: FR31, Scoping | No sidecar, no gateway, Prometheus scrapes health_path directly |
| 4 | Idempotent regeneration | PRD: Design Constraints | `.volra/` always fully regenerated, no merge. Generated artifacts are ephemeral |
| 5 | Python-only detection | PRD: Scoping | Detector hardcoded to Python patterns (v0.1) |
| 6 | Framework: generic + LangGraph | PRD: FR5 | Detection strategy extensible but only 2 implementations shipped |
| 7 | No plugins, no hooks | PRD: Design Constraints | Override = edit generated files. No user-facing extension points |
| 8 | v0.2 migration path | PRD: Scoping | v0.1 independent stacks → v0.2 shared infrastructure. Architecture must not block this transition |
| 9 | Topology variation point | Party Mode R2 | Templates must support standalone (v0.1) vs shared (v0.2) topology. v0.1 implements standalone only, but the template structure must accommodate the switch without rewrite |
| 10 | Artifact vs data lifecycle | Party Mode R4 | Generated artifacts (`.volra/`) are ephemeral — regenerated every deploy. Runtime data (Prometheus volumes) is persistent — survives rebuilds. Two distinct lifecycle domains that must not be conflated |
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

**Explicitly NOT included: Viper.** Volra has no config file system. The Agentfile is parsed directly with yaml.v3 in `internal/agentfile/`. Viper would add ~15 transitive dependencies for zero benefit. Any tutorial suggesting Viper integration should be ignored.

#### Template & Static Asset Strategy

Two categories of embeddable files, with distinct processing strategies:

| Category | Directory | Processing | Contents |
|----------|-----------|-----------|----------|
| **Templates** (processed) | `templates/` | `text/template` stdlib with `template.ParseFS` | Dockerfile.tmpl, docker-compose.yml.tmpl, prometheus.yml.tmpl, datasource.yml.tmpl |
| **Static assets** (copied) | `static/` | `go:embed` → copy to output, no processing | Grafana dashboard JSONs, Grafana provisioning (dashboard provider) |

**Why dashboards are static, not templates:**
- Grafana dashboard JSON is deeply nested (200-400 lines). Go templates (`{{ }}`) inside JSON (`{}`) creates an escaping nightmare
- Grafana provides native template variables (`$agent_name`) — panel queries reference these variables, defined once in the dashboard JSON
- The only dynamic value Volra needs to inject is the Prometheus datasource URL, which is handled via the datasource provisioning YAML (a template), not the dashboard JSON itself
- Static dashboards only need `json.Valid()` tests, not golden file testing — simpler to maintain

**For files needing minimal injection** (not full template logic): `strings.NewReplacer()` with `${PLACEHOLDER}` convention. One call, explicit, debuggable.

#### Project Structure

```
volra/
├── cmd/
│   └── volra/
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
- **`internal/` not `pkg/`**: Volra is a CLI tool, not a library. `internal/` is enforced by the Go compiler — no external package can import these. Aligned with constraint #7 (no plugins, no hooks)
- **`internal/setup/` not `internal/init/`**: Avoids semantic collision with Go's special `func init()`. The CLI command is `volra init`, the package is `setup` — different names, same responsibility
- **`testdata/` per package**: Go convention. Each package owns its test fixtures. `go build` ignores `testdata/` directories
- **`deploy/` internal file structure**: 6 files for the largest package. Not sub-packages — Go idiom is flat packages with clear file boundaries

#### Docker Interaction Strategy

**No Docker SDK.** All Docker interaction via `os/exec.Command("docker", ...)` shell-out.

**Rationale:** The Docker Go SDK (`github.com/docker/docker/client`) would add 20+ transitive dependencies for 3 operations: check if running, compose up, query container status. `os/exec` is stdlib, zero deps, and sufficient for Volra's needs.

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
GOOS=darwin GOARCH=arm64 go build -o dist/volra-darwin-arm64 ./cmd/volra
GOOS=linux GOARCH=amd64 go build -o dist/volra-linux-amd64 ./cmd/volra
# + sha256sum dist/volra-* > dist/checksums.txt
```

**Makefile over GoReleaser for v0.1:** GoReleaser solves homebrew taps, Docker images, snapcraft, changelogs — none of which exist in v0.1. A Makefile with 5 targets is simpler, educational, and constraint #0 compliant. GoReleaser is a v0.2 upgrade when distribution channels expand.

### Initialization Command

```bash
mkdir volra && cd volra
go mod init github.com/romerox3/volra
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
# Agentfile — generated by volra init
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
| `dockerfile` | enum | Yes | `auto` | `auto` = Volra generates. `custom` = uses `./Dockerfile` (must exist, error if not found) |

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
- Agentfile `version` > CLI supported → `❌ Agentfile version 2 requires Volra >= v0.2. You have v0.1.0. Update: curl -sSL https://...`
- Agentfile `version` < CLI current → accepted (backward compatible)

#### .env Lifecycle

**Generated by init:** `.env.example` with variable names and empty values, plus instructive header:

```
# Volra — Environment Variables
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
- **Name change detection:** If `.volra/` exists with a previous deploy and the name has changed → warning: `⚠️ Previous stack '{old-name}' still running. Stop it with: docker compose -p {old-name} down`

#### Re-deploy Strategy (FR25)

Re-running `volra deploy` on an existing deployment:

1. Regenerate `.volra/` completely (idempotent, constraint #4)
2. Execute `docker compose -p {name} up -d --build` — forces agent image rebuild
3. Prometheus and Grafana containers are NOT recreated if their config hasn't changed (Docker Compose native behavior)
4. Prometheus data volumes survive rebuild (NFR9 — named volumes)
5. Health check on the new agent container

### Category 2: Error Handling & Output Strategy

_3 rounds of Party Mode. 15 adjustments consolidated to 12 net decisions. Key: Presenter as sole output channel, stderr/stdout separation, fix-and-re-run philosophy, context propagation._

#### Failure Philosophy

**No cleanup on failure. No rollback. Fix and re-run.**

Volra has zero CLI state (Step 2 constraint). Generated artifacts in `.volra/` are idempotent — the next deploy overwrites them. Docker Compose manages container lifecycle. If a deploy fails mid-way, the user fixes the issue and runs `volra deploy` again. This eliminates an entire category of error handling complexity (rollback logic, partial state cleanup, transaction-like semantics).

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
| Result (summary, URLs, tables) | stdout | Capturable: `volra deploy > result.txt` |

This makes Volra pipe-friendly from v0.1. When `--json` arrives in v0.2, JSON goes to stdout clean.

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
// cmd/volra/main.go
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
- **Ctrl+C behavior:** Signal handler prints `⚠️ Cancelled. Run volra deploy to retry.` via Presenter, then exits with code 130
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
| E201 | setup | No Python project detected | Volra requires requirements.txt or pyproject.toml |
| E202 | setup | No entry point found | Create main.py or specify in Agentfile |
| E203 | setup | Agentfile already exists | Use --force to overwrite |
| E301 | deploy | Docker is not running | Start Docker and try again |
| E302 | deploy | Docker build failed | Check Dockerfile and dependencies. Logs: docker logs {name}-agent-1 |
| E303 | deploy | Health check failed after {timeout}s | Check agent starts on {port}{health_path}. Logs: docker logs {name}-agent-1 |
| E304 | deploy | Agent container OOM killed | Increase Docker memory limit or optimize agent memory usage |
| E305 | deploy | .env not found | Copy .env.example to .env and fill in values |
| E401 | status | No deployment found | Run volra deploy first |
| E402 | status | Docker not running | Start Docker and try again |
| E501 | agentfile | Invalid field: {field} — {problem} | Field-specific fix instruction |
| E502 | agentfile | Unsupported version {n} | Requires Volra >= v{required}. Update: curl -sSL ... |

### Category 3: Docker Compose & Dockerfile Generation

_4 rounds of Party Mode. 19 adjustments consolidated to 12 net decisions. Key: Prometheus 3.x, Grafana 12.x, no Docker HEALTHCHECK, .dockerignore generated, Dockerfile limited to 2 dep managers, TemplateContext struct._

#### Container Images (Pinned)

| Service | Image | Version | Notes |
|---------|-------|---------|-------|
| Prometheus | `prom/prometheus` | `v3.10.0` | Prometheus 3.x — breaking changes from 2.x don't affect basic scrape config |
| Grafana | `grafana/grafana-oss` | `12.4.0` | OSS image (lighter, no enterprise features). NOT `grafana/grafana` |

**Update strategy:** Versions are pinned in the Go binary's embedded templates. Updated with each Volra release. No user-configurable version override in v0.1.

#### docker-compose.yml Template

```yaml
# .volra/docker-compose.yml — Generated by Volra. Do not edit.
# Regenerate with: volra deploy

name: {{.Name}}

services:
  agent:
    build:
      context: ..
      dockerfile: .volra/Dockerfile
    container_name: {{.Name}}-agent
    restart: unless-stopped
    ports:
      - "{{.Port}}:{{.Port}}"
    {{- if .Env}}
    env_file:
      - ../.env
    {{- end}}
    networks:
      - volra

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
      - volra

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
      - volra

volumes:
  prometheus-data:

networks:
  volra:
    driver: bridge
```

#### Compose Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Network** | Dedicated bridge `volra` | Isolation. Services reference each other by service name (`http://agent:{{.Port}}`) |
| **Project name** | `name: {{.Name}}` top-level (mandatory) | Without it, Compose uses directory name (`.volra`) — all projects would collide |
| **Agent build context** | `..` (parent directory) | Dockerfile is in `.volra/`, agent code is in project root |
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

Generated by `volra deploy` in the project root (NOT in `.volra/`) — `.dockerignore` is read relative to build context, which is `..` (project root).

**Rules:**
- If `.dockerignore` already exists → respect it, do not modify
- If not exists → generate with minimal set:

```
.volra/
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
# Generated by Volra. Override with dockerfile: custom in Agentfile.
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
| `requirements.txt` exists (pip) | `COPY requirements.txt` → `pip install --prefix=/install -r` → `COPY . .` | Optimized — deps cached until requirements.txt changes |
| `pyproject.toml` only (pip) | `COPY . .` → `pip install --prefix=/install .` | Not optimized — any file change re-installs |
| `poetry.lock` (poetry) | `pip install poetry` → `COPY pyproject.toml poetry.lock*` → `poetry install --only main --no-root` | Optimized — lockfile cached |
| `uv.lock` (uv) | `COPY --from=ghcr.io/astral-sh/uv` → `COPY pyproject.toml uv.lock*` → `uv pip install --system --prefix=/install` | Optimized — lockfile cached, fast install |
| `Pipfile.lock` (pipenv) | `pip install pipenv` → `COPY Pipfile Pipfile.lock*` → `pipenv install --deploy --system` | Optimized — lockfile cached |

**Auto-detected by lockfile** (priority): `uv.lock` > `poetry.lock` > `Pipfile.lock` > `requirements.txt` > `pyproject.toml`. Override with `package_manager:` field in Agentfile.

**Not supported:** conda. Use `dockerfile: custom` for conda environments.

**ARM64 note:** `python:X.Y-slim` is multi-arch (Docker pulls correct variant automatically). If build fails due to C extension compilation on ARM64, error E302 includes hint: `Tip: Some Python packages lack ARM64 wheels. Try dockerfile: custom with --platform linux/amd64.`

#### Template Context

The Agentfile has ~16 fields. The Dockerfile template needs additional metadata fields detected at deploy time:

```go
// internal/deploy/context.go
type TemplateContext struct {
    agentfile.Agentfile          // Embedded — all fields
    PythonVersion   string       // Detected from pyproject.toml, fallback "3.11"
    EntryPoint      string       // Detected from scanner (main.py/app.py/server.py), fallback "main.py"
    PackageManager  string       // "pip", "poetry", "uv", "pipenv" — from Agentfile or auto-detected by lockfile
    HasRequirements bool         // requirements.txt exists (used for pip branch)
}
```

**Why not store metadata in Agentfile?** Between `init` and `deploy`, the user may change files (rename app.py → main.py, add requirements.txt). Deploy always re-scans for fresh metadata. Stateless — aligned with zero-CLI-state philosophy.

**Why not add fields to Agentfile?** PRD scope rule: "The Agentfile has 6 fields — adding a 7th requires removing one or explicit justification." (Already used that budget for `version`.) PythonVersion and EntryPoint are build-time metadata, not user-facing configuration.

#### dockerfile: custom Path

When Agentfile has `dockerfile: custom`:
1. Deploy skips Dockerfile generation entirely
2. Deploy reads `./Dockerfile` from project root (validated in Agentfile semantic validation — error if not found)
3. Compose template uses `dockerfile: ../Dockerfile` instead of `.volra/Dockerfile`
4. All other generation proceeds normally (compose, prometheus, grafana)

This is the escape hatch for: conda projects, multi-stage builds, non-standard project structures, ARM64-specific builds, any case where auto-generated Dockerfile is insufficient.

### Category 4: Prometheus & Grafana Configuration

_2 rounds of Party Mode. 10 adjustments consolidated to 8 net decisions. Critical bug fixes: corrected PromQL metrics (scrape_duration_seconds, not probe_duration_seconds), alert timing aligned with NFR11, HasMetricsEndpoint detection eliminated._

#### Shared Constants

Values that appear across multiple generated artifacts. Defined once in `internal/deploy/constants.go`:

```go
const (
    JobHealth      = "agent-health"    // prometheus.yml, alert_rules.yml, dashboards
    JobMetrics     = "agent-metrics"   // prometheus.yml
    DatasourceName = "Prometheus"       // datasource.yml, dashboards
    NetworkName    = "volra"       // docker-compose.yml
)
```

#### prometheus.yml Template

```yaml
# .volra/prometheus.yml — Generated by Volra. Do not edit.
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
# .volra/alert_rules.yml — Generated by Volra.
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
  - name: 'Volra'
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

**No atomic writes.** Generated artifacts are disposable — if a write fails mid-way, the user runs `volra deploy` again. Fix-and-re-run philosophy.

**Respect existing files:** `deploy` generates `.dockerignore` only if one doesn't already exist. Uses `os.Stat` to check before writing.

### Template Patterns

```go
// Template field names match Go struct fields exactly
// TemplateContext.Name, TemplateContext.Port, TemplateContext.HealthPath
const dockerComposeTemplate = `# Generated by Volra — do not edit manually
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
1. Generated file comment on first line: `# Generated by Volra — do not edit manually`
2. Template field names = Go struct field names (no mapping layer)
3. Use `{{- }}` and `{{ -}}` for whitespace control
4. Templates are `go:embed` strings, not external files
5. `text/template` only — never `html/template` (we generate YAML/Dockerfile, not HTML)

### Cobra Wiring Pattern

```go
// cmd/volra/deploy.go — WIRING ONLY, zero business logic
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
8. **All generated files start with `# Generated by Volra`** — user knows what's safe to delete

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
| CLI Wiring | `cmd/volra/` | — | `main.go`, `doctor.go`, `deploy.go`, `init.go`, `status.go` |
| Build & Release | Root / `.github/` | FR38-FR41 | `Makefile`, `install.sh`, `ci.yml`, `release.yml` |

### Complete Project Directory Structure

_This is the **definitive** project tree — supersedes the preliminary tree in Starter Template Evaluation (Step 3)._

```
volra/
├── cmd/
│   └── volra/
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
├── .gitignore                               # /volra, /dist/, .volra/
├── go.mod                                   # module github.com/romerox3/volra
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

What Volra generates in the user's project directory:

```
my-agent/                                    # User's project root
├── Agentfile                                # Created by `volra init`, persists across deploys
├── .env                                     # Optional — user creates/manages, never generated
├── .gitignore                               # Updated by `volra init` — adds .volra/ entry (FR14)
├── .dockerignore                            # Generated by `volra deploy` (if not exists)
│                                            # MUST exclude .volra/ from Docker build context
├── .volra/                             # Generated by `volra deploy` — FULLY REGENERATED each time
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
├── main.py                                  # User's agent code (untouched by Volra)
├── requirements.txt                         # User's dependencies (untouched)
└── ...
```

**Docker Compose invocation:** `docker compose -f .volra/docker-compose.yml -p {name} up -d`

**Build context resolution:** `docker-compose.yml` uses `build: { context: .., dockerfile: .volra/Dockerfile }`. The `..` resolves to the project root (relative to the compose file in `.volra/`), so `COPY requirements.txt .` in the Dockerfile accesses user source files correctly.

**`.dockerignore` contents** (generated by deploy if not exists):
```
.volra/
.git/
.env
__pycache__/
*.pyc
```

### Package Dependency Graph

```
cmd/volra/
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

### Data Flow: `volra deploy`

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
                              │                         ├── writeAll(.volra/)│
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
BINARY    := volra
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS   := -s -w -X main.version=$(VERSION)
# Ubuntu AMD64 and Debian AMD64 share linux/amd64 binary (2 combos = 3 PRD targets)
PLATFORMS := darwin/arm64 linux/amd64

.PHONY: build test test-integration lint build-all checksums clean

build:               ## Build for current platform
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/volra/

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
		go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-$$os-$$arch ./cmd/volra/; \
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
| Generated files marked | Template content | First line: `# Generated by Volra` |

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
- Generated output in `.volra/` — isolated from user code, fully regenerated per deploy

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
volra/
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
- [x] Generated output structure (.volra/) documented
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
10. `cmd/volra/` — Cobra wiring, assembled last

---

## v1.1 Architecture Extension

_Mode: EXTEND. Existing architecture is preserved. This section adds new decisions and patterns for v1.1 features (FR42-FR44). Discovered during E2E testing with 8 agent systems of increasing complexity._

### Extension 1: Configurable Health Timeout (FR42)

#### Agentfile Schema Addition

```yaml
# Agentfile — v1.1 optional field
version: 1
name: my-ml-agent
framework: generic
port: 8000
health_path: /health
health_timeout: 300        # NEW — optional, seconds, default 60
env:
  - OPENAI_API_KEY
dockerfile: auto
```

| Field | Type | Required | Default | Validation |
|-------|------|----------|---------|-----------|
| `health_timeout` | integer | No | `60` | Range: 10-600 seconds. Must be > 0. Zero value means "use default (60s)" |

**Backward compatibility:** Field is optional with `yaml:"health_timeout,omitempty"`. Agentfiles without this field get the current 60s default behavior. Zero value in Go (0) maps to the default — no behavior change for existing users.

#### Go Type Changes

```go
// internal/agentfile/agentfile.go — extended
type Agentfile struct {
    Version       int            `yaml:"version"`
    Name          string         `yaml:"name"`
    Framework     Framework      `yaml:"framework"`
    Port          int            `yaml:"port"`
    HealthPath    string         `yaml:"health_path"`
    HealthTimeout int            `yaml:"health_timeout,omitempty"` // NEW: seconds, 0 = default 60
    Env           []string       `yaml:"env,omitempty"`
    Dockerfile    DockerfileMode `yaml:"dockerfile"`
}
```

#### Validation Rules

```go
// internal/agentfile/validate.go — new function
func validateHealthTimeout(timeout int) error {
    if timeout == 0 {
        return nil // default, not specified
    }
    if timeout < 10 || timeout > 600 {
        return &output.UserError{
            Code: output.CodeInvalidAgentfile,
            What: fmt.Sprintf("Invalid field: health_timeout — %d is out of range (10-600)", timeout),
            Fix:  "Set health_timeout between 10 and 600 seconds, or remove the field for default (60s)",
        }
    }
    return nil
}
```

#### Health Check Propagation

```go
// internal/deploy/healthcheck.go — updated
const (
    healthRetryInterval = 2 * time.Second
    defaultHealthTimeout = 60 * time.Second
)

// WaitForHealth now accepts configurable timeout.
func WaitForHealth(ctx context.Context, port int, healthPath string, name string, timeout time.Duration, p output.Presenter) error {
    if timeout == 0 {
        timeout = defaultHealthTimeout
    }
    deadline := time.Now().Add(timeout)
    // ... rest unchanged
}
```

**Caller change in deploy.go:**

```go
// Resolve timeout: Agentfile value → default
healthTimeout := time.Duration(af.HealthTimeout) * time.Second
if err := WaitForHealth(ctx, af.Port, af.HealthPath, af.Name, healthTimeout, p); err != nil {
    return err
}
```

#### Docker Compose Health Check

The `health_timeout` also propagates to the Docker Compose agent service healthcheck:

```yaml
# docker-compose.yml.tmpl — agent service addition
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:{{.Port}}{{.HealthPath}}"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: {{.HealthTimeoutStr}}
```

Where `HealthTimeoutStr` is computed from the Agentfile's `health_timeout` (default: "60s").

### Extension 2: Multi-Stage Dockerfile Builds (FR43)

#### Template Redesign

The current single-stage Dockerfile template is replaced with a multi-stage pattern:

```dockerfile
# Generated by Volra — override with dockerfile: custom in Agentfile
# Stage 1: Builder
FROM python:{{.PythonVersion}}-slim AS builder

WORKDIR /build

{{if .HasRequirements -}}
COPY requirements.txt .
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --prefix=/install -r requirements.txt
{{- else -}}
COPY . .
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --prefix=/install .
{{- end}}

# Stage 2: Runtime
FROM python:{{.PythonVersion}}-slim

WORKDIR /app

COPY --from=builder /install /usr/local

{{if .HasRequirements -}}
COPY . .
{{- else -}}
COPY . .
{{- end}}

EXPOSE {{.Port}}

CMD ["python", "{{.EntryPoint}}"]
```

**Key improvements:**
- Builder stage installs dependencies, runtime stage only copies results
- `--mount=type=cache,target=/root/.cache/pip` enables pip cache across builds (BuildKit required)
- `--prefix=/install` installs to a clean directory for selective copy
- Final image contains only runtime dependencies, not build tools
- Image size reduction: typically 40-60% smaller

**BuildKit requirement:** The `--mount=type=cache` syntax requires BuildKit. Docker 23.0+ uses BuildKit by default. For older versions, set `DOCKER_BUILDKIT=1`. The `doctor` command already checks Docker version.

**No TemplateContext changes needed** — the same 3 metadata fields (PythonVersion, EntryPoint, HasRequirements) drive the new template.

### Extension 3: Custom Metrics Dashboard Panels (FR44)

#### Detection Strategy

At deploy time, `BuildContext` checks whether the agent project uses `prometheus_client`:

```go
// internal/deploy/context.go — extended
type TemplateContext struct {
    agentfile.Agentfile
    PythonVersion   string
    EntryPoint      string
    HasRequirements bool
    HasMetrics      bool   // NEW: true if prometheus_client detected in dependencies
}
```

Detection logic:
1. Check `requirements.txt` for `prometheus_client` or `prometheus-client`
2. Check `pyproject.toml` for the same in `[project.dependencies]` or `[tool.poetry.dependencies]`

```go
// internal/deploy/context.go
func detectMetricsLibrary(dir string) bool {
    // Check requirements.txt
    if data, err := os.ReadFile(filepath.Join(dir, "requirements.txt")); err == nil {
        content := strings.ToLower(string(data))
        if strings.Contains(content, "prometheus_client") || strings.Contains(content, "prometheus-client") {
            return true
        }
    }
    // Check pyproject.toml
    if data, err := os.ReadFile(filepath.Join(dir, "pyproject.toml")); err == nil {
        content := strings.ToLower(string(data))
        if strings.Contains(content, "prometheus_client") || strings.Contains(content, "prometheus-client") {
            return true
        }
    }
    return false
}
```

#### Dashboard Architecture Decision

**Decision: Dashboards remain static JSON files, but we add conditional panels.**

The key architectural question was: should dashboards become templates (.json.tmpl)?

**Answer: No.** Grafana JSON is deeply nested and not template-friendly. Instead:
- Keep `overview.json` and `detail.json` as static files
- Add `overview_metrics.json` and `detail_metrics.json` as alternative static files with additional custom metrics panels
- At deploy time, copy the appropriate variant based on `HasMetrics`

This avoids template complexity while supporting both scenarios:

```go
// internal/deploy/grafana.go — updated selection logic
func selectDashboard(name string, hasMetrics bool) string {
    if hasMetrics {
        return "static/" + name + "_metrics.json"
    }
    return "static/" + name + ".json"
}
```

#### Custom Metrics Panels

When `HasMetrics` is true, the dashboards include additional panels:

**Overview (overview_metrics.json) — 2 additional panels:**

| Panel | Type | PromQL | Notes |
|-------|------|--------|-------|
| Request Rate | Stat | `sum(rate(http_requests_total{job="agent-metrics"}[5m]))` | Requests per second (common prometheus_client metric) |
| Active Requests | Stat | `sum(http_requests_in_progress{job="agent-metrics"})` | Currently processing |

**Detail (detail_metrics.json) — 3 additional panels:**

| Panel | Type | PromQL | Notes |
|-------|------|--------|-------|
| Request Rate Over Time | Time series | `sum(rate(http_requests_total{job="agent-metrics"}[5m]))` | Line chart |
| Request Duration | Time series | `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{job="agent-metrics"}[5m]))` | P95 latency |
| Custom Metrics | Table | `{job="agent-metrics", __name__!~"http_.*\|process_.*\|python_.*"}` | Any non-standard metrics from the agent |

**Panel descriptions clearly indicate:** "Application-reported metric — requires prometheus_client instrumentation in agent code."

#### Prometheus Configuration

The existing `prometheus.yml.tmpl` already has an `agent-metrics` job that scrapes `/metrics`. No changes needed — it always scrapes, and if no `/metrics` endpoint exists, it silently gets 404s (by design).

### v1.1 Test Strategy

#### New Test Fixtures

```
internal/agentfile/testdata/
  valid_health_timeout.yaml       # version: 1, health_timeout: 300
  valid_no_health_timeout.yaml    # version: 1, no health_timeout field (backward compat)
  invalid_health_timeout_low.yaml # health_timeout: 5 (below 10)
  invalid_health_timeout_high.yaml # health_timeout: 700 (above 600)
```

#### Golden File Updates

All existing golden files are regenerated with the new multi-stage Dockerfile template. New golden files added:

```
internal/deploy/testdata/golden/
  dockerfile_requirements.golden    # UPDATED: multi-stage with requirements.txt
  dockerfile_pyproject.golden       # UPDATED: multi-stage with pyproject.toml
  compose_minimal.golden            # UNCHANGED (health_timeout is optional)
  compose_health_timeout.golden     # NEW: with custom health_timeout
```

#### Key Test Cases

1. **Backward compatibility:** Existing v1.0 Agentfile fixtures parse without errors, health_timeout defaults to 0 (→ 60s behavior)
2. **Health timeout validation:** Out-of-range values rejected, zero allowed, boundary values (10, 600) accepted
3. **Multi-stage Dockerfile:** Golden file comparison for both requirements.txt and pyproject.toml paths
4. **HasMetrics detection:** Positive/negative cases for requirements.txt and pyproject.toml
5. **Dashboard selection:** Correct JSON file copied based on HasMetrics flag

---

## v1.2 Architecture Extension

_Mode: EXTEND. This section adds decisions and patterns for v1.2 features (FR45-FR46). Builds on v1.1 architecture — all existing patterns preserved._

### Extension 4: Persistent Volume Mounts (FR45)

#### Agentfile Schema Addition

```yaml
# Agentfile — v1.2 optional field
version: 1
name: my-ml-agent
framework: generic
port: 8000
health_path: /health
health_timeout: 300
volumes:                   # NEW — optional, list of absolute paths
  - /data
  - /models
env:
  - OPENAI_API_KEY
dockerfile: auto
```

| Field | Type | Required | Default | Validation |
|-------|------|----------|---------|-----------|
| `volumes` | list of strings | No | `[]` | Each path absolute (starts with `/`), no duplicates, no `/app` or `/app/*` (conflicts with WORKDIR), max 10 entries |

**Backward compatibility:** Field is optional with `yaml:"volumes,omitempty"`. Agentfiles without this field produce identical output to v1.0/v1.1.

#### Go Type Changes

```go
// internal/agentfile/agentfile.go — extended
type Agentfile struct {
    Version       int            `yaml:"version"`
    Name          string         `yaml:"name"`
    Framework     Framework      `yaml:"framework"`
    Port          int            `yaml:"port"`
    HealthPath    string         `yaml:"health_path"`
    HealthTimeout int            `yaml:"health_timeout,omitempty"`
    Volumes       []string       `yaml:"volumes,omitempty"` // NEW: absolute mount paths
    Env           []string       `yaml:"env,omitempty"`
    Dockerfile    DockerfileMode `yaml:"dockerfile"`
}
```

#### Validation Rules

```go
// internal/agentfile/validate.go — new function
func validateVolumes(volumes []string) error {
    if len(volumes) > 10 {
        return &output.UserError{
            Code: output.CodeInvalidAgentfile,
            What: "Invalid field: volumes — too many entries (max 10)",
            Fix:  "Reduce the number of volume mounts to 10 or fewer",
        }
    }
    seen := make(map[string]bool)
    for _, v := range volumes {
        if v == "" {
            return error: "empty entry"
        }
        if !strings.HasPrefix(v, "/") {
            return error: "must be absolute path"
        }
        if v == "/app" || strings.HasPrefix(v, "/app/") {
            return error: "conflicts with container WORKDIR /app"
        }
        if seen[v] {
            return error: "duplicate"
        }
        seen[v] = true
    }
    return nil
}
```

#### TemplateContext Extension

```go
// internal/deploy/context.go — new type
type VolumeSpec struct {
    Name      string // Docker named volume, e.g., "my-agent-data"
    MountPath string // Container mount path, e.g., "/data"
}

// TemplateContext extended
type TemplateContext struct {
    agentfile.Agentfile
    PythonVersion   string
    EntryPoint      string
    HasRequirements bool
    HasMetrics      bool
    VolumeSpecs     []VolumeSpec // NEW: computed from Volumes + Name
}
```

Volume name generation: `{agent-name}-{sanitized-path}` where the path is cleaned of leading `/` and `/` replaced with `-`. Example: volume `/data/models` on agent `my-agent` → `my-agent-data-models`.

#### Docker Compose Template Changes

```yaml
# docker-compose.yml.tmpl — agent service addition
services:
  agent:
    # ... existing fields ...
    {{- if .VolumeSpecs}}
    volumes:
      {{- range .VolumeSpecs}}
      - {{.Name}}:{{.MountPath}}
      {{- end}}
    {{- end}}

# Top-level volumes (extend existing)
volumes:
  prometheus-data:
  {{- range .VolumeSpecs}}
  {{.Name}}:
  {{- end}}
```

#### Architecture Decision: Simple Path List vs Named Map

**Decision:** Simple path list (`[]string`) for v1.2. Named map (`map[string]string`) deferred to v2.0 if needed.

**Rationale:** (1) Covers 80% use case — "I want /data to persist". (2) Volume naming is deterministic from agent name + path. (3) Aligns with YAGNI — no evidence users need custom volume names. (4) If v2.0 introduces `services` with `volumes` on service containers, the map format may be needed then.

#### Architecture Decision: Block /app Paths

**Decision:** Reject any volume path that is `/app` or starts with `/app/`.

**Rationale:** Docker volumes mount at container start, AFTER the image is built. A volume on `/app` would shadow the `COPY . .` content, meaning the agent code would be invisible. This is a silent, hard-to-debug failure. Blocking it at validation prevents the issue entirely.

### Extension 5: LLM Token Tracking Panels (FR46)

#### Strategy: Extend Existing Dashboard Variants

**Decision:** Add LLM-specific panels to the existing `overview_metrics.json` and `detail_metrics.json` files. No new dashboard files or detection mechanisms.

**Rationale:** (1) Build-time detection cannot determine which metrics an agent exposes — only that it uses prometheus_client. (2) Grafana handles missing metrics gracefully ("No data" display). (3) Eliminates need for HasLLMMetrics boolean, new file variants, and combinatorial explosion. (4) Party Mode convergence: simpler is better.

#### Volra LLM Metrics Convention

Recommended metric names for AI/LLM agents. These are documented conventions, not enforced requirements:

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `llm_tokens_total` | Counter | `direction` (input\|output), `model` | Total tokens processed |
| `llm_request_cost_dollars_total` | Counter | `model` | Cumulative cost in USD |
| `llm_model_requests_total` | Counter | `model` | Requests per model |

Agents that expose these metrics automatically benefit from dedicated dashboard panels. Agents that don't expose them see "No data" in those panels — no errors, no configuration needed.

#### Dashboard Panel Additions

**overview_metrics.json** — 1 new panel (total: 6):

| Panel | Type | PromQL | Notes |
|-------|------|--------|-------|
| LLM Token Rate | stat | `sum(rate(llm_tokens_total{job="agent-metrics"}[5m]))` | Tokens per second, both directions |

**detail_metrics.json** — 3 new panels (total: 10):

| Panel | Type | PromQL | Notes |
|-------|------|--------|-------|
| Token Consumption Over Time | timeseries | `sum(rate(llm_tokens_total{job="agent-metrics"}[5m])) by (direction)` | Input vs output token rates |
| LLM Cost Trending | timeseries | `sum(increase(llm_request_cost_dollars_total{job="agent-metrics"}[1h]))` | Hourly cost accumulation |
| Per-Model Request Breakdown | bargauge | `sum(rate(llm_model_requests_total{job="agent-metrics"}[5m])) by (model)` | Model comparison |

All LLM panels include description: `"Volra LLM Metrics Convention — shows 'No data' if agent does not expose this metric"`

#### Files Changed

**Story 7.1 (Volumes):**
- `internal/agentfile/agentfile.go` — add `Volumes []string` field
- `internal/agentfile/validate.go` — add `validateVolumes()`
- `internal/deploy/context.go` — add `VolumeSpec` type, compute in `BuildContext`
- `internal/deploy/templates/docker-compose.yml.tmpl` — conditional volumes
- New test fixtures: `valid_volumes.yaml`, `invalid_volumes_not_absolute.yaml`, `invalid_volumes_app_path.yaml`, `invalid_volumes_duplicate.yaml`
- New golden file: `compose_volumes.golden`

**Story 7.2 (LLM Panels):**
- `internal/deploy/static/overview_metrics.json` — add 1 LLM panel
- `internal/deploy/static/detail_metrics.json` — add 3 LLM panels
- `internal/deploy/grafana_test.go` — extend metrics tests for LLM panels

### v1.2 Test Strategy

#### New Test Fixtures

```
internal/agentfile/testdata/
  valid_volumes.yaml                # volumes: [/data, /models]
  invalid_volumes_not_absolute.yaml # volumes: [data] (no leading /)
  invalid_volumes_app_path.yaml     # volumes: [/app/data] (conflicts with WORKDIR)
  invalid_volumes_duplicate.yaml    # volumes: [/data, /data]
```

#### Golden File Updates

```
internal/deploy/testdata/golden/
  compose_volumes.golden            # NEW: with agent volumes + top-level declarations
```

#### Key Test Cases

1. **Volumes validation:** Absolute path required, /app blocked, no duplicates, max 10, empty list OK
2. **VolumeSpec computation:** Name generation from agent name + path sanitization
3. **Docker Compose volumes:** Golden file with volumes, existing golden files unchanged (backward compat)
4. **LLM dashboard panels:** Assert new panels present in overview_metrics.json and detail_metrics.json
5. **Backward compatibility:** v1.0/v1.1 Agentfile fixtures parse without errors, Volumes defaults to nil

---

## v2.0 Architecture Extension

_Mode: EXTEND. This section adds new decisions and patterns for v2.0 features (FR47, FR48, FR50). FR49 (streaming probes) is DEFERRED. Schema version remains 1._

### Extension 6: Infrastructure Services (FR47)

#### Schema Design

**ADR: Services use `map[string]Service` — explicit definitions, no presets**

Users specify exactly what services they need. No magic presets. The map key is the service name.

```yaml
# Agentfile — v2.0 optional field
version: 1
name: my-agent
framework: generic
port: 8000
health_path: /health
services:                          # NEW — optional, map of service definitions
  redis:
    image: redis:7-alpine
  db:
    image: postgres:16
    port: 5432
    volumes:
      - /var/lib/postgresql/data
    env:
      - POSTGRES_PASSWORD
```

#### Go Types

```go
// Service represents an infrastructure service declared in the Agentfile.
type Service struct {
    Image   string   `yaml:"image"`
    Port    int      `yaml:"port,omitempty"`    // 0 = internal only (no host port)
    Volumes []string `yaml:"volumes,omitempty"` // absolute paths, same rules as agent volumes (except /app check)
    Env     []string `yaml:"env,omitempty"`     // env var names, values from .env file
}

// In Agentfile struct:
Services map[string]Service `yaml:"services,omitempty"`
```

**ADR: ServiceContext sorted by name for deterministic compose output**

Go map iteration is non-deterministic. ServiceContexts must be sorted by name to ensure golden file tests are stable.

```go
// ServiceContext is the pre-computed deploy-time representation of a Service.
type ServiceContext struct {
    Name        string       // map key, used as compose service name suffix
    Image       string
    Port        int
    Env         []string
    VolumeSpecs []VolumeSpec // computed from agentName + serviceName + volume paths
}

// In TemplateContext:
ServiceContexts []ServiceContext
```

#### Validation Rules

```go
func validateServices(services map[string]Service, agentPort int) error
```

1. Max 5 services
2. Service name: must match `dnsLabelRegex` (reuse from name validation)
3. Reserved names blocked: `agent`, `prometheus`, `grafana`, `blackbox`
4. Image: required, non-empty
5. Port: 0-65535. Must not conflict with agent port, 9090 (prometheus), 3001 (grafana)
6. Port uniqueness across all services
7. Volumes: reuse volume path validation (absolute, no dupes within service)
8. Env: reuse env validation (non-empty, no dupes within service)

#### Docker Compose Template Changes

Services are inserted between agent and blackbox in the compose template:

```yaml
{{- range .ServiceContexts}}
  {{$.Agentfile.Name}}-{{.Name}}:
    image: {{.Image}}
    container_name: {{$.Agentfile.Name}}-{{.Name}}
    restart: unless-stopped
{{- if .Port}}
    ports:
      - "{{.Port}}:{{.Port}}"
{{- end}}
{{- if .Env}}
    env_file:
      - ../.env
{{- end}}
{{- if .VolumeSpecs}}
    volumes:
{{- range .VolumeSpecs}}
      - {{.Name}}:{{.MountPath}}
{{- end}}
{{- end}}
    networks:
      - volra
{{- end}}
```

Agent service gets `depends_on` for all declared services:

```yaml
{{- if .ServiceContexts}}
    depends_on:
{{- range .ServiceContexts}}
      - {{$.Agentfile.Name}}-{{.Name}}
{{- end}}
{{- end}}
```

Service volumes are added to the top-level volumes section alongside agent and prometheus volumes.

#### Volume Naming Convention

Service volumes follow: `{projectName}-{serviceName}-{sanitized-path}`

Example: project "my-agent", service "db", path "/var/lib/postgresql/data" → `my-agent-db-var-lib-postgresql-data`

Uses same sanitization as agent volumes (TrimPrefix "/", ReplaceAll "/" → "-").

#### Test Strategy

```
internal/agentfile/testdata/
  valid_services.yaml              # services with redis + postgres
  invalid_services_reserved.yaml   # service name "agent" (reserved)
  invalid_services_no_image.yaml   # service without image field
  invalid_services_port_conflict.yaml  # service port same as agent
  invalid_services_too_many.yaml   # 6 services (max 5)
```

Golden file: `compose_services.golden` — compose output with 2 services

### Extension 7: Container Runtime Configuration (FR48 + FR50)

#### Security Context (FR48)

```yaml
# Agentfile — v2.0 optional field
version: 1
name: my-agent
framework: generic
port: 8000
health_path: /health
security:                          # NEW — optional
  read_only: true
  no_new_privileges: true
  drop_capabilities: [ALL]
```

```go
type SecurityContext struct {
    ReadOnly         bool     `yaml:"read_only,omitempty"`
    NoNewPrivileges  bool     `yaml:"no_new_privileges,omitempty"`
    DropCapabilities []string `yaml:"drop_capabilities,omitempty"`
}

// In Agentfile struct — pointer for nil check:
Security *SecurityContext `yaml:"security,omitempty"`
```

Docker Compose mapping:
```yaml
{{- if .Security}}
{{- if .Security.ReadOnly}}
    read_only: true
{{- end}}
{{- if .Security.NoNewPrivileges}}
    security_opt:
      - no-new-privileges:true
{{- end}}
{{- if .Security.DropCapabilities}}
    cap_drop:
{{- range .Security.DropCapabilities}}
      - {{.}}
{{- end}}
{{- end}}
{{- end}}
```

Validation: minimal. Docker validates capabilities at runtime. No Agentfile-level validation of capability names.

#### GPU/Hardware Acceleration (FR50)

**ADR: GPU field is boolean, defaults to nvidia driver**

Only nvidia-container-toolkit is mature for Docker. AMD ROCm support is experimental. Simple boolean avoids premature abstraction.

```yaml
gpu: true   # NEW — optional, default false
```

```go
// In Agentfile struct:
GPU bool `yaml:"gpu,omitempty"`
```

Docker Compose:
```yaml
{{- if .GPU}}
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: all
              capabilities: [gpu]
{{- end}}
```

Validation: none. Boolean field.

Testing: template-only verification (can't E2E test without GPU hardware). Assert compose output contains `nvidia` and `capabilities: [gpu]`.

#### Test Fixtures

```
internal/agentfile/testdata/
  valid_security.yaml    # security with all 3 fields
  valid_gpu.yaml         # gpu: true
```

#### Field Count Summary

After v2.0: 12 Agentfile fields (version, name, framework, port, health_path, health_timeout, volumes, env, dockerfile, services, security, gpu)

#### Backward Compatibility

All v1.0, v1.1, and v1.2 Agentfile fixtures MUST parse without errors with the v2.0 CLI. New fields default to zero values (nil maps, nil pointers, false booleans)

## v0.2 Architecture Extensions (Post-Market-Research Strategic Pivot)

*Added March 2026 after market research revealing Railway ($100M), Vercel AI Platform, and Aegra as competitors. These extensions implement the "Own your agent infrastructure" differentiation strategy. Designed via Party Mode (Lisa/Frink/Homer) with sequential thinking consensus.*

### Extension 8: Agent Observability Level 2

**ADR: Prometheus-native over OpenTelemetry for v0.2**

OpenTelemetry is the industry standard for traces but adds heavy dependencies (Go SDK + trace storage backend like Jaeger/Tempo + additional containers). For v0.2, Prometheus-native metrics deliver the key insights (token count, cost, latency) without new infrastructure. Migration to OpenTelemetry is a v0.3+ consideration.

**ADR: Framework-agnostic instrumentation over framework-specific hooks**

Volra's core differentiator is being framework-agnostic. Level 2 observability MUST work with any Python agent, not just LangGraph. Instrumentation targets LLM provider SDKs (OpenAI, Anthropic, etc.) rather than agent framework internals. Optional framework-specific addons (LangGraph graph metrics, CrewAI crew metrics) are deferred to v0.3.

**New Agentfile Field: `observability`**

```yaml
# Agentfile
observability:         # NEW — optional, entire struct
  level: 2             # 1 (default) = probe only, 2 = LLM metrics
  metrics_port: 9101   # default 9101, port for metrics endpoint
```

```go
type ObservabilityConfig struct {
    Level       int `yaml:"level,omitempty"`        // 1 or 2, default 1
    MetricsPort int `yaml:"metrics_port,omitempty"` // default 9101
}

type Agentfile struct {
    // ... existing fields ...
    Observability *ObservabilityConfig `yaml:"observability,omitempty"` // NEW
}
```

Validation:
- `level` must be 1 or 2. Default: 1.
- `metrics_port` must be valid port range (1-65535). Default: 9101.
- `level: 2` is valid with ANY framework (generic or langgraph). No framework constraint.

**New Python Package: `volra-observe`**

Separate repository. PyPI distribution. Works with any Python agent that makes LLM calls:

```python
# Option 1: Automatic instrumentation (patches OpenAI/Anthropic SDKs)
import volra_observe
volra_observe.init(port=9101)  # starts Prometheus HTTP server, patches LLM SDKs

# Option 2: Decorator for manual instrumentation
from volra_observe import track_llm

@track_llm(model="gpt-4o")
def call_llm(prompt: str) -> str:
    return openai_client.chat.completions.create(...)

# Option 3: Context manager
from volra_observe import llm_context

with llm_context(model="claude-sonnet-4-20250514"):
    response = anthropic_client.messages.create(...)
```

Core metrics exposed on `:9101/metrics` (framework-agnostic):
- `volra_llm_tokens_total{model, direction}` — token count (input/output)
- `volra_llm_cost_dollars_total{model}` — estimated cost (embedded pricing table)
- `volra_llm_request_duration_seconds{model, status}` — LLM call latency histogram
- `volra_llm_errors_total{model, error_type}` — errors by type (rate_limit, timeout, api_error)
- `volra_tool_calls_total{tool_name}` — tool/function call count

Cost calculation uses embedded pricing table:
```python
COST_PER_1K_TOKENS = {
    "gpt-4o": {"input": 0.005, "output": 0.015},
    "gpt-4o-mini": {"input": 0.00015, "output": 0.0006},
    "claude-sonnet-4-20250514": {"input": 0.003, "output": 0.015},
    "claude-haiku-4-5-20251001": {"input": 0.0008, "output": 0.004},
}
```
Table updated with each package release. Override via `~/.volra/pricing.json` deferred to v0.3.

**SDK Instrumentation Strategy:**

Auto-patching targets (v0.2):
- `openai` Python SDK — patch `chat.completions.create()` and `completions.create()`
- `anthropic` Python SDK — patch `messages.create()`

Auto-patching targets (v0.3+):
- `google-generativeai` — Gemini
- `litellm` — unified proxy (covers 100+ providers)
- LangGraph callback handler (framework-specific addon)
- CrewAI telemetry hook (framework-specific addon)

**CLI Changes:**

1. **Prometheus config**: When `observability.level: 2`, add scrape target:
```yaml
- job_name: 'agent-level2'
  static_configs:
    - targets: ['{{.Name}}:{{.ObservabilityMetricsPort}}']
  scrape_interval: 15s
```

2. **Grafana dashboards**: New variant `agent-level2-overview.json` with panels:
   - Token Rate (rate of `volra_llm_tokens_total`)
   - Cost Trending (`rate(volra_llm_cost_dollars_total[1h])` by model)
   - Token Usage (stacked bar: input vs output by model)
   - Cost per Model (pie chart)
   - LLM Latency P50/P95/P99 (`volra_llm_request_duration_seconds`)
   - Error Rate by Type (`rate(volra_llm_errors_total[5m])`)
   - Daily Cost Gauge (with configurable threshold)
   - Tool Call Frequency (`rate(volra_tool_calls_total[5m])`)

3. **`volra doctor`**: When `level: 2`, add check:
   - Verify `:9101/metrics` responds (warns if endpoint not reachable: "Install volra-observe to enable Level 2 metrics")

**Template Context Extension:**

```go
type TemplateContext struct {
    // ... existing ...
    ObservabilityLevel       int  // 1 or 2
    ObservabilityMetricsPort int  // resolved port
    HasLevel2                bool // shorthand for Level >= 2
}
```

**Test Strategy:**
- Unit tests: `ObservabilityConfig` parsing, validation (no framework constraint), template context resolution
- Golden file tests: prometheus.yml with Level 2 scrape target, new dashboard JSON
- No E2E test for Level 2 (requires Python package + LLM API keys — integration test)

---

### Extension 9: Volra MCP Server

**ADR: Direct protocol implementation over MCP SDK**

MCP over stdio is JSON-RPC 2.0 — approximately 200 lines of Go to implement the protocol layer. Using an external SDK adds dependency risk for a simple protocol. Direct implementation gives full control and zero dependencies.

**New Command: `volra mcp`**

```
volra mcp    # Start MCP server on stdio (for editor integration)
```

Reads JSON-RPC from stdin, writes to stdout. Stderr for debug logs.

**Editor Configuration Example (Cursor/Claude Code):**
```json
{
  "mcpServers": {
    "volra": {
      "command": "volra",
      "args": ["mcp"]
    }
  }
}
```

**Package Structure:**

```
internal/mcp/
  server.go       — JSON-RPC server loop (stdin reader → dispatch → stdout writer)
  handler.go      — Tool registry + dispatch (map[string]ToolHandler)
  tools.go        — Tool definitions (name, description, inputSchema per MCP spec)
  protocol.go     — MCP protocol types (InitializeRequest, ToolCallRequest, etc.)
```

**Tools Exposed:**

| Tool | Description | Parameters | Returns |
|------|-------------|------------|---------|
| `volra_deploy` | Deploy agent from directory | `path` (string, default cwd) | Deploy result (status, URLs, timing) |
| `volra_status` | Check agent health | `path` (string, default cwd) | Health summary (agent + services) |
| `volra_logs` | Get recent agent logs | `path` (string), `lines` (int, default 50) | Log output (string) |
| `volra_doctor` | Run diagnostics | none | Check results (pass/fail list) |

**Implementation Pattern:**

```go
// server.go
func Serve(ctx context.Context, in io.Reader, out io.Writer) error {
    scanner := bufio.NewScanner(in)
    encoder := json.NewEncoder(out)
    for scanner.Scan() {
        var req jsonrpc.Request
        json.Unmarshal(scanner.Bytes(), &req)
        resp := dispatch(ctx, req)
        encoder.Encode(resp)
    }
    return scanner.Err()
}
```

**Cobra Integration:**

```go
// cmd/mcp.go
var mcpCmd = &cobra.Command{
    Use:   "mcp",
    Short: "Start MCP server for editor integration",
    RunE: func(cmd *cobra.Command, args []string) error {
        return mcp.Serve(cmd.Context(), os.Stdin, os.Stdout)
    },
}
```

**HTTP/SSE Transport:** Deferred to v0.3 (for remote management scenarios).

**Test Strategy:**
- Unit tests: JSON-RPC parsing, tool dispatch, response formatting
- Integration tests: pipe stdin/stdout in test, verify tool call → response cycle
- No E2E: MCP server is tested via protocol simulation, not actual editor

---

### Extension: `volra logs` Command

**Minimal command — not a full architecture extension.**

```go
// cmd/logs.go
var logsCmd = &cobra.Command{
    Use:   "logs [service]",
    Short: "Stream logs from deployed agent",
    RunE: func(cmd *cobra.Command, args []string) error {
        af, err := agentfile.Load(".")
        if err != nil { return err }
        service := af.Name // default: agent container
        if len(args) > 0 { service = args[0] }
        composePath := filepath.Join(".volra", "docker-compose.yml")
        return docker.StreamLogs(cmd.Context(), composePath, service, cmd.Flag("follow").Value.String() == "true")
    },
}
```

Flags: `--follow` / `-f` (stream), `--lines` / `-n` (tail count, default 100).

---

### Extension: `volra quickstart` Command

**New command + embedded templates.**

```go
// cmd/quickstart.go
var quickstartCmd = &cobra.Command{
    Use:   "quickstart [template] [directory]",
    Short: "Create a new agent project from a template",
    RunE:  runQuickstart,
}
```

**Embedded Templates:**

```
internal/templates/
  basic/
    main.py
    requirements.txt
    Agentfile
    README.md
  rag/
    main.py
    requirements.txt
    Agentfile
    README.md
  conversational/
    main.py
    requirements.txt
    Agentfile
    README.md
```

Templates embedded via `//go:embed templates/*` in `internal/templates/templates.go`.

Converted from existing E2E test agents (A1 echo-agent → basic, A4 rag-kb → rag, A5 conv-agent → conversational) with user-facing README and documentation.

**Command Flow:**
1. List templates if no argument: `volra quickstart` shows available templates
2. Copy template to target directory: `volra quickstart rag my-rag-agent`
3. Replace placeholder names with project name
4. Print next steps: "cd my-rag-agent && volra deploy"

---

### v0.2 Field Count Summary

After v0.2: ~16 Agentfile fields (existing 15 from v2.1 + `observability` struct with 2 sub-fields)

### v0.2 New Package Summary

| Package | Type | Purpose |
|---------|------|---------|
| `internal/mcp/` | NEW | MCP server implementation |
| `internal/templates/` | NEW | Embedded quickstart templates |
| `cmd/mcp.go` | NEW | MCP command entry point |
| `cmd/logs.go` | NEW | Logs command entry point |
| `cmd/quickstart.go` | NEW | Quickstart command entry point |
| `volra-observe` (Python, separate repo) | NEW | Framework-agnostic LLM instrumentation for Level 2 metrics |

### v0.2 Backward Compatibility

All v0.1 Agentfiles (including v2.1 schema) MUST parse without errors with v0.2 CLI. New `observability` field defaults to nil (Level 1 behavior). New commands (`mcp`, `logs`, `quickstart`) do not affect existing commands.
