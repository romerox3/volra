---
stepsCompleted: [step-01-validate-prerequisites, step-02-design-epics, step-03-create-stories, step-04-final-validation]
inputDocuments:
  - '_bmad-output/planning-artifacts/prd.md'
  - '_bmad-output/planning-artifacts/architecture.md'
executionMode: 'GENERATE'
project_name: 'Volra'
---

# Volra - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for Volra, decomposing the requirements from the PRD and Architecture into implementable stories.

## Requirements Inventory

### Functional Requirements

**Environment Diagnosis (FR1-FR3)**
- FR1: Developer can run a pre-flight check that validates all prerequisites: Docker installed, Docker running, Docker Compose V2 available, required ports free, Python >= 3.10 present, sufficient disk space, and Volra version reported
- FR2: Developer can see a pass/fail result for each individual check with a specific fix suggestion for each failure
- FR3: Developer can determine from the command exit code whether all checks passed (0) or any failed (1)

**Project Detection & Configuration (FR4-FR15)**
- FR4: Developer can point Volra at a Python project directory and receive an auto-generated Agentfile with a summary of detected values and instructions to customize
- FR5: System can detect the agent framework used (generic or LangGraph) by scanning dependency files and Python imports
- FR6: System can detect the application entry point by scanning for common filenames (main.py, app.py, server.py) in priority order
- FR7: System can detect the application port by scanning entry point code for server startup patterns
- FR8: System can detect health check endpoints by scanning route decorators
- FR9: System can detect environment variable references in agent code (os.environ, os.getenv patterns) to populate .env.example
- FR10: When detection is partial or ambiguous, system uses safe defaults and emits a warning with override instructions referencing the specific Agentfile field
- FR11: Developer can override any detected value by editing the generated Agentfile
- FR12: System generates a .env.example file listing detected environment variable names
- FR13: System adds .volra/ and .env to the project's .gitignore
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
- FR38: Developer can install Volra via a single shell command that auto-detects OS and architecture
- FR39 [SHOULD]: Installation script verifies binary integrity via SHA256 checksum before placing the binary
- FR40: When the install location requires elevated permissions, system suggests an alternative user-local path with PATH configuration instructions
- FR41: Developer can check the installed Volra version

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
- Generated output in .volra/ directory, fully regenerated each deploy
- docker compose -f .volra/docker-compose.yml -p {name} up -d

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
| FR13 | Epic 2 | Add .volra/ and .env to .gitignore |
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
Developer can verify their system is ready for Volra. Includes project scaffolding, shared infrastructure (Presenter, DockerRunner, error types, test utilities), and a fully functional `megacenter doctor` command.
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
Developer can install Volra with a single shell command. Cross-compilation, GitHub Release with binaries + SHA256 checksums, install.sh with OS/arch auto-detection, and open-source project artifacts.
**FRs covered:** FR38, FR39 [SHOULD], FR40
**NFRs addressed:** NFR6, NFR12, NFR18 [SHOULD]
**Packages:** install.sh, .github/workflows/release.yml, CONTRIBUTING.md, .github/ISSUE_TEMPLATE/

---

## Stories

### Epic 1: Foundation & Environment Readiness

#### Story 1.1: Project Scaffolding & Build System

**As a** developer contributing to Volra,
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

**As a** Volra developer,
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

**As a** Volra developer,
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

**As a** developer setting up Volra,
**I want** to run `megacenter doctor` to validate all prerequisites,
**So that** I know my system is ready before attempting to deploy an agent.

**Acceptance Criteria:**

```gherkin
Given Docker is installed and running
When I run `megacenter doctor`
Then it checks: Docker installed, Docker running, Compose V2 available, Python >= 3.10 present, sufficient disk space (1GB)
  And each check shows pass/fail with a specific fix suggestion on failure (FR2)
  And Volra version is reported in the output (FR41)
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
Then the check fails with error E105 and fix: "Free up disk space. Volra needs at least 1GB."

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

**As a** Volra developer,
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
**So that** my project is configured for Volra deployment.

**Acceptance Criteria:**

```gherkin
Given a Python project directory with no existing Agentfile
When I run `megacenter init ./my-agent`
Then it generates Agentfile in ./my-agent/ with detected values (FR4)
  And generates .env.example listing detected env var names (FR12)
  And adds .volra/ and .env to .gitignore (FR13)
  And prints a summary of detected values and instructions to customize (FR4)

Given a project directory with an existing Agentfile
When I run `megacenter init ./my-agent`
Then it exits with error E206 and suggests `--force` flag (FR15)

Given a project directory with an existing Agentfile
When I run `megacenter init --force ./my-agent`
Then it overwrites the existing Agentfile (FR15)

Given .gitignore already contains .volra/
When init runs
Then it does not duplicate the entry (FR13)

Given no .gitignore exists
When init runs
Then it creates .gitignore with .volra/ and .env entries (FR13)

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

**As a** Volra developer,
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
- Output written to `.volra/Dockerfile`

---

#### Story 3.2: Docker Compose Generation

**As a** Volra developer,
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
- Output written to `.volra/docker-compose.yml`
- FR36 (anonymous Grafana + default dashboard) implemented as ACs here

---

#### Story 3.3: Prometheus Configuration & Alert Rules

**As a** Volra developer,
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
- Output: `.volra/prometheus.yml` and `.volra/alert_rules.yml`

---

#### Story 3.4: Grafana Dashboards & Provisioning

**As a** developer deploying with Volra,
**I want** pre-built Grafana dashboards showing agent health, uptime, and latency,
**So that** I get instant observability without manual dashboard configuration.

**Acceptance Criteria:**

