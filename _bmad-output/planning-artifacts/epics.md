---
stepsCompleted: [step-01-validate-prerequisites, step-02-design-epics, step-03-create-stories, step-04-final-validation]
inputDocuments:
  - '_bmad-output/planning-artifacts/prd.md'
  - '_bmad-output/planning-artifacts/architecture.md'
executionMode: 'GENERATE'
project_name: 'MegaCenter'
---

# MegaCenter - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for MegaCenter, decomposing the requirements from the PRD and Architecture into implementable stories.

## Requirements Inventory

### Functional Requirements

**Environment Diagnosis (FR1-FR3)**
- FR1: Developer can run a pre-flight check that validates all prerequisites: Docker installed, Docker running, Docker Compose V2 available, required ports free, Python >= 3.10 present, sufficient disk space, and MegaCenter version reported
- FR2: Developer can see a pass/fail result for each individual check with a specific fix suggestion for each failure
- FR3: Developer can determine from the command exit code whether all checks passed (0) or any failed (1)

**Project Detection & Configuration (FR4-FR15)**
- FR4: Developer can point MegaCenter at a Python project directory and receive an auto-generated Agentfile with a summary of detected values and instructions to customize
- FR5: System can detect the agent framework used (generic or LangGraph) by scanning dependency files and Python imports
- FR6: System can detect the application entry point by scanning for common filenames (main.py, app.py, server.py) in priority order
- FR7: System can detect the application port by scanning entry point code for server startup patterns
- FR8: System can detect health check endpoints by scanning route decorators
- FR9: System can detect environment variable references in agent code (os.environ, os.getenv patterns) to populate .env.example
- FR10: When detection is partial or ambiguous, system uses safe defaults and emits a warning with override instructions referencing the specific Agentfile field
- FR11: Developer can override any detected value by editing the generated Agentfile
- FR12: System generates a .env.example file listing detected environment variable names
- FR13: System adds .megacenter/ and .env to the project's .gitignore
- FR14: Generated Agentfile includes a schema version field for forward compatibility
- FR15: When Agentfile already exists, system exits with error and suggests --force flag to overwrite

**Stack Generation & Deployment (FR16-FR26)**
- FR16: Developer can generate a complete deployment stack (Dockerfile, docker-compose.yml, Prometheus config, Grafana dashboards) from an Agentfile
- FR17 [SHOULD]: Developer can generate the deployment stack without executing it (dry-run mode)
- FR18: System generates a Dockerfile using auto-detection results, or uses an existing project Dockerfile when configured as custom mode
- FR19: System generates a docker-compose.yml that orchestrates 3 containers: agent, Prometheus, and Grafana
- FR20: System generates a Prometheus configuration that scrapes the agent's health endpoint directly, with an alert rule for agent-down detection
- FR21: System generates Grafana dashboard definitions and provisioning configuration, including setting the Overview dashboard as the default landing page
- FR22: System executes docker compose to start all services after artifact generation
- FR23: System verifies agent health by polling the configured health endpoint with a default timeout
- FR24: Developer can see a structured deploy output listing: generated artifacts, service status with ports, summary URLs (agent + dashboard), and the exact command to stop services
- FR25: Developer can update a deployed agent by re-running deploy, which rebuilds the agent container without losing monitoring history
- FR26: System detects and reports common deployment failures with actionable error messages (Docker not running, port in use, OOM killed, health check timeout, build failure)

**Health Monitoring & Status (FR27-FR30)**
- FR27 [SHOULD]: Developer can check the current health status of a deployed agent and its supporting services
- FR28 [SHOULD]: Developer can see agent health state (healthy/unhealthy), uptime, and port assignment
- FR29 [SHOULD]: Developer can see the running state and port of each supporting service (Prometheus, Grafana)
- FR30: System detects when Docker daemon has restarted and reports that agents need redeployment

**Observability — Probe Metrics (FR31-FR35)**
- FR31: System provides health probe metrics by configuring Prometheus to scrape the agent's health endpoint directly
- FR32: Developer can view an Overview dashboard showing: agent status badge, uptime, and current probe latency
- FR33: Developer can view a Detail dashboard showing: probe latency over time, health status timeline, and uptime percentage (24h/7d)
- FR34: Overview dashboard displays a visual alert indicator (red status) when health check is failing
- FR35: If the agent exposes Prometheus-format metrics on /metrics, system automatically collects them alongside probe metrics

**Observability — Dashboard UX (FR36-FR37)**
- FR36: Developer can open Grafana without authentication and land directly on the Overview dashboard
- FR37: Dashboards clearly label all metrics as probe-based to distinguish from real user traffic metrics

**Installation & Distribution (FR38-FR41)**
- FR38: Developer can install MegaCenter via a single shell command that auto-detects OS and architecture
- FR39 [SHOULD]: Installation script verifies binary integrity via SHA256 checksum before placing the binary
- FR40: When the install location requires elevated permissions, system suggests an alternative user-local path with PATH configuration instructions
- FR41: Developer can check the installed MegaCenter version

**Classification: 36 MUST + 5 SHOULD (FR17, FR27, FR28, FR29, FR39)**

### NonFunctional Requirements

**Performance (NFR1-NFR6)**
- NFR1: CLI commands (doctor, init, status) complete in < 5 seconds
- NFR2: Deploy first run (cold) completes in < 5 minutes for typical Python agent
- NFR3: Deploy subsequent runs (warm) complete in < 3 minutes; no-change redeploy < 30 seconds
- NFR4: Grafana dashboards load and display within 5 seconds
- NFR5: Prometheus scrapes health endpoint every 15 seconds with 10s timeout
- NFR6: Total time-to-first-deploy < 15 minutes

**Reliability (NFR7-NFR11)**
- NFR7: CLI commands complete without unexpected errors > 95% of the time
- NFR8: Same Agentfile + same code produces functionally equivalent deployments across macOS ARM64, Ubuntu 24.04, Debian 12
- NFR9: Prometheus data volumes survive agent container rebuilds
- NFR10: Generated artifacts are valid and functional without manual modification
- NFR11: Health check probe detects agent-down state within 1 minute