```gherkin
Given the deploy command generates Grafana assets
When I inspect the output
Then the following files are generated in .volra/:
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
Given all artifacts are generated in .volra/
When the deploy orchestrator runs
Then it executes: docker compose -f .volra/docker-compose.yml -p {name} up -d --build (FR22)
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
- Command: `docker compose -f .volra/docker-compose.yml -p {name} up -d --build`
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
    - Generated artifacts (files in .volra/)
    - Service status with ports (agent:{port}, Prometheus:9090, Grafana:3000)
    - Summary URLs: Agent URL + Grafana dashboard URL
    - Exact command to stop services: `docker compose -f .volra/docker-compose.yml -p {name} down`

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

Given the .volra/ directory
When deploy runs
Then all artifacts are fully regenerated each time (no incremental)
  And no cleanup or rollback on failure (fix-and-re-run philosophy)

Given generated artifacts
When I inspect .volra/ contents
Then environment variable VALUES never appear — only names are referenced (NFR16)
```

**Technical Notes:**
- `internal/deploy/deploy.go`: `Run(ctx, p Presenter, dr DockerRunner, agentfilePath string) error`
- Pipeline: Load Agentfile → Generate artifacts → Orchestrate → Health check → Output summary
- `cmd/megacenter/deploy.go`: Cobra subcommand
- .volra/ is fully regenerated each deploy
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

**As a** Volra maintainer,
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
**I want** to install Volra with `curl -fsSL https://... | sh`,
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

---

## Epic 6: Operational Polish (v1.1)

**Goal:** Address 3 operational limitations discovered during E2E testing: health timeout too short for ML agents, Docker images unnecessarily large, no custom metrics in dashboards.

**FRs covered:** FR42, FR43, FR44
**NFRs covered:** NFR24 (backward compatibility), NFR25 (functional equivalence)
**Dependencies:** Epic 3 (deploy infrastructure must be complete)
**Risk:** Medium — changes touch templates and dashboards but are additive (no breaking changes)

### Story 6.1: Configurable Health Timeout

**As a** developer deploying an ML agent with a 2-minute model loading time,
**I want** to configure the health check timeout in my Agentfile,
**So that** Volra waits long enough for my agent to start instead of failing with a timeout error.

**Acceptance Criteria:**

```gherkin
Given an Agentfile with health_timeout: 300
When I run volra deploy
Then the health check waits up to 300 seconds before timing out
  And the timeout error message shows the configured timeout value

Given an Agentfile without a health_timeout field
When I run volra deploy
Then the health check uses the default 60-second timeout
  And behavior is identical to v1.0

Given an Agentfile with health_timeout: 5
When I run volra deploy
Then validation fails with error "Invalid field: health_timeout — 5 is out of range (10-600)"
  And the fix suggests "Set health_timeout between 10 and 600 seconds"

Given an Agentfile with health_timeout: 700
When I run volra deploy
Then validation fails with error "Invalid field: health_timeout — 700 is out of range (10-600)"

Given a v1.0 Agentfile (valid_full.yaml, valid_minimal.yaml)
When I parse and validate it with the v1.1 CLI
Then parsing succeeds without errors
  And health_timeout defaults to 0 (interpreted as 60s)
```

**Technical Notes:**
- Add `HealthTimeout int` field to `Agentfile` struct with `yaml:"health_timeout,omitempty"`
- Add `validateHealthTimeout()` to validation chain in `validate.go`
- Change `WaitForHealth()` signature to accept `timeout time.Duration` parameter
- Update `deploy.go:Run()` to pass resolved timeout (Agentfile value or default 60s)
- Update error messages in `WaitForHealth` to show actual configured timeout
- Add 4 test fixtures: `valid_health_timeout.yaml`, `valid_no_health_timeout.yaml`, `invalid_health_timeout_low.yaml`, `invalid_health_timeout_high.yaml`
- Update `healthcheck_test.go` to test configurable timeout

**Files changed:**
- `internal/agentfile/agentfile.go` — new field
- `internal/agentfile/validate.go` — new validation function
- `internal/deploy/healthcheck.go` — configurable timeout parameter
- `internal/deploy/deploy.go` — pass timeout to WaitForHealth
- `internal/agentfile/agentfile_test.go` — new test cases
- `internal/deploy/healthcheck_test.go` — updated tests
- `internal/agentfile/testdata/*.yaml` — new fixtures

---

### Story 6.2: Multi-Stage Dockerfile Builds

**As a** developer deploying agents to production,
**I want** Volra to generate optimized multi-stage Dockerfiles,
**So that** my container images are smaller and build faster with pip cache.

**Acceptance Criteria:**

```gherkin
Given an Agentfile with dockerfile: auto and a requirements.txt
When I run volra deploy
Then the generated Dockerfile uses a multi-stage build (builder + runtime)
  And the builder stage uses --mount=type=cache for pip
  And the runtime stage only contains installed packages and application code
  And the Dockerfile starts with "# Generated by Volra"

Given an Agentfile with dockerfile: auto and pyproject.toml only
When I run volra deploy
Then the generated Dockerfile uses multi-stage build with pip install .
  And pip cache mount is used in the builder stage

Given an Agentfile with dockerfile: custom
When I run volra deploy
Then no Dockerfile is generated (unchanged from v1.0)

Given the multi-stage Dockerfile golden files
When I compare rendered output
Then dockerfile_requirements.golden matches the multi-stage pattern
  And dockerfile_pyproject.golden matches the multi-stage pattern
```

**Technical Notes:**
- Replace `Dockerfile.tmpl` content with multi-stage build pattern
- Builder stage: `FROM python:X.Y-slim AS builder`, install deps with `--prefix=/install` and `--mount=type=cache`
- Runtime stage: `FROM python:X.Y-slim`, `COPY --from=builder /install /usr/local`, copy app code
- No TemplateContext changes needed — same 3 metadata fields
- Update golden files: `dockerfile_requirements.golden`, `dockerfile_pyproject.golden`
- Existing tests remain valid, just golden files change

**Files changed:**
- `internal/deploy/templates/Dockerfile.tmpl` — rewritten with multi-stage pattern
- `internal/deploy/testdata/golden/dockerfile_requirements.golden` — regenerated
- `internal/deploy/testdata/golden/dockerfile_pyproject.golden` — regenerated

---

### Story 6.3: Custom Metrics Dashboard Panels

**As a** developer whose agent exposes Prometheus metrics via prometheus_client,
**I want** Volra to auto-detect this and add custom metrics panels to my Grafana dashboards,
**So that** I can see request rates, latencies, and custom counters alongside health probes.

**Acceptance Criteria:**

```gherkin
Given a project with prometheus_client in requirements.txt
When I run volra deploy
Then HasMetrics is true in the template context
  And the Overview dashboard includes Request Rate and Active Requests panels
  And the Detail dashboard includes Request Rate, P95 Latency, and Custom Metrics panels
  And custom panels are clearly labeled as "Application-reported metric"

Given a project with prometheus-client in pyproject.toml
When I run volra deploy
Then HasMetrics is true (hyphen variant also detected)

Given a project without prometheus_client in any dependency file
When I run volra deploy
Then HasMetrics is false
  And dashboards contain only the probe-based panels (same as v1.0)
  And behavior is identical to v1.0

Given a project with HasMetrics true
When I inspect the generated Prometheus config
Then the agent-metrics scrape job exists (already present in v1.0)
  And no additional Prometheus configuration is needed
```

**Technical Notes:**
- Add `HasMetrics bool` field to `TemplateContext`
- Add `detectMetricsLibrary(dir string) bool` function in `context.go`
- Create `overview_metrics.json` — copy of `overview.json` + 2 additional panels
- Create `detail_metrics.json` — copy of `detail.json` + 3 additional panels
- Update `grafana.go` to select dashboard variant based on `HasMetrics`
- Prometheus config already has `agent-metrics` job — no changes needed
- Detection: case-insensitive check for `prometheus_client` or `prometheus-client` in requirements.txt and pyproject.toml

**Files changed:**
- `internal/deploy/context.go` — new field + detection function
- `internal/deploy/grafana.go` — dashboard selection logic
- `internal/deploy/static/overview_metrics.json` — new dashboard variant
- `internal/deploy/static/detail_metrics.json` — new dashboard variant
- `internal/deploy/dockerfile_test.go` — updated BuildContext tests
- `internal/deploy/grafana_test.go` — new dashboard selection tests

---

## Epic 7: Data Persistence & LLM Observability (v1.2)

**Goal:** Persistent volumes for agent data + LLM token tracking dashboard panels.
**FRs:** FR45 (volumes), FR46 (LLM token tracking panels — revised, extends FR44)
**Status:** Implementation-ready — stories detailed with GWT acceptance criteria.

### Story 7.1: Persistent Volume Mounts

**As a** developer deploying an AI agent,
**I want** to declare persistent volume mounts in my Agentfile,
**so that** agent data (model weights, databases, caches) survives container rebuilds.

**FR mapping:** FR45
**Effort:** Medium

#### Acceptance Criteria

```gherkin
Given an Agentfile with volumes: [/data, /models]
When I run volra deploy
Then the docker-compose.yml includes named volumes my-agent-data and my-agent-models
  And the agent service mounts my-agent-data:/data and my-agent-models:/models
  And both volumes appear in the top-level volumes section alongside prometheus-data

Given an Agentfile without a volumes field
When I run volra deploy
Then the docker-compose.yml is identical to v1.1 output
  And no extra volumes are declared beyond prometheus-data

Given an Agentfile with volumes: [data] (no leading /)
When I run volra deploy
Then validation fails with error "must be absolute path"
  And the fix suggests "Use absolute paths starting with / for volume mounts"

Given an Agentfile with volumes: [/app/data]
When I run volra deploy
Then validation fails with error "conflicts with container WORKDIR /app"
  And the fix explains that /app is reserved for agent code

Given an Agentfile with volumes: [/data, /data]
When I run volra deploy
Then validation fails with error "duplicate volume path"

Given a v1.0 or v1.1 Agentfile (valid_full.yaml, valid_minimal.yaml, valid_health_timeout.yaml)
When I parse it with the v1.2 CLI
Then parsing succeeds without errors
  And Volumes defaults to nil (empty)

Given an Agentfile with name: my-agent and volumes: [/data/models]
When VolumeSpecs are computed in BuildContext
Then the volume name is "my-agent-data-models" (leading / removed, remaining / replaced with -)
```

#### Technical Notes

- `Volumes []string` field added to Agentfile struct with `yaml:"volumes,omitempty"`
- `validateVolumes()` added to validation chain in validate.go
- `VolumeSpec{Name, MountPath}` type in context.go, computed by `BuildContext`
- `docker-compose.yml.tmpl` extended with conditional `{{- if .VolumeSpecs}}` blocks
- Volume name formula: `{agent-name}-{path.TrimPrefix("/").ReplaceAll("/", "-")}`
- Max 10 volumes to prevent compose file bloat

#### Files Changed