**Compatibility (NFR12-NFR15)**
- NFR12: Binary runs on macOS ARM64, Ubuntu 24.04 AMD64, Debian 12 AMD64 without additional dependencies
- NFR13: Generated Docker Compose files compatible with Compose V2
- NFR14: Generated Dockerfiles produce functional images on both AMD64 and ARM64
- NFR15: CLI output renders correctly with and without emoji/Unicode support (NO_COLOR, TERM=dumb, 60 cols min)

**Security (NFR16-NFR20)**
- NFR16: Environment variable values never appear in generated artifacts — only names referenced
- NFR17: .env file automatically added to .gitignore
- NFR18 [SHOULD]: Install script verifies binary integrity via SHA256
- NFR19: Grafana uses anonymous access (no credentials) — acceptable for localhost-only v0.1
- NFR20: No telemetry, no phone-home, no data collection

**Resource Efficiency (NFR21)**
- NFR21: Monitoring stack (Prometheus + Grafana) uses < 500MB RAM at rest

**Developer Experience (NFR22-NFR23)**
- NFR22: Every error message follows: what happened + actionable fix instruction
- NFR23: Every warning message follows: what happened + what was assumed + how to override

**Classification: 22 MUST + 1 SHOULD (NFR18)**

### Additional Requirements

**From Architecture — Project Scaffolding:**
- Go project with `go mod init github.com/antonioromero/megacenter`
- 3 external dependencies: cobra, yaml.v3, testify (dev)
- 9 internal packages: cmd/megacenter/, internal/{agentfile, output, docker, doctor, setup, deploy, status, testutil}
- Makefile with targets: build, test, test-integration, lint, build-all, checksums, clean
- CI/CD: ci.yml (go build + test + lint + shellcheck) + release.yml (cross-compile + GitHub Release)

**From Architecture — Shared Infrastructure (leaf packages first):**
- Presenter interface with 4 methods (Progress, Result, Error, Warn) + DetectMode() + 3 modes
- UserError{Code, What, Fix} and UserWarning{What, Assumed, Override} types + error catalog (E101-E502)
- Agentfile struct with Parse-Don't-Validate: custom types (Framework, DockerfileMode) with UnmarshalYAML
- Two-level validation: Parse() syntactic + Validate() semantic → Load() combined
- DockerRunner interface: Run(ctx, args...) (string, error) + ExecRunner via os/exec
- Test utilities: AssertGolden() with UPDATE_GOLDEN=1, MockPresenter, MockDockerRunner

**From Architecture — Template & Static Assets:**
- 3 templates (go:embed + text/template): Dockerfile.tmpl, docker-compose.yml.tmpl, prometheus.yml.tmpl
- 5 static files (go:embed + copy): alert_rules.yml, overview.json, detail.json, dashboards.yml, datasource.yml
- Templates and static dirs live inside internal/deploy/ (sole consumer)
- TemplateContext struct: embedded Agentfile + PythonVersion + EntryPoint + HasRequirements

**From Architecture — Implementation Constraints:**
- Cobra wiring pattern: RunE = 3-4 lines, zero business logic in cmd/
- Exact Run() signatures for all 4 commands
- 8 enforcement guidelines (no fmt.Print outside output/, no os.Exit outside cmd/, etc.)
- Exit codes: {0, 1, 130} only
- Fix-and-re-run philosophy: no cleanup on failure, no rollback
- Generated output in .megacenter/ directory, fully regenerated each deploy
- docker compose -f .megacenter/docker-compose.yml -p {name} up -d

**From Architecture — Build & Distribution:**
- Cross-compile for darwin/arm64 and linux/amd64 (2 combos = 3 PRD targets)
- install.sh: curl | sh installer with OS/arch detection
- CONTRIBUTING.md + GitHub Issue templates (bug_report.md, feature_request.md)

**From Architecture — Recommended Implementation Order:**
1. Project scaffolding (go mod init, directory structure, Makefile)
2. internal/output/ (Presenter, modes, error types — leaf package)
3. internal/agentfile/ (Parse, Validate, Load — leaf package)
4. internal/docker/ (DockerRunner interface + ExecRunner — leaf package)
5. internal/testutil/ (MockPresenter, MockDockerRunner, AssertGolden)
6. internal/doctor/ (simplest command)
7. internal/setup/ (second command, creates Agentfile)
8. internal/deploy/ (largest package, highest risk)
9. internal/status/ (read-only, depends on deploy)
10. cmd/megacenter/ (Cobra wiring, assembled last)

### FR Coverage Map

_Autonomous Party Mode — 3 rounds, 9 adjustments. Key decisions: FR41 moved to Epic 1, FR36/FR37 as acceptance criteria not stories, NFRs cross-cutting as ACs, Epic 3 not split._

| FR | Epic | Description |
|----|------|-------------|
| FR1 | Epic 1 | Pre-flight check validates all prerequisites |
| FR2 | Epic 1 | Pass/fail per check with fix suggestion |
| FR3 | Epic 1 | Exit code 0/1 for scripting |
| FR41 | Epic 1 | Version check (--version flag) |
| FR4 | Epic 2 | Point at Python project → auto-generate Agentfile |
| FR5 | Epic 2 | Detect framework (generic/LangGraph) |
| FR6 | Epic 2 | Detect entry point (main.py/app.py/server.py) |
| FR7 | Epic 2 | Detect application port from code |
| FR8 | Epic 2 | Detect health check endpoints |
| FR9 | Epic 2 | Detect environment variable references |
| FR10 | Epic 2 | Safe defaults + warning with override instructions |
| FR11 | Epic 2 | Override via Agentfile editing |
| FR12 | Epic 2 | Generate .env.example |
| FR13 | Epic 2 | Add .megacenter/ and .env to .gitignore |
| FR14 | Epic 2 | Agentfile schema version field |
| FR15 | Epic 2 | Agentfile exists → error + --force suggestion |
| FR16 | Epic 3 | Generate complete deployment stack from Agentfile |
| FR17 | Deferred | [SHOULD] Dry-run mode — v0.2 |
| FR18 | Epic 3 | Generate Dockerfile (auto/custom mode) |
| FR19 | Epic 3 | Generate docker-compose.yml with 3 containers |
| FR20 | Epic 3 | Generate Prometheus config + alert rule |
| FR21 | Epic 3 | Generate Grafana dashboards + provisioning |
| FR22 | Epic 3 | Execute docker compose up |
| FR23 | Epic 3 | Verify agent health post-deploy |
| FR24 | Epic 3 | Structured deploy output with URLs and stop command |
| FR25 | Epic 3 | Re-deploy rebuilds agent, preserves monitoring |
| FR26 | Epic 3 | Deploy error detection with actionable messages |
| FR31 | Epic 3 | Probe metrics via Prometheus direct scrape |
| FR32 | Epic 3 | Overview dashboard: status, uptime, latency |
| FR33 | Epic 3 | Detail dashboard: latency timeline, health timeline, uptime % |
| FR34 | Epic 3 | Visual alert indicator on Overview dashboard |
| FR35 | Epic 3 | Automatic bonus metrics collection from /metrics |
| FR36 | Epic 3 | Grafana anonymous access + default dashboard (AC of Grafana story) |
| FR37 | Epic 3 | Honest probe-based metric labeling (AC of dashboard story) |
| FR27 | Epic 4 | [SHOULD] Check deployed agent health status |
| FR28 | Epic 4 | [SHOULD] Show agent health state, uptime, port |
| FR29 | Epic 4 | [SHOULD] Show service states and ports |
| FR30 | Epic 4 | Detect Docker daemon restart |
| FR38 | Epic 5 | Single shell command installation |
| FR39 | Epic 5 | [SHOULD] SHA256 checksum verification |
| FR40 | Epic 5 | Permission-aware install with PATH instructions |

**Coverage: 40/41 FRs mapped. 1 deferred (FR17). FR36/FR37 as acceptance criteria within Epic 3 stories.**

## Epic List

### Epic 1: Foundation & Environment Readiness
Developer can verify their system is ready for MegaCenter. Includes project scaffolding, shared infrastructure (Presenter, DockerRunner, error types, test utilities), and a fully functional `megacenter doctor` command.
**FRs covered:** FR1, FR2, FR3, FR41
**NFRs addressed:** NFR1, NFR7, NFR12, NFR15, NFR22, NFR23
**Packages:** project scaffolding, cmd/megacenter (root + doctor), internal/output, internal/docker, internal/testutil, internal/doctor

### Epic 2: Project Configuration
Developer can set up their agent project with intelligent auto-detection. `megacenter init` scans a Python project, detects framework/port/health/env, and generates the Agentfile + .env.example + .gitignore updates.
**FRs covered:** FR4, FR5, FR6, FR7, FR8, FR9, FR10, FR11, FR12, FR13, FR14, FR15
**NFRs addressed:** NFR1, NFR10, NFR16, NFR17
**Packages:** internal/agentfile, internal/setup, cmd/megacenter/init.go

### Epic 3: Agent Deployment with Monitoring
Developer can deploy their agent with full observability stack in one command. `megacenter deploy` reads the Agentfile, generates 8 artifacts, executes docker compose up, verifies health, and delivers functional Prometheus + Grafana dashboards.
**FRs covered:** FR16, FR18, FR19, FR20, FR21, FR22, FR23, FR24, FR25, FR26, FR31, FR32, FR33, FR34, FR35, FR36 (AC), FR37 (AC)
**FR17 [SHOULD]:** Deferred to v0.2
**NFRs addressed:** NFR2-5, NFR8-11, NFR13-14, NFR16, NFR19-21
**Packages:** internal/deploy (all files + templates/ + static/), cmd/megacenter/deploy.go

### Epic 4: Health Status Monitoring
Developer can check current health of deployed agents from the CLI. `megacenter status` queries Docker for container states, verifies health endpoint, and reports agent + service status.
**FRs covered:** FR27 [SHOULD], FR28 [SHOULD], FR29 [SHOULD], FR30
**NFRs addressed:** NFR1
**Packages:** internal/status, cmd/megacenter/status.go

### Epic 5: Distribution & Installation
Developer can install MegaCenter with a single shell command. Cross-compilation, GitHub Release with binaries + SHA256 checksums, install.sh with OS/arch auto-detection, and open-source project artifacts.
**FRs covered:** FR38, FR39 [SHOULD], FR40
**NFRs addressed:** NFR6, NFR12, NFR18 [SHOULD]
**Packages:** install.sh, .github/workflows/release.yml, CONTRIBUTING.md, .github/ISSUE_TEMPLATE/

---

## Stories

### Epic 1: Foundation & Environment Readiness

#### Story 1.1: Project Scaffolding & Build System

**As a** developer contributing to MegaCenter,
**I want** the Go project initialized with module, directory structure, dependencies, and Makefile,
**So that** all subsequent stories have a compilable, testable foundation to build on.

**Acceptance Criteria:**

```gherkin
Given the repository is freshly cloned
When I run `go mod tidy`
Then go.mod exists with module `github.com/antonioromero/megacenter`
  And go.sum contains entries for cobra, yaml.v3, and testify

Given the repository is freshly cloned
When I inspect the directory layout
Then the following directories exist:
  cmd/megacenter/
  internal/agentfile/
  internal/output/
  internal/docker/
  internal/doctor/
  internal/setup/
  internal/deploy/
  internal/status/
  internal/testutil/

Given the Makefile exists
When I run `make build`
Then a binary is produced at `bin/megacenter`
  And it exits 0 when invoked with `--version`

Given the Makefile exists
When I run `make test`
Then `go test ./...` runs with `-race` flag
  And all tests pass (even if there are zero tests initially)

Given the Makefile exists
When I run `make lint`
Then `golangci-lint run` executes without errors

Given the Makefile exists
When I run `make build-all`
Then binaries are produced for darwin/arm64 and linux/amd64

Given the Makefile exists
When I run `make checksums`
Then a SHA256SUMS file is generated covering all binaries in bin/

Given the Makefile exists
When I run `make clean`
Then the bin/ directory is removed

Given go.mod exists
When I run `go get` for cobra, yaml.v3, and testify
Then all 3 dependencies resolve and are recorded in go.mod and go.sum
```