| File | Change |
|------|--------|
| `internal/agentfile/agentfile.go` | Add `Volumes []string` field |
| `internal/agentfile/validate.go` | Add `validateVolumes()` to chain |
| `internal/deploy/context.go` | Add `VolumeSpec` type, compute in `BuildContext` |
| `internal/deploy/templates/docker-compose.yml.tmpl` | Conditional agent volumes + top-level declarations |
| `internal/agentfile/testdata/valid_volumes.yaml` | New fixture |
| `internal/agentfile/testdata/invalid_volumes_not_absolute.yaml` | New fixture |
| `internal/agentfile/testdata/invalid_volumes_app_path.yaml` | New fixture |
| `internal/agentfile/testdata/invalid_volumes_duplicate.yaml` | New fixture |
| `internal/deploy/testdata/golden/compose_volumes.golden` | New golden file |

---

### Story 7.2: LLM Token Tracking Dashboard Panels

**As a** developer running an LLM-powered agent with prometheus_client,
**I want** dedicated dashboard panels for token consumption, cost, and per-model breakdowns,
**so that** I can monitor LLM usage and costs alongside probe and request metrics.

**FR mapping:** FR46 (revised — extends FR44 dashboard variants)
**Effort:** Small

#### Acceptance Criteria

```gherkin
Given overview_metrics.json
When I inspect its panels
Then it contains a "LLM Token Rate" stat panel
  And the panel uses expr sum(rate(llm_tokens_total{job="agent-metrics"}[5m]))
  And the panel description mentions "Volra LLM Metrics Convention"

Given detail_metrics.json
When I inspect its panels
Then it contains "Token Consumption Over Time" timeseries panel with llm_tokens_total by direction
  And it contains "LLM Cost Trending" timeseries panel with llm_request_cost_dollars_total
  And it contains "Per-Model Request Breakdown" bargauge panel with llm_model_requests_total by model
  And all three panels use job="agent-metrics" selector
  And all three panels mention "Volra LLM Metrics Convention" in description

Given a project with HasMetrics=true (prometheus_client detected)
When I run volra deploy
Then the overview dashboard file contains "llm_tokens_total" in its content
  And the detail dashboard file contains "llm_request_cost_dollars_total" in its content

Given a project with HasMetrics=false
When I run volra deploy
Then probe-only dashboards are deployed (overview.json, detail.json)
  And no LLM or custom metrics panels are present
```

#### Technical Notes

- No new Go source code changes needed — only static JSON file edits
- Extend existing `overview_metrics.json` with 1 new panel (total: 6 panels)
- Extend existing `detail_metrics.json` with 3 new panels (total: 10 panels)
- Panel IDs must be unique within each dashboard (continue from existing max ID)
- All LLM panels use `job="agent-metrics"` — same Prometheus scrape job as generic custom metrics
- Grafana displays "No data" for panels referencing metrics that don't exist — no error handling needed

#### Files Changed

| File | Change |
|------|--------|
| `internal/deploy/static/overview_metrics.json` | Add LLM Token Rate panel |
| `internal/deploy/static/detail_metrics.json` | Add 3 LLM panels (Token Consumption, Cost Trending, Per-Model Breakdown) |
| `internal/deploy/grafana_test.go` | Add test assertions for new LLM panels |

---

## Epic 8: Infrastructure Services (v2.0)

**Goal:** Allow developers to declare sidecar infrastructure services (Redis, PostgreSQL, etc.) in Agentfile for automatic docker-compose generation with networking and dependencies.
**FRs:** FR47
**Schema:** Version 1 retained (no bump). KnownFields handles forward compatibility.

---

### Story 8.1: Service Schema & Validation

**As a** developer deploying an AI agent with infrastructure dependencies,
**I want** to declare services like Redis and PostgreSQL in my Agentfile,
**so that** the CLI validates my service definitions before generating deployment artifacts.

**FR mapping:** FR47 (schema and validation portion)
**Effort:** Medium

#### Acceptance Criteria

```gherkin
Given an Agentfile with services (redis with image, db with image+port+volumes+env)
When I parse the Agentfile
Then Services map has 2 entries with correct fields (Image, Port, Volumes, Env)

Given a service with reserved name "agent"
When I validate the Agentfile
Then validation fails with error "reserved service name"
  And the fix lists reserved names (agent, prometheus, grafana, blackbox)

Given a service without an image field
When I validate the Agentfile
Then validation fails with error "image is required"

Given a service with port equal to the agent's port
When I validate the Agentfile
Then validation fails with error "port conflict"
  And the fix explains which port conflicts

Given 6 services declared (over max 5)
When I validate the Agentfile
Then validation fails with error "too many services (max 5)"

Given a v1.0, v1.1, or v1.2 Agentfile without a services field
When I parse it with the v2.0 CLI
Then parsing succeeds without errors
  And Services defaults to nil (empty map)

Given a service with DNS-invalid name (uppercase, dots, etc.)
When I validate the Agentfile
Then validation fails with the existing DNS label error pattern
```

#### Technical Notes

- New `Service` struct in agentfile.go with `Image`, `Port`, `Volumes`, `Env` fields
- `Services map[string]Service` field added to Agentfile with `yaml:"services,omitempty"`
- `validateServices()` added to validation chain in validate.go
- Reuse `dnsLabelRegex` for service name validation
- Reuse volume/env validation logic for per-service fields
- Test fixtures: valid_services.yaml, invalid_services_reserved.yaml, invalid_services_no_image.yaml, invalid_services_port_conflict.yaml, invalid_services_too_many.yaml

#### Files Changed

| File | Change |
|------|--------|
| `internal/agentfile/agentfile.go` | Add Service struct + Services field |
| `internal/agentfile/validate.go` | Add validateServices() |
| `internal/agentfile/agentfile_test.go` | Add service validation + parse tests |
| `internal/agentfile/testdata/*.yaml` | Add 5 test fixtures |

---

### Story 8.2: Service Compose Generation

**As a** developer deploying an AI agent with declared services,
**I want** `volra deploy` to generate docker-compose service definitions with automatic networking,
**so that** my agent can connect to Redis, PostgreSQL, etc. without manual compose configuration.

**FR mapping:** FR47 (compose generation portion)
**Effort:** Medium

#### Acceptance Criteria

```gherkin
Given 2 services declared (redis + db)
When I render the docker-compose template
Then output contains both service blocks with correct images and container names ({projectName}-{serviceName})
  And both services are on the volra network

Given a service with port: 5432
When I render the docker-compose template
Then the service has port mapping "5432:5432"

Given a service with env: [POSTGRES_PASSWORD]
When I render the docker-compose template
Then the service has env_file: ../.env directive

Given a service with volumes: [/var/lib/postgresql/data]
When I render the docker-compose template
Then the service has volume mount with named volume
  And the named volume appears in the top-level volumes section

Given no services declared
When I render the docker-compose template
Then the output is identical to v1.2 output (backward compat)

Given 2 services declared
When I render the docker-compose template
Then the agent service has depends_on listing all service container names

Given 2 services (db before redis alphabetically)
When I render the docker-compose template
Then services appear in alphabetical order (deterministic output for golden file testing)
```

#### Technical Notes

- `ServiceContext` type in context.go with Name, Image, Port, Env, VolumeSpecs
- `buildServiceContexts()` function computes VolumeSpecs per service, sorts by name
- `BuildContext` updated to populate ServiceContexts
- docker-compose.yml.tmpl: range loop for services, depends_on for agent, service volumes in top-level
- Volume naming: `{projectName}-{serviceName}-{sanitized-path}`
- Golden file: compose_services.golden

#### Files Changed

| File | Change |
|------|--------|
| `internal/deploy/context.go` | Add ServiceContext type, buildServiceContexts() |
| `internal/deploy/templates/docker-compose.yml.tmpl` | Add service range loop, depends_on, service volumes |
| `internal/deploy/compose_test.go` | Add service compose tests + golden |
| `internal/deploy/dockerfile_test.go` | Add BuildContext service tests |
| `internal/deploy/testdata/golden/compose_services.golden` | New golden file |

---

## Epic 9: Container Runtime Configuration (v2.0)

**Goal:** Container security hardening and GPU acceleration support for AI agent deployments.
**FRs:** FR48 (security context), FR50 (GPU)

---

### Story 9.1: Security Context Defaults

**As a** developer deploying an AI agent in a security-conscious environment,
**I want** to declare container security settings in my Agentfile,
**so that** my agent runs with hardened defaults (read-only filesystem, dropped capabilities).

**FR mapping:** FR48
**Effort:** Small

#### Acceptance Criteria

```gherkin
Given an Agentfile with security (read_only: true, no_new_privileges: true, drop_capabilities: [ALL])
When I render the docker-compose template
Then output contains read_only: true, security_opt with no-new-privileges, and cap_drop with ALL

Given an Agentfile with only security.read_only: true
When I render the docker-compose template
Then only read_only: true directive is present
  And no security_opt or cap_drop directives appear

Given an Agentfile without a security field
When I render the docker-compose template
Then no security directives are present (backward compat)

Given a v1.0-v1.2 Agentfile
When I parse it with the v2.0 CLI
Then parsing succeeds with nil Security pointer
```

#### Technical Notes

- `SecurityContext` struct with ReadOnly, NoNewPrivileges, DropCapabilities
- `Security *SecurityContext` pointer field in Agentfile (nil = not set)
- Template conditionals for each security directive independently
- No Agentfile-level validation of capability names — Docker validates at runtime

#### Files Changed

| File | Change |
|------|--------|
| `internal/agentfile/agentfile.go` | Add SecurityContext struct + Security field |
| `internal/agentfile/agentfile_test.go` | Add security parse tests |
| `internal/agentfile/testdata/valid_security.yaml` | New fixture |
| `internal/deploy/templates/docker-compose.yml.tmpl` | Add security directives |
| `internal/deploy/compose_test.go` | Add security compose tests |

---

### Story 9.2: GPU/Hardware Acceleration

**As a** developer deploying an AI agent that uses GPU inference,
**I want** to declare GPU requirements in my Agentfile,
**so that** Docker configures nvidia runtime and GPU access automatically.

**FR mapping:** FR50
**Effort:** Small

#### Acceptance Criteria

```gherkin
Given an Agentfile with gpu: true
When I render the docker-compose template
Then output contains deploy.resources.reservations.devices with driver: nvidia, count: all, capabilities: [gpu]

Given an Agentfile without gpu or with gpu: false
When I render the docker-compose template
Then no deploy.resources block is present (backward compat)

Given a v1.0-v1.2 Agentfile
When I parse it with the v2.0 CLI
Then parsing succeeds with GPU=false
```

#### Technical Notes

- `GPU bool` field in Agentfile with `yaml:"gpu,omitempty"`
- Defaults to nvidia driver (only mature Docker GPU runtime)
- Template-only testing — cannot E2E test without GPU hardware
- No validation needed — boolean field

#### Files Changed

| File | Change |
|------|--------|
| `internal/agentfile/agentfile.go` | Add GPU field |
| `internal/agentfile/agentfile_test.go` | Add GPU parse tests |
| `internal/agentfile/testdata/valid_gpu.yaml` | New fixture |
| `internal/deploy/templates/docker-compose.yml.tmpl` | Add GPU deploy block |
| `internal/deploy/compose_test.go` | Add GPU compose tests |

---

## Deferred: WebSocket/SSE Streaming Probes (FR49)

**Status:** DEFERRED to backlog. No current E2E test agent uses streaming connections. Will be planned when a real use case emerges and requires dedicated health probe architecture for long-lived connections.

---

## v0.2 Epics (Post-Market-Research Strategic Pivot)