**Technical Notes:**
- `go mod init github.com/antonioromero/megacenter`
- 3 external deps only: `github.com/spf13/cobra`, `gopkg.in/yaml.v3`, `github.com/stretchr/testify` (dev)
- Makefile targets: build, test, test-integration, lint, build-all, checksums, clean
- `cmd/megacenter/main.go` can be a minimal stub that imports root command

---

#### Story 1.2: Output System (Presenter, Error Types, Error Catalog)

**As a** MegaCenter developer,
**I want** a Presenter interface with mode detection and structured error/warning types,
**So that** all commands produce consistent, actionable output following NFR22/NFR23.

**Acceptance Criteria:**

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

**Technical Notes:**
- `internal/output/presenter.go`: interface + ColorPresenter/NoColorPresenter/PlainPresenter
- `internal/output/types.go`: UserError (implements `error`), UserWarning
- `internal/output/catalog.go`: error code constants
- `internal/output/mode.go`: DetectMode() function
- No `fmt.Print` outside this package (Enforcement Guideline #1)
- `Presenter.Error(err error)` uses `errors.As(&ue)` to check for `*UserError`

---

#### Story 1.3: Docker Runner & Test Infrastructure

**As a** MegaCenter developer,
**I want** a DockerRunner interface for all Docker interactions and shared test utilities,
**So that** commands can be tested without a real Docker daemon.

**Acceptance Criteria:**

```gherkin
Given the docker package exists
When I inspect DockerRunner
Then it defines: Run(ctx context.Context, args ...string) (string, error)

Given ExecRunner implements DockerRunner
When Run is called with args ["compose", "version"]
Then it executes `docker compose version` via os/exec
  And returns stdout as string, or error on non-zero exit

Given the testutil package exists
When I inspect MockDockerRunner
Then it implements DockerRunner
  And records all calls (args) for assertion
  And returns configurable (string, error) responses

Given the testutil package exists
When I inspect MockPresenter
Then it implements output.Presenter
  And records all calls to Progress, Result, Error, Warn

Given the testutil package exists
When I call AssertGolden(t, "testname", got)
Then it compares `got` against testdata/testname.golden
  And fails the test if they differ
  And when UPDATE_GOLDEN=1 is set, it overwrites the golden file with `got`
```

**Technical Notes:**
- `internal/docker/runner.go`: DockerRunner interface + ExecRunner struct
- `internal/testutil/mock_presenter.go`: MockPresenter
- `internal/testutil/mock_docker.go`: MockDockerRunner
- `internal/testutil/golden.go`: AssertGolden helper
- No `os/exec` outside docker package (Enforcement Guideline #4)

---

#### Story 1.4: Doctor Command

**As a** developer setting up MegaCenter,
**I want** to run `megacenter doctor` to validate all prerequisites,
**So that** I know my system is ready before attempting to deploy an agent.

**Acceptance Criteria:**

```gherkin
Given Docker is installed and running
When I run `megacenter doctor`
Then it checks: Docker installed, Docker running, Compose V2 available, Python >= 3.10 present, sufficient disk space (1GB)
  And each check shows pass/fail with a specific fix suggestion on failure (FR2)
  And MegaCenter version is reported in the output (FR41)
  And exit code is 0 (FR3)

Given Docker is not installed
When I run `megacenter doctor`
Then the Docker check fails with error E101 and fix: "Install Docker Desktop from https://docker.com"
  And exit code is 1

Given Docker is installed but not running
When I run `megacenter doctor`
Then the Docker-running check fails with error E102 and fix: "Start Docker Desktop or run: sudo systemctl start docker"

Given Compose V2 is not available
When I run `megacenter doctor`
Then the check fails with error E103 and fix referencing Docker Compose V2 installation

Given Python < 3.10 or not found
When I run `megacenter doctor`
Then the check fails with error E104 and fix referencing Python installation

Given disk space < 1GB available
When I run `megacenter doctor`
Then the check fails with error E105 and fix: "Free up disk space. MegaCenter needs at least 1GB."

Given ports 9090 and 3000 are in use
When I run `megacenter doctor`
Then it warns about port conflicts (warning, not failure)

Given the doctor command runs
When all checks pass
Then the command completes in < 5 seconds (NFR1)
```

**Technical Notes:**
- `internal/doctor/doctor.go`: `Run(ctx, p Presenter, dr DockerRunner) error`
- Checks call DockerRunner for Docker-related checks, os/exec for Python (via DockerRunner pattern or direct)
- Each check is independent — run all, report all, fail if any failed
- Uses Presenter for all output
- Unit tests use MockDockerRunner + MockPresenter

---

#### Story 1.5: CLI Entry Point & Root Command

**As a** developer,
**I want** a `megacenter` binary with Cobra root command and `doctor` subcommand wired,
**So that** I can run `megacenter doctor` and `megacenter --version` from my terminal.

**Acceptance Criteria:**

```gherkin
Given the binary is built
When I run `megacenter --version`
Then it prints the version string (set via ldflags at build time)
  And exit code is 0

Given the binary is built
When I run `megacenter doctor`
Then it delegates to internal/doctor.Run()
  And the Cobra RunE function is 3-4 lines: create deps, call Run(), return error

Given the binary is built
When I run `megacenter` with no subcommand
Then it shows help text listing available commands

Given the binary is built
When I run `megacenter nonexistent`
Then Cobra prints an error and suggests valid commands

Given the root command is wired
When I inspect cmd/megacenter/root.go
Then it creates the Presenter (via DetectMode) and DockerRunner
  And passes them to subcommand RunE functions
  And contains zero business logic (Enforcement Guideline #2)

Given any command returns a non-nil error
When the error reaches main()
Then main() calls os.Exit(1)
  And os.Exit is only called in main() (Enforcement Guideline #3)
```

**Technical Notes:**
- `cmd/megacenter/main.go`: main() calls root.Execute(), os.Exit on error
- `cmd/megacenter/root.go`: Cobra root command, version via ldflags
- `cmd/megacenter/doctor.go`: Cobra doctor subcommand, RunE = create deps + call doctor.Run()
- Version set via: `go build -ldflags "-X main.version=..."'`

---

### Epic 2: Project Configuration

#### Story 2.1: Agentfile Schema & Validation

**As a** MegaCenter developer,
**I want** a typed Agentfile struct with parse-don't-validate semantics,
**So that** invalid configurations are caught at load time with actionable error messages.

**Acceptance Criteria:**

```gherkin
Given a valid Agentfile YAML with all 7 fields
When I call agentfile.Load(path)
Then it returns an *Agentfile with all fields populated
  And Framework is a typed enum (Generic or LangGraph)
  And DockerfileMode is a typed enum (Generate or Custom)

Given a YAML with `framework: langgraph`
When agentfile.Parse() processes it
Then Framework.UnmarshalYAML succeeds and Framework == LangGraph

Given a YAML with `framework: invalid`
When agentfile.Parse() processes it
Then UnmarshalYAML returns an error with code E201

Given an Agentfile missing required field `name`
When agentfile.Validate() runs
Then it returns UserError{Code: "E202", What: "...", Fix: "Add 'name' to your Agentfile"}

Given an Agentfile with name containing spaces
When agentfile.Validate() runs
Then it returns an error because name must be a valid DNS label

Given an Agentfile with schema_version field
When agentfile.Load() processes it
Then the version is preserved for forward compatibility (FR14)

Given an Agentfile with all valid fields
When agentfile.Validate() runs
Then it returns nil (no error)

Given a two-level validation design
When agentfile.Load(path) is called
Then it calls Parse() (syntactic) then Validate() (semantic)
  And returns the first error encountered
```

**Technical Notes:**
- `internal/agentfile/agentfile.go`: Agentfile struct, 7 fields (schema_version, name, framework, entry_point, port, health_path, dockerfile_mode)
- `internal/agentfile/types.go`: Framework, DockerfileMode custom types with UnmarshalYAML
- `internal/agentfile/parse.go`: Parse() via yaml.v3
- `internal/agentfile/validate.go`: Validate() semantic rules
- `internal/agentfile/load.go`: Load() = Parse() + Validate()
- DNS label validation for name field
- Test with golden files for error messages

---

#### Story 2.2: Project Scanning & Detection Engine

**As a** developer running `megacenter init`,
**I want** the system to auto-detect my agent's framework, entry point, port, health endpoint, and environment variables,
**So that** I get a pre-populated Agentfile with minimal manual editing.

**Acceptance Criteria:**

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
Then it returns 8000 as default and emits a warning with override instructions (FR10)

Given entry point code with `@app.get("/health")`
When the health endpoint detector runs
Then it returns /health (FR8)

Given agent code with `os.environ["OPENAI_API_KEY"]` and `os.getenv("MODEL_NAME")`
When the env var detector runs
Then it returns ["OPENAI_API_KEY", "MODEL_NAME"] (FR9)

Given detection is partial (e.g., port not found)
When the scan completes
Then safe defaults are used and a UserWarning is emitted with the specific Agentfile field to override (FR10, NFR23)

Given the testdata/ directory
When I inspect fixture projects
Then there are fixtures for: generic project, LangGraph project, project with custom Dockerfile, project with pyproject.toml dependencies, minimal project with no detectable values
```

**Technical Notes:**
- `internal/setup/detect.go`: ScanProject() returns detection results
- `internal/setup/detect_framework.go`: scans requirements.txt, pyproject.toml, setup.py imports
- `internal/setup/detect_entry.go`: scans for common filenames
- `internal/setup/detect_port.go`: regex patterns on entry point code
- `internal/setup/detect_health.go`: scans route decorators
- `internal/setup/detect_env.go`: scans for os.environ/os.getenv patterns
- Fixture projects in `internal/setup/testdata/` for each detection scenario

---

#### Story 2.3: Init Command Pipeline

**As a** developer,
**I want** to run `megacenter init <path>` to generate an Agentfile, .env.example, and .gitignore updates,
**So that** my project is configured for MegaCenter deployment.

**Acceptance Criteria:**

```gherkin
Given a Python project directory with no existing Agentfile
When I run `megacenter init ./my-agent`
Then it generates Agentfile in ./my-agent/ with detected values (FR4)
  And generates .env.example listing detected env var names (FR12)
  And adds .megacenter/ and .env to .gitignore (FR13)
  And prints a summary of detected values and instructions to customize (FR4)

Given a project directory with an existing Agentfile
When I run `megacenter init ./my-agent`
Then it exits with error E206 and suggests `--force` flag (FR15)

Given a project directory with an existing Agentfile
When I run `megacenter init --force ./my-agent`
Then it overwrites the existing Agentfile (FR15)

Given .gitignore already contains .megacenter/
When init runs
Then it does not duplicate the entry (FR13)

Given no .gitignore exists
When init runs
Then it creates .gitignore with .megacenter/ and .env entries (FR13)

Given the generated Agentfile
When I inspect it
Then it includes schema_version field (FR14)
  And all fields have comments explaining their purpose (FR11)

Given the generated .env.example
When I inspect it
Then it lists env var names without values (NFR16)
  And each line is `VAR_NAME=` with no value

Given init completes
When I check the command duration
Then it completes in < 5 seconds (NFR1)
```

**Technical Notes:**
- `internal/setup/setup.go`: `Run(ctx, p Presenter, projectPath string, force bool) error`
- Calls ScanProject(), generates Agentfile YAML, writes .env.example, updates .gitignore
- `cmd/megacenter/init.go`: Cobra subcommand, `--force` flag, RunE delegates to setup.Run()
- Environment variable values NEVER written to any generated file (NFR16)

---

### Epic 3: Agent Deployment with Monitoring

#### Story 3.1: Dockerfile Generation

**As a** MegaCenter developer,
**I want** the deploy command to generate a Dockerfile from a template when mode is "generate",
**So that** the agent can be containerized without manual Docker knowledge.

**Acceptance Criteria:**

```gherkin
Given an Agentfile with dockerfile_mode: generate
When the Dockerfile generator runs
Then it produces a Dockerfile using the Dockerfile.tmpl template
  And the base image is python:3.11-slim
  And it copies requirements.txt and runs pip install
  And it copies all source code
  And the CMD uses the configured entry_point
  And the EXPOSE uses the configured port

Given an Agentfile with dockerfile_mode: custom
When the Dockerfile generator runs
Then it skips generation and uses the existing Dockerfile in the project (FR18)

Given the template rendering
When TemplateContext is populated
Then it includes: embedded Agentfile fields, PythonVersion, EntryPoint, HasRequirements

Given a project with both requirements.txt and pyproject.toml
When the Dockerfile generator runs
Then it uses requirements.txt for pip install

Given the deploy package
When I inspect the file structure
Then templates/Dockerfile.tmpl exists within internal/deploy/
  And embed.go declares the go:embed directive for templates/ and static/
  And constants.go defines shared constants (JobHealth, JobMetrics, DatasourceName, NetworkName)
```

**Technical Notes:**
- `internal/deploy/templates/Dockerfile.tmpl`: go:embed template
- `internal/deploy/generate_dockerfile.go`: renders template with TemplateContext
- `internal/deploy/embed.go`: `//go:embed templates/* static/*`
- `internal/deploy/constants.go`: shared constants
- `internal/deploy/context.go`: TemplateContext struct
- Output written to `.megacenter/Dockerfile`

---

#### Story 3.2: Docker Compose Generation

**As a** MegaCenter developer,
**I want** the deploy command to generate a docker-compose.yml orchestrating agent, Prometheus, and Grafana,
**So that** the full stack is defined declaratively.

**Acceptance Criteria:**

```gherkin
Given a valid Agentfile
When the compose generator runs
Then it produces docker-compose.yml with exactly 3 services: agent, prometheus, grafana (FR19)

Given the generated agent service
When I inspect it
Then it builds from the generated Dockerfile (or project Dockerfile in custom mode)
  And exposes the configured port
  And loads environment from .env file
  And has a healthcheck using the configured health_path

Given the generated prometheus service
When I inspect it
Then it uses image prom/prometheus:v3.1.0
  And mounts prometheus.yml to /etc/prometheus/prometheus.yml:ro
  And mounts alert_rules.yml to /etc/prometheus/alert_rules.yml:ro
  And uses `--config.file=/etc/prometheus/prometheus.yml` command flag
  And has a named volume prometheus-data for persistence (NFR9)
  And exposes port 9090

Given the generated grafana service
When I inspect it
Then it uses image grafana/grafana-oss:12.4.0
  And mounts provisioning configs (datasource.yml, dashboards.yml) and dashboard JSON files
  And sets GF_AUTH_ANONYMOUS_ENABLED=true and GF_AUTH_ANONYMOUS_ORG_ROLE=Viewer (NFR19, FR36)
  And sets GF_DASHBOARDS_DEFAULT_HOME_DASHBOARD_UID to the Overview dashboard UID (FR36)
  And exposes port 3000

Given the generated compose file
When I inspect the network configuration
Then all 3 services share a common network named {name}-net

Given the generated compose file
When I inspect the project name
Then it uses the Agentfile name field as the compose project name
```

**Technical Notes:**
- `internal/deploy/templates/docker-compose.yml.tmpl`: go:embed template
- `internal/deploy/generate_compose.go`: renders template with TemplateContext
- Output written to `.megacenter/docker-compose.yml`
- FR36 (anonymous Grafana + default dashboard) implemented as ACs here

---

#### Story 3.3: Prometheus Configuration & Alert Rules

**As a** MegaCenter developer,
**I want** the deploy command to generate Prometheus config that scrapes the agent's health endpoint,
**So that** probe metrics (up, latency) are collected automatically.

**Acceptance Criteria:**

```gherkin
Given a valid Agentfile with health_path: /health
When the Prometheus config generator runs
Then prometheus.yml is generated with a scrape job targeting http://agent:{port}{health_path}
  And scrape_interval is 15s (NFR5)
  And scrape_timeout is 10s (NFR5)
  And rule_files references /etc/prometheus/alert_rules.yml

Given the generated alert_rules.yml (static file)
When I inspect it
Then it contains an alert rule for agent-down detection (FR20)
  And the alert fires when `up == 0` for the health job for > 1 minute (NFR11)

Given the agent exposes /metrics in Prometheus format
When Prometheus scrapes
Then a second scrape job automatically collects those metrics (FR35)

Given the Prometheus config
When the Overview dashboard queries `up{job="health"}`
Then it can display the agent status badge (FR32, FR34)

Given the deploy package
When I inspect static/alert_rules.yml
Then it is a static embedded file (not a template)

Given the deploy package test suite
When tests run
Then consistency_test.go verifies that all static/ and templates/ files referenced in code actually exist as embedded assets
```

**Technical Notes:**
- `internal/deploy/templates/prometheus.yml.tmpl`: go:embed template (needs agent host/port/path)
- `internal/deploy/static/alert_rules.yml`: go:embed static file
- `internal/deploy/generate_prometheus.go`: renders prometheus.yml template
- `internal/deploy/consistency_test.go`: verifies embedded assets match code references
- Two scrape jobs: "health" (health_path) and "metrics" (/metrics, optional via FR35)
- Output: `.megacenter/prometheus.yml` and `.megacenter/alert_rules.yml`

---

#### Story 3.4: Grafana Dashboards & Provisioning

**As a** developer deploying with MegaCenter,
**I want** pre-built Grafana dashboards showing agent health, uptime, and latency,
**So that** I get instant observability without manual dashboard configuration.

**Acceptance Criteria:**

```gherkin
Given the deploy command generates Grafana assets
When I inspect the output
Then the following files are generated in .megacenter/:
  datasource.yml, dashboards.yml, overview.json, detail.json (FR21)

Given the Overview dashboard (overview.json)
When loaded in Grafana
Then it shows: agent status badge (healthy/unhealthy), uptime, current probe latency (FR32)
  And it shows a visual alert indicator (red status) when health check is failing (FR34)
  And all metrics are clearly labeled as probe-based (FR37)

Given the Detail dashboard (detail.json)
When loaded in Grafana
Then it shows: probe latency over time, health status timeline, uptime percentage for 24h and 7d (FR33)
  And all metrics are clearly labeled as probe-based (FR37)

Given datasource.yml
When Grafana loads provisioning
Then it auto-configures Prometheus at http://prometheus:9090 as default datasource

Given dashboards.yml
When Grafana loads provisioning
Then it discovers dashboard JSON files from the provisioned path

Given the Overview dashboard
When Grafana starts
Then it is set as the default home dashboard (FR36)

Given Grafana dashboards
When they load in the browser
Then they display within 5 seconds (NFR4)
```

**Technical Notes:**
- `internal/deploy/static/overview.json`: Overview dashboard definition (go:embed)
- `internal/deploy/static/detail.json`: Detail dashboard definition (go:embed)
- `internal/deploy/static/datasource.yml`: Prometheus datasource provisioning (go:embed)
- `internal/deploy/static/dashboards.yml`: Dashboard provisioning config (go:embed)
- All static files — no templating needed
- FR37 (honest labeling) implemented as panel titles/descriptions within JSON

---

#### Story 3.5: Deploy Orchestration (docker compose up)

**As a** developer,
**I want** the deploy command to execute `docker compose up` after generating artifacts,
**So that** all services start with a single command.

**Acceptance Criteria:**

```gherkin
Given all artifacts are generated in .megacenter/
When the deploy orchestrator runs
Then it executes: docker compose -f .megacenter/docker-compose.yml -p {name} up -d --build (FR22)
  And the agent container is rebuilt (FR25 for re-deploys)
  And monitoring containers are NOT rebuilt if unchanged (FR25 — preserves history)

Given a re-deploy scenario
When the deploy orchestrator runs again
Then the agent container is rebuilt with new code
  And Prometheus data volume is preserved (NFR9)
  And Grafana retains dashboard state

Given Docker is not running
When the deploy orchestrator attempts docker compose
Then it fails with UserError E301: "Docker is not running" + fix (FR26)

Given the configured port is already in use
When docker compose up fails
Then it reports UserError E302: "Port {port} is already in use" + fix (FR26)

Given the agent container is OOM killed
When docker compose reports the failure
Then it reports UserError E303: "Agent container ran out of memory" + fix (FR26)

Given the build fails (e.g., pip install error)
When docker compose up --build fails
Then it reports UserError E304: "Agent build failed" with the build log excerpt + fix (FR26)

Given a warm re-deploy with no code changes
When the deploy runs
Then it completes in < 30 seconds (NFR3)
```

**Technical Notes:**
- `internal/deploy/orchestrate.go`: runs docker compose via DockerRunner
- Command: `docker compose -f .megacenter/docker-compose.yml -p {name} up -d --build`
- Error detection: parse docker compose stderr for known failure patterns
- Maps Docker errors to UserError codes E301-E304

---

#### Story 3.6: Health Check Verification

**As a** developer,
**I want** the deploy command to verify my agent is healthy after starting,
**So that** I know the deployment succeeded before seeing the summary.

**Acceptance Criteria:**

```gherkin
Given docker compose up succeeded
When the health checker runs
Then it polls GET http://localhost:{port}{health_path} (FR23)
  And it retries every 2 seconds
  And it has a default timeout of 60 seconds

Given the agent responds with HTTP 200 within timeout
When the health checker evaluates
Then it reports healthy status
  And the deploy continues to output summary

Given the agent does not respond within timeout
When the health checker times out
Then it reports UserError E305: "Health check timed out after 60s" + fix (FR26)
  And the fix suggests checking agent logs: docker logs {name}-agent-1

Given the health check is in progress
When the Presenter shows progress
Then it shows a polling indicator (e.g., "Waiting for agent health...")
```

**Technical Notes:**
- `internal/deploy/healthcheck.go`: polls health endpoint via net/http (NOT DockerRunner)
- Uses context with timeout for cancellation
- Direct HTTP call to localhost:{port}{health_path}

---

#### Story 3.7: Deploy Command Pipeline & Output

**As a** developer,
**I want** to run `megacenter deploy` and see a structured summary of what was deployed,
**So that** I know where to access my agent and dashboards.

**Acceptance Criteria:**

```gherkin
Given a valid Agentfile and .env file exist
When I run `megacenter deploy`
Then it: loads Agentfile, generates all artifacts, runs docker compose up, verifies health
  And prints structured output listing (FR24):
    - Generated artifacts (files in .megacenter/)
    - Service status with ports (agent:{port}, Prometheus:9090, Grafana:3000)
    - Summary URLs: Agent URL + Grafana dashboard URL
    - Exact command to stop services: `docker compose -f .megacenter/docker-compose.yml -p {name} down`

Given a valid Agentfile but no .env file
When I run `megacenter deploy`
Then it reports UserError E306 with fix: "Create .env file from .env.example"

Given the deploy command
When I inspect cmd/megacenter/deploy.go
Then Cobra RunE is 3-4 lines delegating to deploy.Run()
  And deploy.Run() signature: Run(ctx context.Context, p output.Presenter, dr docker.DockerRunner, agentfilePath string) error

Given first-time deploy (cold)
When I measure total time
Then it completes in < 5 minutes for a typical Python agent (NFR2)

Given subsequent deploy (warm)
When I measure total time
Then it completes in < 3 minutes (NFR3)

Given the .megacenter/ directory
When deploy runs
Then all artifacts are fully regenerated each time (no incremental)
  And no cleanup or rollback on failure (fix-and-re-run philosophy)

Given generated artifacts
When I inspect .megacenter/ contents
Then environment variable VALUES never appear — only names are referenced (NFR16)
```

**Technical Notes:**
- `internal/deploy/deploy.go`: `Run(ctx, p Presenter, dr DockerRunner, agentfilePath string) error`
- Pipeline: Load Agentfile → Generate artifacts → Orchestrate → Health check → Output summary
- `cmd/megacenter/deploy.go`: Cobra subcommand
- .megacenter/ is fully regenerated each deploy
- .env validation: file must exist if .env.example has entries

---

### Epic 4: Health Status Monitoring

#### Story 4.1: Status Health Reporting

**As a** developer with a deployed agent,
**I want** to run `megacenter status` to see the health of my agent and supporting services,
**So that** I can quickly diagnose issues without opening multiple tools.

**Acceptance Criteria:**

```gherkin
Given an agent is deployed and running
When I run `megacenter status`
Then it shows agent health state: healthy or unhealthy (FR28)
  And it shows agent uptime (FR28)
  And it shows agent port assignment (FR28)
  And it shows Prometheus running state and port 9090 (FR29)
  And it shows Grafana running state and port 3000 (FR29)

Given an agent is deployed but unhealthy
When I run `megacenter status`
Then the agent shows as unhealthy
  And a fix suggestion is provided (check logs command)

Given no agent is deployed
When I run `megacenter status`
Then it reports that no deployment was found
  And suggests running `megacenter deploy`

Given the status command
When I measure execution time
Then it completes in < 5 seconds (NFR1)
```

**Technical Notes:**
- `internal/status/status.go`: `Run(ctx, p Presenter, dr DockerRunner, agentfilePath string) error`
- Queries Docker for container states via DockerRunner
- Checks health endpoint directly for agent health state
- Read-only command — no side effects

---

#### Story 4.2: Docker Daemon Detection & Status CLI Wiring

**As a** developer,
**I want** the status command to detect if Docker has restarted and report that agents need redeployment,
**So that** I understand why my agent is not responding after a Docker restart.

**Acceptance Criteria:**

```gherkin
Given Docker daemon has restarted since last deploy
When I run `megacenter status`
Then it detects that containers are stopped/restarting (FR30)
  And reports: "Docker appears to have restarted. Run `megacenter deploy` to redeploy."

Given Docker daemon is not running
When I run `megacenter status`
Then it reports UserError E401: "Docker is not running" + fix

Given the status command
When I inspect cmd/megacenter/status.go
Then Cobra RunE is 3-4 lines delegating to status.Run()
  And status.Run() signature matches the architecture specification
```

**Technical Notes:**
- `internal/status/status.go`: detect daemon restart by checking container states
- `cmd/megacenter/status.go`: Cobra subcommand wiring
- Detection: if containers exist but are all stopped/exited, likely daemon restart

---

### Epic 5: Distribution & Installation

#### Story 5.1: Cross-Compilation & Release Pipeline

**As a** MegaCenter maintainer,
**I want** a GitHub Actions release pipeline that cross-compiles and publishes binaries,
**So that** users can download pre-built binaries for their platform.

**Acceptance Criteria:**

```gherkin
Given a git tag is pushed (e.g., v0.1.0)
When the release workflow triggers
Then it cross-compiles for darwin/arm64 and linux/amd64
  And generates SHA256 checksums for all binaries
  And creates a GitHub Release with binaries and SHA256SUMS attached

Given the CI workflow
When a PR is opened or push to main occurs
Then it runs: go build, go test -race, golangci-lint, shellcheck on install.sh

Given the CI workflow
When tests run
Then only unit tests run (no integration tests in CI)
  And integration tests are available locally via `make test-integration`

Given the release binaries
When downloaded on macOS ARM64 or Ubuntu 24.04 AMD64 or Debian 12 AMD64
Then they run without additional dependencies (NFR12)
```

**Technical Notes:**
- `.github/workflows/ci.yml`: build + test + lint + shellcheck
- `.github/workflows/release.yml`: cross-compile + GitHub Release
- Build matrix: GOOS/GOARCH combinations for darwin/arm64, linux/amd64
- Version injected via ldflags from git tag
- CI runs `go build` as first step (fast fail)

---

#### Story 5.2: Install Script

**As a** developer,
**I want** to install MegaCenter with `curl -fsSL https://... | sh`,
**So that** installation is a single command regardless of my OS.

**Acceptance Criteria:**

```gherkin
Given a macOS ARM64 system
When I run the install script
Then it detects OS=darwin, ARCH=arm64
  And downloads the correct binary from GitHub Releases (FR38)

Given a Linux AMD64 system
When I run the install script
Then it detects OS=linux, ARCH=amd64
  And downloads the correct binary (FR38)

Given the downloaded binary
When SHA256 verification is enabled [SHOULD]
Then it verifies the binary checksum against SHA256SUMS (FR39, NFR18)
  And aborts with error if checksum doesn't match

Given /usr/local/bin requires sudo
When the install script detects permission issues
Then it suggests installing to ~/.local/bin instead (FR40)
  And provides PATH configuration instructions for the user's shell (FR40)

Given the install completes successfully
When I run `megacenter --version`
Then it prints the installed version (FR41)
```

**Technical Notes:**
- `install.sh`: POSIX-compatible shell script
- OS detection: `uname -s`, arch detection: `uname -m`
- Downloads from GitHub Releases URL pattern
- SHA256 verification with `sha256sum` or `shasum -a 256`
- ShellCheck clean (validated in CI)

---

#### Story 5.3: Open-Source Project Artifacts

**As a** potential contributor,
**I want** CONTRIBUTING.md and issue templates,
**So that** I know how to contribute and report issues effectively.

**Acceptance Criteria:**

```gherkin
Given the repository root
When I inspect CONTRIBUTING.md
Then it explains: how to build, how to test, how to submit PRs
  And references the Makefile targets

Given the .github/ISSUE_TEMPLATE/ directory
When I inspect templates
Then bug_report.md exists with structured fields (description, steps to reproduce, expected, actual, environment)
  And feature_request.md exists with structured fields (description, use case, alternatives considered)

Given a new contributor
When they follow CONTRIBUTING.md
Then they can build and test locally using the Makefile
```

**Technical Notes:**
- `CONTRIBUTING.md`: build instructions, test instructions, PR guidelines
- `.github/ISSUE_TEMPLATE/bug_report.md`: GitHub issue template
- `.github/ISSUE_TEMPLATE/feature_request.md`: GitHub issue template