*Added March 2026 after market research and sprint change proposal. Designed via Party Mode (Lisa/Frink/Homer) consensus. These epics implement the "Own your agent infrastructure" strategy with parallel tracks: Launch (Epic 11) + Differentiation (Epics 12-15).*

---

## Epic 11: Launch Readiness (P0 — Track A) — **DONE**

**Goal:** Package v0.1 for early access public launch. Code is complete — this epic is about packaging, distribution, and communication.

**Timeline:** This week (2-3 days)
**Status:** COMPLETE — All stories implemented.

### Story 11.1: Create User-Facing README.md

**Given** a developer finds the Volra GitHub repository
**When** they read the README
**Then** they understand: (1) what Volra does ("own your agent infrastructure"), (2) how to install it (`curl | sh`), (3) how to use it (`volra init → deploy → status`), (4) what it generates (Docker Compose + Prometheus + Grafana), (5) link to examples

**Acceptance Criteria:**
- README includes: tagline, 30-second quickstart, feature list, screenshot/demo, comparison table vs competitors, link to examples
- No mention of "Volra" — all references use "Volra"
- Tone: developer-to-developer, not marketing

**Files:** `README.md` (NEW)

### Story 11.2: Create Install Script

**Given** a developer on macOS ARM64 or Linux AMD64
**When** they run `curl -fsSL https://get.volra.dev | sh`
**Then** the Volra binary is installed to `/usr/local/bin/volra` and `volra --version` works

**Acceptance Criteria:**
- Detects OS (darwin/linux) and arch (arm64/amd64)
- Downloads correct binary from GitHub Releases
- Verifies SHA256 checksum before installing
- Falls back to `~/.local/bin` with PATH instructions if no write permission
- Prints success message with next steps

**Files:** `install.sh` (NEW)

### Story 11.3: Configure GitHub Releases Pipeline

**Given** a git tag is pushed (e.g., `v0.1.0`)
**When** GitHub Actions runs
**Then** cross-compiled binaries for macOS ARM64, Linux AMD64, and Linux ARM64 are uploaded as release assets with SHA256 checksums

**Acceptance Criteria:**
- GitHub Actions workflow builds for 3 targets
- Each binary includes version info from git tag
- SHA256 checksum file generated alongside binaries
- Release notes auto-generated from commits

**Files:** `.github/workflows/release.yml` (NEW), `Makefile` (UPDATE — add release targets)

### Story 11.4: Rename Repository to Volra

**Given** the GitHub repository is named "Volra"
**When** the rename is executed
**Then** the repo is accessible as `github.com/{owner}/volra`

**Acceptance Criteria:**
- GitHub repo renamed (GitHub handles redirects from old name)
- Go module path updated in `go.mod`
- All internal import paths updated
- CI/CD references updated
- README reflects new URL

**Risk:** HIGH — all import paths change. Must be done atomically with a passing test suite.

**Files:** `go.mod`, all `*.go` files with import paths, `.github/workflows/*`

### Story 11.5: Create Example Projects

**Given** a developer wants to try Volra
**When** they clone an example project
**Then** they can run `volra init → deploy` and see it working

**Acceptance Criteria:**
- 3 example directories: `examples/basic/`, `examples/rag/`, `examples/conversational/`
- Each has: `main.py`, `requirements.txt`, `Agentfile`, `README.md` with step-by-step instructions
- Converted from E2E test agents (A1, A4, A5) with user-facing docs
- Each example tested end-to-end

**Files:** `examples/basic/*`, `examples/rag/*`, `examples/conversational/*` (NEW)

### Story 11.6: Write Launch Announcements

**Given** v0.1 is packaged and published
**When** launch announcements are posted
**Then** Hacker News, Reddit r/MachineLearning, and Twitter posts are live

**Acceptance Criteria:**
- HN post: "Show HN: Volra — Own your agent infrastructure (open-source CLI)"
- Reddit post: focused on self-hosted differentiator
- Twitter thread: demo GIF + key features
- All link to GitHub repo with clear README

**Files:** `_bmad-output/planning-artifacts/launch-posts.md` (NEW — drafts)

---

## Epic 12: Adoption Accelerators (P0 — Week 1-2) — **DONE**

**Goal:** Reduce Day 0 friction with templates and basic operational commands.
**Status:** COMPLETE — `volra quickstart` with 3 templates (basic, rag, conversational) + `volra logs` implemented.

### Story 12.1: Implement `volra quickstart` Command

**Given** a developer runs `volra quickstart`
**When** no template is specified
**Then** they see a list of available templates with descriptions

**Given** a developer runs `volra quickstart rag my-agent`
**When** the template exists
**Then** a new directory `my-agent/` is created with the template files, name placeholders replaced

**Acceptance Criteria:**
- Lists templates when invoked without arguments
- Copies template to target directory with name substitution
- Prints next steps: "cd my-agent && volra deploy"
- Error if target directory already exists
- Templates embedded via `go:embed`

**Architecture ref:** Extension: `volra quickstart` Command

**Files:** `cmd/quickstart.go` (NEW), `internal/templates/templates.go` (NEW), `internal/templates/*/` (NEW)

### Story 12.2: Create "basic" Template

**Given** a developer runs `volra quickstart basic my-agent`
**When** the template is scaffolded
**Then** `my-agent/` contains a working FastAPI agent with health endpoint

**Acceptance Criteria:**
- `main.py`: FastAPI app with `/health` and `/ask` endpoints
- `requirements.txt`: fastapi, uvicorn
- `Agentfile`: framework: generic, port: 8000
- `README.md`: Quick explanation + `volra deploy` instructions
- Deploys successfully with `volra deploy`

**Files:** `internal/templates/basic/*` (NEW)

### Story 12.3: Create "rag" Template

**Given** a developer runs `volra quickstart rag my-rag`
**When** the template is scaffolded
**Then** `my-rag/` contains a working RAG agent with ChromaDB service

**Acceptance Criteria:**
- `main.py`: LangChain RAG pipeline with ChromaDB
- `requirements.txt`: langchain, chromadb, fastapi
- `Agentfile`: framework: generic, port: 8000, services: { chromadb }
- `.env.example`: OPENAI_API_KEY
- Deploys successfully with ChromaDB service

**Files:** `internal/templates/rag/*` (NEW)

### Story 12.4: Create "conversational" Template

**Given** a developer runs `volra quickstart conversational my-bot`
**When** the template is scaffolded
**Then** `my-bot/` contains a conversational agent with Redis + PostgreSQL

**Acceptance Criteria:**
- `main.py`: LangGraph agent with conversation memory
- `requirements.txt`: langgraph, langchain, psycopg2-binary, redis
- `Agentfile`: framework: langgraph, services: { redis, postgres }
- Deploys successfully with both services

**Files:** `internal/templates/conversational/*` (NEW)

### Story 12.5: Implement `volra logs` Command

**Given** a deployed agent exists
**When** the developer runs `volra logs`
**Then** they see the agent's recent logs

**Given** `volra logs -f` is run
**When** new logs are produced
**Then** they stream in real-time

**Acceptance Criteria:**
- Default: shows last 100 lines of agent container logs
- `--follow` / `-f`: streams logs in real-time
- `--lines` / `-n`: configure tail count
- Optional service name argument: `volra logs redis`
- Error if no deployment exists

**Architecture ref:** Extension: `volra logs` Command

**Files:** `cmd/logs.go` (NEW)

---

## Epic 13: Agent Observability Level 2 (P1 — Week 2-5) — **DONE**

**Goal:** Ship framework-agnostic LLM observability that works with any Python agent — reinforcing Volra's core differentiator.
**Status:** COMPLETE — `volra-observe` Python package (25 tests), Level 2 Grafana dashboard (9 panels), Prometheus Level 2 scrape config, doctor Level 2 check.

### Story 13.1: Add `observability` Agentfile Struct

**Given** an Agentfile with `observability: { level: 2, metrics_port: 9101 }`
**When** parsed by Volra
**Then** the observability config is available in TemplateContext

**Acceptance Criteria:**
- `ObservabilityConfig` struct: `level` (1 or 2), `metrics_port` (default 9101)
- Validation: level must be 1 or 2, metrics_port valid range
- NO framework constraint — level 2 works with `generic` and `langgraph`
- Backward compatible: omitted = level 1
- TemplateContext includes `HasLevel2` boolean shorthand

**Architecture ref:** Extension 8: Agent Observability Level 2

**Files:** `internal/agentfile/agentfile.go`, `internal/agentfile/validate.go`, `internal/deploy/context.go`

### Story 13.2a: Spike — Research volra-observe SDK Instrumentation

**Given** we need to instrument OpenAI and Anthropic Python SDKs
**When** research is complete
**Then** we have: (1) working prototype patching both SDKs, (2) package structure, (3) PyPI distribution plan

**Acceptance Criteria:**
- Prototype monkey-patching OpenAI `chat.completions.create()` to extract tokens + latency
- Prototype monkey-patching Anthropic `messages.create()` to extract tokens + latency
- Confirmed: both SDKs expose token counts in response objects
- Package structure defined (pyproject.toml, src layout)
- Decision: monkey-patching vs wrapper approach
- Prometheus metrics server starts on configurable port

**Risk:** SDK monkey-patching may break with major version updates. Spike validates approach stability.

**Files:** `volra-observe/` (NEW repo or subdirectory — spike only)

### Story 13.2b: Implement volra-observe Package

**Given** the spike is complete
**When** the package is published to PyPI
**Then** `pip install volra-observe` provides framework-agnostic LLM instrumentation

**Acceptance Criteria:**
- Auto-patches OpenAI and Anthropic SDKs when `volra_observe.init()` is called
- Exposes 5 core metrics: tokens, cost, latency, errors, tool calls
- Starts Prometheus HTTP server on configurable port (default 9101)
- Embedded pricing table for GPT-4o, GPT-4o-mini, Claude Sonnet, Claude Haiku
- Decorator `@track_llm` and context manager `llm_context` for manual instrumentation
- Works with any Python agent (FastAPI, Flask, plain script)
- PyPI package published as `volra-observe`

**Files:** `volra-observe/` (separate package)

### Story 13.3: Extend Prometheus Config for Level 2

**Given** an Agentfile with `observability.level: 2`
**When** `volra deploy` generates prometheus.yml
**Then** it includes a scrape target for `:9101/metrics`

**Acceptance Criteria:**
- New scrape job `agent-level2` targeting `{name}:{metrics_port}`
- Only generated when level: 2 (regardless of framework)
- Existing probe scraping unchanged

**Files:** `internal/deploy/templates/prometheus.yml.tmpl`, `internal/deploy/context.go`

### Story 13.4: Create Level 2 Grafana Dashboard

**Given** Level 2 observability is enabled
**When** `volra deploy` generates Grafana dashboards
**Then** a Level 2 dashboard is included with token, cost, latency, and error panels

**Acceptance Criteria:**
- New dashboard: `agent-level2-overview.json`
- Panels: token rate, cost trending, cost per model, token usage (input/output), LLM latency P50/P95/P99, error rate by type, daily cost gauge, tool call frequency
- "No data" annotations when metrics not yet available
- Auto-provisioned alongside existing dashboards

**Files:** `internal/deploy/grafana/dashboards/agent-level2-overview.json` (NEW)

### Story 13.5: Add Level 2 Doctor Checks

**Given** an Agentfile with `observability.level: 2`
**When** `volra doctor` runs
**Then** it checks if the Level 2 metrics endpoint responds

**Acceptance Criteria:**
- New doctor check: HTTP GET to `localhost:{metrics_port}/metrics`
- Warning (not error) if endpoint not reachable: "Install volra-observe to enable Level 2 metrics"
- Only checked when level: 2

**Files:** `internal/doctor/checks.go`

---

## Epic 14: MCP Integration (P1 — Week 3-5) — **DONE**

**Goal:** Competitive parity with Railway — enable deployment from AI-powered editors.
**Status:** COMPLETE — MCP server with 4 tools (deploy/status/logs/doctor), JSON-RPC 2.0 protocol, `volra mcp` command, editor config docs (Cursor/VS Code/Claude Code). 13 tests.

### Story 14.1: Implement MCP Protocol Layer

**Given** the MCP protocol specification
**When** `internal/mcp/` package is implemented
**Then** it can parse JSON-RPC requests and emit responses

**Acceptance Criteria:**
- `protocol.go`: MCP types (Request, Response, Tool, ToolCall, etc.)
- `server.go`: stdin/stdout JSON-RPC loop
- Handles: `initialize`, `tools/list`, `tools/call` methods
- Proper error responses for unknown methods

**Architecture ref:** Extension 9: Volra MCP Server

**Files:** `internal/mcp/protocol.go` (NEW), `internal/mcp/server.go` (NEW)

### Story 14.2: Implement MCP Tool Handlers

**Given** the MCP server receives a `tools/call` request
**When** the tool name matches a registered handler
**Then** the handler executes the corresponding Volra operation

**Acceptance Criteria:**
- 4 tools implemented: `volra_deploy`, `volra_status`, `volra_logs`, `volra_doctor`
- Each tool has: name, description, inputSchema (JSON Schema)
- Tool handlers call existing Volra functions (deploy.Run, status.Run, etc.)
- Results returned as MCP content blocks (text type)

**Files:** `internal/mcp/handler.go` (NEW), `internal/mcp/tools.go` (NEW)

### Story 14.3: Add `volra mcp` Command

**Given** a developer configures their editor to use Volra as MCP server
**When** they type `volra mcp` (or editor launches it)
**Then** the MCP server starts on stdio

**Acceptance Criteria:**
- New Cobra command: `volra mcp`
- Reads from stdin, writes to stdout
- Debug logs to stderr
- Graceful shutdown on SIGTERM/SIGINT

**Files:** `cmd/mcp.go` (NEW)

### Story 14.4: Write Editor Configuration Docs

**Given** a developer wants to use Volra from Cursor/VS Code/Claude Code
**When** they read the configuration guide
**Then** they can set up the MCP server in their editor

**Acceptance Criteria:**
- Configuration snippets for: Cursor, VS Code (with MCP extension), Claude Code
- Each snippet shows the JSON/settings needed
- Tested with at least one editor

**Files:** `docs/mcp-integration.md` (NEW), `README.md` (UPDATE — link to MCP docs)

---

## Epic 15: Day 2 Operations (P2 — Week 5-6) — **DONE** (15.1, 15.2)

**Goal:** Operational polish and quality-of-life improvements for early adopters.
**Status:** Stories 15.1 (interactive quickstart) and 15.2 (--json flag) COMPLETE. Story 15.3 (Homebrew formula) DEFERRED — requires separate repository.

### Story 15.1: Interactive `volra quickstart`

**Given** a developer runs `volra quickstart` without arguments
**When** they are prompted to select a template
**Then** they can choose via interactive menu and specify a project name

**Acceptance Criteria:**
- Template selection menu with descriptions
- Project name prompt with validation (DNS-safe)
- Same output as non-interactive mode

**Files:** `cmd/quickstart.go` (UPDATE)

### Story 15.2: `--json` Output Flag

**Given** a developer runs any Volra command with `--json`
**When** the command completes
**Then** output is structured JSON instead of human-readable text

**Acceptance Criteria:**
- Global flag `--json` on all commands
- JSON schema matches human output fields
- Machine-parseable for CI/CD integration

**Files:** `cmd/root.go` (UPDATE), `internal/output/presenter.go` (UPDATE)

### Story 15.3: Homebrew Formula

**Given** a macOS developer
**When** they run `brew install volra`
**Then** the latest Volra binary is installed

**Acceptance Criteria:**
- Homebrew formula in `homebrew-volra` tap
- Auto-updated on new GitHub Release
- Works for macOS ARM64

**Files:** `homebrew-volra/` (NEW repo — Homebrew tap)

---

## Deferred: Unified Multi-Agent Dashboard

**Status:** DEFERRED to v0.3. Priya's journey (unified view of multiple agents) requires significant Grafana configuration work (federated Prometheus or shared instance). Will be planned when there are users with 3+ agents.

---

## v0.2 Epic Summary

| Epic | Priority | Effort | Stories | Track | Status |
|------|----------|--------|---------|-------|--------|
| 11: Launch Readiness | P0 | 2-3 days | 6 | A (Launch) | **DONE** |
| 12: Adoption Accelerators | P0 | 1 week | 5 | B (v0.2) | **DONE** |
| 13: Agent Observability Level 2 | P1 | 2-3 weeks | 6 | B (v0.2) | **DONE** |
| 14: MCP Integration | P1 | 1-2 weeks | 4 | B (v0.2) | **DONE** |
| 15: Day 2 Operations | P2 | 1 week | 3 (2 done) | B (v0.2) | **DONE** (15.3 deferred) |
| **Total** | | **~6 weeks** | **24** | | **v0.2 COMPLETE** |
