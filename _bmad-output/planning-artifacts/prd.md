---
stepsCompleted: [step-01-init, step-02-discovery, step-03-success, step-04-journeys, step-05-domain-skipped, step-06-innovation, step-07-project-type, step-08-scoping, step-09-functional, step-10-nonfunctional, step-11-polish]
inputDocuments:
  - '_bmad-output/planning-artifacts/product-brief-MegaCenter-2026-03-02.md'
  - '_bmad-output/brainstorming/brainstorming-session-2026-03-02.md'
  - 'guide.md'
workflowType: 'prd'
executionMode: 'GENERATE'
documentCounts:
  briefs: 1
  research: 0
  brainstorming: 1
  projectDocs: 0
classification:
  projectType: 'cli_tool + developer_platform'
  primaryUxSurface: 'grafana_dashboards'
  domain: 'general_devtools'
  projectContext: 'greenfield'
designConstraints:
  - 'All generated artifacts must be idempotent, readable, and functional without modification'
  - 'No plugins, no hooks. Override = edit generated files directly'
riskFlags:
  - 'Dockerfile generation is highest complexity/fragility feature — requires dedicated edge cases section'
---

# Product Requirements Document - MegaCenter

**Author:** Antonio
**Date:** 2026-03-02

## Success Criteria

### User Success

**Diego (Primary — Acquisition):**
- Completes first deploy (init → deploy → status healthy) without external help
- Time to first deploy < 15 minutes (given Docker installed + functional agent)
- Opens Grafana dashboard and sees agent metrics without any manual configuration
- Emotional signal: "This is what Heroku should have been for agents"

**Priya (Secondary — Retention):**
- Deploys 2nd and 3rd agent in < 5 minutes each using the same workflow
- All agents visible in a single unified dashboard
- Standardized health checks, metrics, and alerting across all agents regardless of how they were built
- Emotional signal: "Finally, I don't have to reinvent this for every agent"

**Success "aha!" moment:** Diego opens Grafana after `megacenter deploy` and sees — with zero configuration — his agent's health status, uptime, and probe latency. The dashboard is green, his agent is alive and monitored. The moment he realizes MegaCenter gave him production monitoring for free. (Real traffic metrics — request count, latency percentiles — come in v0.2 with the metrics gateway.)

### Business Success

**3 months post-launch:**
- Evidence of organic interest: at least 1 unprompted discussion on HN/Reddit
- At least 5 successful deploys by external users (outside close circle)
- Install → Deploy Conversion Rate measurable (target > 50%)

**12 months:**
- At least 5 users with agents running in production > 7 days
- At least 1 user with 3+ agents on MegaCenter
- At least 1 user willing to pay for support or hosted tier
- At least 1 meaningful external code contribution (not typo-fix)

**Go/No-Go gate:** If at 3 months < 10 people have tried the MVP outside close circle, pause and validate demand before building v0.2.

### Technical Success

- **CLI reliability:** > 95% of commands complete without unexpected errors
- **Deploy reproducibility:** Same Agentfile + code produces same result on Mac ARM, Ubuntu 24.04, Debian 12
- **Framework detection accuracy:** > 90% correct identification for LangGraph projects; generic fallback for everything else
- **Generated artifacts quality:** All generated files (Dockerfile, docker-compose.yml, prometheus.yml, Grafana dashboards) are idempotent, human-readable, and functional without modification for the standard case
- **Smoke test:** `megacenter init` → `deploy` → `status healthy` → metrics in Grafana → cleanup passes on all 3 target environments

### Measurable Outcomes

| Outcome | Metric | Target | How to measure |
|---------|--------|--------|----------------|
| Onboarding works | Install → Deploy Conversion | > 50% | Discord/GitHub: count starts vs. completions |
| Deploy is fast | TTFD (p80) | < 15 min | Beta tester self-reported via Google Form |
| Dashboard is useful | Daily dashboard opens by returning users | Qualitative: "yes, I check it" | Discord survey at Day 7 |
| Agents stay alive | Agents running > 7 days | At least 1 by month 3 | Direct ask in community |
| Expansion happens | Users with 2+ agents | At least 1 by month 3 | GitHub issues / Discord reports |

## Product Scope

*Summary view. For detailed MVP boundaries, cut list, and risk mitigation, see [Project Scoping & Phased Development](#project-scoping--phased-development).*

### MVP — Minimum Viable Product (v0.1, 6-8 weeks, 1 developer)

**4 CLI commands:**
- `megacenter doctor` — Environment diagnosis (10 checks)
- `megacenter init .` — Auto-detect agent, generate Agentfile
- `megacenter deploy` — Generate Docker stack, deploy with monitoring
- `megacenter status` — Report agent health

**Agentfile** (6 fields): version, name, framework, port, health_path, env, dockerfile mode

**Runtime** (Level 1 — universal): Docker Compose + Prometheus + Grafana (3 containers per project)

**2 Grafana dashboards**: Agent Overview, Agent Detail

**Design constraints:**
- Generates, doesn't abstract (all artifacts human-readable, no lock-in)
- No plugins, no hooks (override = edit generated files)
- Level 1 only (no framework introspection)

**Launch strategy:** Alpha at Week 4 (init + deploy), v0.1 at Week 8 (full), Launch Week at Week 8-9

### Growth Features (Post-MVP)

**v0.2 — Adoption Unlock:**
- `megacenter dev` (hot-reload), `megacenter quickstart` (scaffolding)
- Level 2: LangGraph-aware observability (traces, cost attribution)
- Second framework support (CrewAI or Mastra)
- 3 starter templates

**v0.3 — Operations Depth:**
- Multi-agent composition via MCP
- Audit trail, basic RBAC
- MCP Hub (service discovery)
- Advanced alerting (degradation detection)

### Vision (Future)

**v0.4+:**
- MCP Gateway (security policies), Cost Autopilot, Agent Marketplace, Plugin architecture

**v0.5+:**
- Custom control plane (replaces direct Prometheus/Alertmanager)
- Custom Console UI (replaces Grafana)
- Multi-tenancy, Federation, Kubernetes native

**2-3 year vision:** MegaCenter is the de facto standard for operating AI agents in production. Open-source, self-hosted, with an ecosystem of plugins, templates, and certified operators. Supabase model: everything free self-hosted, monetize hosted convenience + enterprise support.

## User Journeys

### Journey 1: Diego's First Deploy (Primary — Success Path)

**Narrative:**

Diego is a freelance backend developer in Buenos Aires. He built an AI agent for a client — a Python script using the OpenAI API to answer questions about product documentation. It works in his terminal. The client says: "Can you have it running 24/7 by Friday?"

Diego stares at his `main.py`. Running 24/7 means Docker, a server, health checks, monitoring, cost awareness. He's stressed — deadline is Friday, and he estimates a week of infrastructure work. He googles "how to deploy AI agent production" and finds MegaCenter on a Hacker News thread.

He installs, runs `megacenter doctor` (all green), runs `megacenter init .` (detects his agent, generates an Agentfile — he sees the CLI explain "Generated Agentfile (your agent config) — you can customize it later"). Runs `megacenter deploy`. 47 seconds later, everything is running.

He opens `localhost:3001`. No login screen (anonymous access enabled by default). Grafana loads with "Agent Overview" dashboard. Status: healthy. Uptime: 2 minutes. Probe latency: 45ms. His agent is alive and monitored.

He expected this to take a week. It took 12 minutes. The emotional contrast — from anxiety to relief — is what makes him tweet about it.

**Functional Steps:**

1. **`megacenter doctor`**
   - Input: none
   - System checks: Docker installed + running, Docker Compose V2, ports 9090/3001 free, Python ≥ 3.10, disk space
   - Output: pass/fail list with fix suggestions per check
   - Exit code: 0 (all pass) or 1 (failures found)

2. **`megacenter init .`**
   - Input: current directory path
   - System scans: requirements.txt, Python imports, app entry points
   - System detects: framework (generic/langgraph), port, health_path
   - System generates: `./Agentfile` (YAML, 6 fields)
   - Output: detection summary + "Generated Agentfile (your agent config) — you can customize it later"

3. **`megacenter deploy`**
   - System generates: `.megacenter/` directory containing Dockerfile, docker-compose.yml, prometheus.yml, grafana dashboards
   - System executes: `docker compose -f .megacenter/docker-compose.yml up -d`
   - System verifies: health check (HTTP 200 on configured health_path) with default timeout
   - Output (see Output Spec below)

4. **User opens `localhost:3001` (Grafana)**
   - Anonymous access enabled by default (no login screen)
   - "Agent Overview" dashboard auto-selected
   - Real-time metrics from Prometheus displayed

**Deploy Output Spec:**

```
$ megacenter deploy

  MegaCenter Deploy v0.1.0

  Generating artifacts...
    ✓ Dockerfile                → .megacenter/Dockerfile
    ✓ docker-compose.yml        → .megacenter/docker-compose.yml
    ✓ prometheus.yml            → .megacenter/prometheus.yml
    ✓ grafana dashboards        → .megacenter/grafana/

  Starting services...
    ✓ agent         (port 8000)    healthy
    ✓ prometheus    (port 9090)    running
    ✓ grafana       (port 3001)    running

  ──────────────────────────────────────────

  ✅ Deploy complete (34s)

  Your agent:     http://localhost:8000
  Dashboard:      http://localhost:3001
  Test it:        curl http://localhost:8000/health

  Stop services:  docker compose -f .megacenter/docker-compose.yml down
```

---

### Journey 2: Diego's Deploy Fails (Primary — Error Recovery)

**Narrative:**

Diego's second client has a RAG agent using `chromadb`. `megacenter init` works fine. `megacenter deploy` starts building, the container comes up, then exits immediately. MegaCenter detects exit code 137, reports OOM with the memory limit, and tells Diego exactly where to fix it and what command to retry. Diego opens the generated docker-compose.yml (readable YAML), increases the memory limit, runs `megacenter deploy` again. It works.

The key insight: MegaCenter didn't hide the problem. It gave a generic but actionable error message — no pretense of "smart" diagnosis, just facts and fix instructions.

**Error Catalog (v0.1):**

| Error | Cause | Output |
|-------|-------|--------|
| Docker not running | Daemon stopped | `❌ Docker is not running. Start Docker and try again.` |
| Port in use | Another service on required port | `❌ Port 3001 is in use (PID 1234: node). Free it or edit .megacenter/docker-compose.yml` |
| OOM killed | Container exceeds memory limit | `❌ Container killed: out of memory (512MB limit). Increase in .megacenter/docker-compose.yml → deploy.resources.limits.memory` |
| Health check timeout | Agent doesn't respond within 60s | `❌ Health check failed: no response on :8000/health after 60s. Check agent logs: docker logs megacenter-agent-1` |
| Build failure | pip install fails | `❌ Dockerfile build failed. See build logs above. Verify: pip install -r requirements.txt works locally` |

**Error pattern:** `❌ What happened. How to fix it.` Every error is actionable.

**Functional Steps:**

1. `megacenter deploy` executes normally through artifact generation
2. `docker compose up -d` starts containers
3. System monitors container status for 60s
4. Container exits with code 137 (OOM)
5. System interprets exit code → "out of memory"
6. System reads memory limit from docker-compose.yml → "512MB"
7. System outputs error with: what happened + current limit + where to change + retry command
8. User edits `.megacenter/docker-compose.yml` (human-readable, editable)
9. User runs `megacenter deploy` again (idempotent — detects existing `.megacenter/`, rebuilds)
10. Deploy succeeds

---

### Journey 3: Priya Standardizes Three Agents (Operations)

**Narrative:**

Priya is the most senior engineer at a 20-person startup. Three agents deployed three different ways — raw Docker, spare EC2, tmux session. The support chatbot went down for 6 hours unnoticed. The CEO found out from a customer. Priya got the blame.

She installs MegaCenter. Runs `megacenter init` + `megacenter deploy` in each of the three project directories. 40 minutes later: three separate Grafana instances, each showing its agent's health status. It's not a unified dashboard yet (that's v0.2), but each agent is monitored identically. She sends the CEO three screenshots. He replies: "This is exactly what I wanted to see."

**Functional Steps:**

1. For each of 3 agent directories: `megacenter init .` → Agentfile generated
2. For each: `megacenter deploy` → `.megacenter/` generated, independent stack started (3 containers each)
3. Each agent has its own Prometheus + Grafana instance
4. Each Grafana shows Overview dashboard: agent status, uptime, probe latency
5. All 3 agents monitored identically regardless of framework/implementation

**v0.1 limitation:** Each deploy creates an independent stack. Priya opens 3 Grafana tabs. Unified multi-agent dashboard and Alertmanager (Slack webhooks) are v0.2 features.

**Requirements revealed:**
- Agent naming: derived from project directory name, configurable in Agentfile
- Identical monitoring workflow regardless of agent implementation

---

### Journey 4: Diego Returns — Update and Redeploy

**Narrative:**

Two weeks later, Diego updates his agent's system prompt. He runs `megacenter deploy` in the same directory. MegaCenter detects `.megacenter/` exists → update mode. Rebuilds only the agent container, restarts services. 8 seconds of downtime. Prometheus metrics history preserved. Same command, same workflow, no new concepts.

**Functional Steps:**

1. User modifies agent code (e.g., `main.py`)
2. User runs `megacenter deploy`
3. System detects `.megacenter/` directory exists → update mode
4. System regenerates `.megacenter/Dockerfile` (in case dependencies changed)
5. System runs `docker compose -f .megacenter/docker-compose.yml up -d --build`
6. Only agent container rebuilds; Prometheus, Grafana untouched
7. System verifies health check on new container
8. Output: same format as initial deploy, with note "Updated existing deployment"
9. Prometheus data volumes persist across redeploy

**Key architectural requirement:** `.megacenter/` is disposable and regenerable. To migrate to a new server: copy project (with Agentfile) → run `megacenter deploy` → `.megacenter/` regenerated for new environment.

---

### Journey 5: Diego Cleans Up

**Narrative:**

Diego finishes a project. He wants to stop the agent and free resources. There is no `megacenter stop` command in v0.1 — MegaCenter generates standard Docker Compose files, and standard Docker Compose commands stop them.

The deploy output already told him: `Stop services: docker compose -f .megacenter/docker-compose.yml down`

He runs it. Services stop. To also remove metrics data: `docker compose -f .megacenter/docker-compose.yml down -v`

To fully clean up: `rm -rf .megacenter/`

**Requirements revealed:**
- Deploy output must include stop/cleanup commands
- `.megacenter/` can be safely deleted (no state that can't be regenerated)
- No dedicated cleanup command in v0.1 (workaround is documented standard Docker Compose)

---

### `--dry-run` Flag

`megacenter deploy --dry-run` generates the `.megacenter/` directory with all artifacts but does NOT execute `docker compose up`. Allows inspection of generated files before deployment.

- Same generation logic as regular deploy
- Output: lists generated files with paths
- Does not start any containers
- Useful for Priya (review before deploy) and for CI/CD pipelines

---

### Project Structure Convention

```
my-agent/
├── main.py              # agent code (user's)
├── requirements.txt     # dependencies (user's)
├── Agentfile            # agent config (generated by init, versioned with git)
├── .env                 # environment variable values (gitignored, user-created)
├── .env.example         # env var names (generated by init, versioned)
├── .megacenter/         # generated artifacts (gitignored, disposable)
│   ├── Dockerfile
│   ├── docker-compose.yml
│   ├── prometheus.yml
│   └── grafana/
│       ├── dashboards/
│       └── provisioning/
└── .gitignore           # includes .megacenter/ and .env
```

- **Agentfile** lives in project root — it's project config, versioned with git
- **`.megacenter/`** lives in project root — generated artifacts, gitignored, disposable and regenerable
- `megacenter init` generates Agentfile + adds `.megacenter/` to .gitignore
- `megacenter deploy` generates `.megacenter/` contents + executes

---

### Journeys Explicitly Not Covered (v0.1)

| Persona | Why excluded |
|---------|-------------|
| Sara (CTO) | Decision stakeholder, not product user. Needs materials (ROI docs), not features. |
| Klaus (Enterprise Architect) | Evaluator, not user. Needs compliance docs, not journeys. |
| Marco (Consultant) | Multi-tenancy required (v0.5+). Architecture won't block it, but no journey needed now. |

---

### Journey Requirements Summary

| Journey | Capabilities Revealed |
|---------|----------------------|
| Diego's First Deploy | doctor (env checks), init (auto-detect + Agentfile), deploy (full stack gen + execute), Grafana (zero-config, anonymous, probe metrics), deploy output spec |
| Diego's Deploy Fails | Error catalog (5 common errors), actionable error messages, editable artifacts, idempotent redeploy |
| Priya Standardizes | Independent stacks per agent (v0.1), identical monitoring workflow, unified dashboard deferred to v0.2 |
| Diego Returns | `.megacenter/` detection (always regenerate), selective rebuild (agent only), metrics persistence, `--dry-run` flag |
| Diego Cleans Up | Cleanup hints in deploy output, `.megacenter/` is disposable, no dedicated command needed |

**Cross-Journey Requirements:**
- `megacenter deploy` is idempotent: works for first deploy, update, and retry after failure
- All generated artifacts are human-readable, editable, and live in `.megacenter/`
- Error messages always follow: ❌ What happened. How to fix it.
- Grafana dashboards display probe-based metrics honestly labeled
- `.megacenter/` is disposable and regenerable — Agentfile is the portable config

### OSS Project Setup Requirements

*(Derived from contributor journey, not a product user journey)*

- `README.md` with installation from source
- `CONTRIBUTING.md` with PR rules (conventional commits, tests required)
- GitHub Issue templates (bug report, feature request)
- CI that runs tests on PRs automatically
- Clear module boundaries in codebase for easy contribution

## Innovation & Novel Patterns

### Innovation Type: Positional (v0.1), Technical (v0.2+)

MegaCenter v0.1 does not introduce novel technology. Every component exists: Docker Compose, Prometheus, Grafana, health checks, YAML configuration. The innovation is **positional** — assembling existing tools into an opinionated workflow that no one has packaged for the AI agent deployment use case.

**Why this gap exists:** Framework vendors (LangChain, CrewAI, Mastra) focus on agent *construction*, not agent *operations*. They can't offer free self-hosted ops tooling without cannibalizing their cloud revenue. Infrastructure tools (Docker, Kubernetes) are too generic — they require significant configuration for each agent. MegaCenter fills the structural gap between "I built an agent" and "my agent runs in production."

### Central Hypothesis

> "Developers abandon agent projects because of infrastructure friction, not framework limitations."

This hypothesis must be validated before writing code. If developers abandon for other reasons (model quality, cost, framework bugs), MegaCenter solves a problem that doesn't exist.

### Moat Analysis (Honest)

| Timeframe | Moat Type | Strength | Notes |
|-----------|-----------|----------|-------|
| v0.1 (launch) | Execution quality + first-mover | Weak | Anyone can replicate a Docker Compose generator in a weekend |
| v0.1 (6 months) | Community + defaults ecosystem | Moderate | Opinionated defaults are hard to replicate without user feedback loops |
| v0.2+ | Agent introspection engine | Strong | Framework-specific observability (LangGraph traces, cost attribution) requires deep integration work |
| v0.3+ | MCP Hub + multi-agent composition | Strong | Network effects from service discovery and agent interoperability |

**Honest assessment:** v0.1 has no technical moat. The bet is that execution quality, developer experience, and community-building during the first-mover window create enough momentum to reach v0.2, where the agent introspection engine becomes a genuine technical differentiator.

### Validation Plan (Pre-Code)

| # | Claim to Validate | Method | Timing | Success Signal |
|---|-------------------|--------|--------|----------------|
| 1 | Developers abandon because of infrastructure | 5 interviews with agent builders | Before writing code | 3/5 confirm infra is primary blocker |
| 2 | "15 minutes to deploy" is achievable | Prototype with 1 real agent project | Week 1 | Achievable with pre-installed Docker |
| 3 | Grafana dashboards provide enough value | Show mock dashboard to 3 developers | Week 1-2 | "I would use this" reaction |
| 4 | Developers trust generated Dockerfiles | Dry-run output review with 2 developers | Week 2-3 | No "I'd rather write my own" response |

**Priority:** Validate claim #1 first. If the central hypothesis fails, pivot or kill the project before investing 6-8 weeks of development.

### Risk: Innovation is Positioning, Not Technology

If a competitor (e.g., LangChain) ships a `langchain deploy` command that does 80% of what MegaCenter does, the positioning advantage evaporates. Fallback strategy:

1. **Reposition as framework-agnostic** — "Works with any agent, not just LangChain"
2. **Accelerate to v0.2** — Agent introspection engine becomes the real product
3. **Community moat** — If MegaCenter has active users and contributors, switching cost exists even without technical superiority

## CLI Tool + Developer Platform Requirements

### Deployment Model

**v0.1 is explicitly single-machine.** All agents, monitoring services, and dashboards run on `localhost`. There is no remote deployment, no SSH, no multi-machine orchestration.

**v0.1 is one stack per project directory.** Each `megacenter deploy` in a separate directory launches its own independent Prometheus + Grafana stack (3 containers: agent + prometheus + grafana). Priya with 3 agents opens 3 Grafana tabs (ports 3001, 3011, 3021). A unified multi-agent dashboard is a v0.2 feature.

**Docker requirement:** Docker Compose V2 (Compose Specification, no `version` field in generated files). Minimum Docker Engine version: current stable at time of v0.1 release. Checked by `megacenter doctor`.

### Command Structure

MegaCenter v0.1 exposes 4 commands with a flat, no-subcommand structure:

| Command | Purpose | Input | Output | Exit Code |
|---------|---------|-------|--------|-----------|
| `megacenter doctor` | Environment diagnosis | none | Pass/fail checklist with fix suggestions | 0 (all pass) / 1 (failures) |
| `megacenter init .` | Auto-detect agent, generate config | directory path | Detection summary + Agentfile | 0 (success) / 1 (detection failed) |
| `megacenter deploy` | Generate stack + deploy | none (reads Agentfile from cwd) | Artifact list + service status + URLs | 0 (healthy) / 1 (failed) |
| `megacenter status` | Report agent health | none | Health summary (agents + services) | 0 (all healthy) / 1 (unhealthy) |

**Agentfile lookup:** Current working directory only. No recursive search upward. If not found: `❌ No Agentfile found. Run megacenter init . first.`

**Global flags (v0.1):**
- `--dry-run` (deploy only): Generate artifacts without executing docker compose
- `--help`: Per-command usage (follows `gh`/`docker` pattern: description, usage, flags, one example)
- `--version`: Version info

**Design decisions:**
- No subcommands — flat namespace for 4 commands
- No interactive prompts — all input comes from Agentfile or flags
- No global config file — each project is self-contained via its Agentfile
- CLI must support standard POSIX flag parsing conventions
- Colors and emoji in output by default; respect `NO_COLOR` env var and `TERM=dumb` (fallback to ASCII: `[OK]`/`[FAIL]` instead of `✓`/`❌`)

### Output Specifications

All commands produce human-readable terminal output. No `--json` or `--yaml` output flags in v0.1.

**Output conventions:**
- `✓` for success steps, `❌` for errors, `⚠️` for warnings (two levels only)
- Structured sections with visual separators (`──────`)
- URLs always shown as clickable `http://localhost:PORT` links
- Timing information for deploy (`Deploy complete (34s)`)
- Error messages follow pattern: `❌ What happened. How to fix it.`
- Warning messages follow pattern: `⚠️ What happened. What was assumed. How to override.`

**`megacenter status` output spec:**

```
$ megacenter status

  MegaCenter Status

  Agent:
    ✓ my-agent         healthy    uptime 2d 4h    port 8000

  Services:
    ✓ prometheus    running    port 9090
    ✓ grafana       running    port 3001

  Dashboard: http://localhost:3001
```

**Scriptability (v0.1):**
- Exit codes are the primary machine-readable interface
- `--dry-run` enables CI/CD integration without requiring output parsing
- Future: `--json` flag for machine-readable output (v0.2)

### Configuration Schema (Agentfile)

```yaml
# Agentfile — generated by megacenter init, versioned with git
version: 1                   # schema version (for future compatibility)
name: my-agent               # derived from directory name, overridable
framework: generic            # generic | langgraph (detection result)
port: 8000                    # detected or default
health_path: /health          # detected or default
env:                          # environment variable NAMES required by the agent
  - OPENAI_API_KEY
dockerfile: auto              # auto | custom (use existing Dockerfile)
```

**6 fields total.** `version` included from day one for forward compatibility.

**`env` field semantics:** Lists environment variable *names* (not values). Values come from a `.env` file in the project root. `megacenter init` creates a `.env.example` listing the detected variables. `megacenter deploy` generates `docker-compose.yml` with `env_file: ../.env`. The `.env` file lives in the project root (next to Agentfile), NOT inside `.megacenter/`, ensuring `.megacenter/` remains fully disposable. `megacenter init` adds both `.megacenter/` and `.env` to `.gitignore`.

**Dockerfile mode:**
- `auto` (default): Generate Dockerfile from detection results
- `custom`: Use existing `./Dockerfile` in project root (user-managed)

No implicit `skip` mode. If a Dockerfile exists and Agentfile says `auto`, MegaCenter overwrites the generated one in `.megacenter/Dockerfile` (not the project root). If Agentfile says `custom`, MegaCenter uses the project root Dockerfile directly.

### Detection Logic (megacenter init)

Detection is **best-effort with safe defaults.** When detection fails partially, MegaCenter uses defaults and emits warnings — never fails silently, never blocks without reason.

**Detection sequence (priority order):**

1. **Project type:** Scan `requirements.txt` / `pyproject.toml` for framework markers (`langgraph`, `langchain`)
2. **Framework:** Scan Python imports for framework usage. If markers conflict (e.g., `requirements.txt` says langgraph but imports don't use it), `requirements.txt` wins.
3. **Entry point:** Look for `main.py`, `app.py`, `server.py` in order. If multiple found: use first match + emit `⚠️ Multiple entry points detected (main.py, app.py). Using main.py. Override in Agentfile.`
4. **Port:** Regex on entry point code (`uvicorn.run`, `app.run`, etc.). If not detected: `⚠️ Could not detect port. Using default: 8000. Override in Agentfile → port.`
5. **Health path:** Scan route decorators for `/health` or `/healthz`. If not detected: use `/health` default with warning.

**Precedence rule:** When sources conflict, file metadata (`requirements.txt`) beats code analysis (import scanning) beats defaults.

**Detection accuracy target:** > 90% correct **framework identification** for LangGraph projects. Port, health_path, and entry point detection are best-effort — defaults are the safety net, not accuracy targets.

### Installation

**v0.1:** Single binary distribution via curl:
```bash
curl -fsSL https://get.megacenter.dev | sh
```

- Detects OS (macOS/Linux) and architecture (ARM64/AMD64)
- Verifies SHA256 checksum before installing binary
- Default install location: `/usr/local/bin/megacenter`
- If no write permission: suggests `--prefix ~/.local/bin` and PATH instructions
- No runtime dependencies (Go compiles to static binary)
- Docker is a runtime dependency (checked by `megacenter doctor`), not an install dependency

**Future (v0.2+):** Homebrew, apt repository, GitHub Releases.

### Platform Support

MegaCenter v0.1 is tested on 3 platforms:

| Platform | Architecture | Tested in CI |
|----------|-------------|--------------|
| macOS | ARM64 (Apple Silicon) | Yes |
| Ubuntu 24.04 | AMD64 | Yes |
| Debian 12 | AMD64 | Yes |

Other platforms may work but are not tested or supported in v0.1.

### Generated Artifacts

MegaCenter generates, it does not abstract. All artifacts are human-readable, editable, idempotent, and disposable.

| Artifact | Location | Purpose |
|----------|----------|---------|
| Dockerfile | `.megacenter/Dockerfile` | Container image for the agent |
| docker-compose.yml | `.megacenter/docker-compose.yml` | Full stack (agent + monitoring) |
| prometheus.yml | `.megacenter/prometheus.yml` | Metrics scraping config + alert rules |
| Grafana dashboards | `.megacenter/grafana/dashboards/` | 2 JSON dashboard definitions (Overview + Detail) |
| Grafana provisioning | `.megacenter/grafana/provisioning/` | Auto-configured Prometheus datasource + default dashboard |

**Regeneration policy:** `megacenter deploy` ALWAYS regenerates everything in `.megacenter/`. No merge logic, no edit detection. If you edited a generated file and want to preserve the change: either (a) modify the Agentfile so generation produces what you want, or (b) switch to `dockerfile: custom` for the Dockerfile case.

### Dockerfile Generation (High-Risk Feature)

**Auto-generation strategy (v0.1):**
- Base image: `python:3.11-slim`
- Install: `pip install -r requirements.txt` (or `pip install .` for `pyproject.toml` projects)
- Entry point: detected from scan
- Healthcheck: `HEALTHCHECK CMD curl -f http://localhost:{port}{health_path}`
- No multi-stage builds in v0.1

**Image size trade-off:** Single-stage builds with `python:3.11-slim` can produce large images (1-2+ GB for projects with heavy dependencies like `chromadb`). This is a known trade-off — simplicity over optimization in v0.1. Multi-stage builds are a v0.2 candidate.

**Known edge cases:**
- `pyproject.toml` instead of `requirements.txt` → handled (use `pip install .`)
- System-level dependencies (e.g., build tools for `chromadb`) → not auto-detected, user edits Dockerfile via `custom` mode
- Existing Dockerfile in project → ignored unless `dockerfile: custom` in Agentfile

### Error Catalog (All Commands)

**`megacenter doctor`:**

| Error | Output |
|-------|--------|
| Docker not installed | `❌ Docker not installed. Install: https://docs.docker.com/get-docker/` |
| Docker not running | `❌ Docker is not running. Start Docker Desktop and try again.` |
| Port in use | `❌ Port 3001 in use (PID 1234: node). Free it or change port.` |

**`megacenter init`:**

| Error | Output |
|-------|--------|
| No Python project | `❌ No Python project detected. MegaCenter requires requirements.txt or pyproject.toml.` |
| No entry point | `❌ No entry point found. Create main.py or specify in Agentfile.` |
| Multiple entry points | `⚠️ Multiple entry points detected (main.py, app.py). Using main.py. Override in Agentfile.` |
| Partial port detection | `⚠️ Could not detect port. Using default: 8000. Override in Agentfile → port.` |

**`megacenter deploy`:** (see Journey 2 error catalog: OOM, health check timeout, build failure, port in use, Docker not running)

**`megacenter status`:**

| Error | Output |
|-------|--------|
| No deployment | `❌ No deployment found. Run megacenter deploy first.` |
| Docker not running | `❌ Docker is not running. Start Docker and try again.` |
| Agent unhealthy | `⚠️ Agent unhealthy: no response on :8000/health. Check logs: docker logs megacenter-agent-1` |
| Docker daemon restarted | `❌ Docker daemon restarted. Your agents stopped. Run megacenter deploy to restart.` |

### Deferred to Architecture Document

The following technical decisions are out of scope for the PRD and will be resolved in the Architecture document:

- Internal Go package structure and CLI framework selection
- Testing strategy (unit, integration, golden file approach)
- Prometheus scraping topology for multi-agent scenarios (v0.2)
- Docker Compose generation internals (templating approach)
- CI/CD pipeline design for MegaCenter itself

## Project Scoping & Phased Development

### MVP Strategy & Philosophy

**MVP Approach:** Problem-solving MVP — prove that the core workflow (init → deploy → dashboard) solves a real problem (infrastructure friction) for a specific user (Diego, backend dev deploying his first agent).

**Resource Reality:** 1 developer, 6-8 weeks. Every scope decision passes the filter: "Can one person build, test, and ship this in 8 weeks?"

**Central bet:** Developers will use MegaCenter because the alternative (manual Docker + Prometheus + Grafana setup) takes a week. If the alternative only takes an hour for a competent dev, MegaCenter's value proposition collapses.

**Measurement definitions:**
- **Install** = curl install script completes successfully (exit code 0)
- **Deploy** = `megacenter deploy` completes with health check passing (exit code 0)
- **Install → Deploy Conversion Rate** = Deploy count / Install count (target > 50%)

### MVP Feature Set (Phase 1 — v0.1)

**4 commands, 1 workflow, 1 persona, 3 containers per project:**

| Feature | MVP Scope | Explicitly NOT in MVP |
|---------|-----------|----------------------|
| `megacenter doctor` | Environment checks with fix suggestions | No auto-fix, no plugin checks |
| `megacenter init` | Python project detection, Agentfile generation | No interactive wizard, no non-Python support |
| `megacenter deploy` | Full stack generation + docker compose up | No remote deploy, no K8s, no hot-reload |
| `megacenter status` | Health check + service status | No historical metrics, no alerting config |
| Agentfile | 6 fields, generated not hand-written | No conditional logic, no includes |
| Dockerfile | auto + custom modes | No multi-stage, no multi-language |
| Monitoring | Prometheus (with built-in alert rules) + Grafana | No Alertmanager, no custom metrics, no tracing |
| Dashboards | 2 Grafana dashboards (Overview + Detail) | No Alerts dashboard, no custom dashboards |
| `--dry-run` | Generate without executing | No diff against previous generation |
| Installation | `curl \| sh` with SHA256 verification | No brew, no apt, no Windows |
| Platforms | macOS ARM64, Ubuntu 24.04, Debian 12 | No Windows, no ARM Linux |
| Frameworks | Generic (any HTTP) + LangGraph detection | No CrewAI, no Mastra, no JS agents |

**Alertmanager decision:** Cut from v0.1. Agent-down alerting is handled by Prometheus alert rules + a visual status indicator (red/green) on the Overview dashboard. Diego sees red in Grafana if his agent is down. Alertmanager (with Slack/email webhooks) is a v0.2 feature for Priya's workflow. This reduces containers from 4 to 3, eliminates `alertmanager.yml` generation, and simplifies the artifact set.

**Dashboard specifications:**

| Dashboard | Content | Primary User |
|-----------|---------|-------------|
| **Overview** | Agent status badge (healthy/unhealthy with visual red/green indicator), uptime (continuous), last seen (timestamp), current probe latency. All probe-based — clearly labeled as such. | Diego (quick glance) |
| **Detail** | Time-series panels: probe latency over time (line chart), health status timeline (green/red state chart), uptime percentage (24h/7d gauges). If agent exposes own `/metrics`, those appear as bonus panels. | Diego (debugging) |

**Metrics honesty:** Dashboards label metrics as what they are — "Probe Latency" not "Request Latency", "Health Checks" not "Request Count". Real user traffic metrics require agent-side instrumentation or a metrics gateway (v0.2).

**Grafana first-open experience:** Grafana opens directly to the Agent Overview dashboard — no navigation required. Configured via Grafana provisioning (default dashboard setting). Anonymous access enabled (no login screen).

**Core journey supported:** Diego's First Deploy (Journey 1) — the only journey that must work flawlessly.

**Journeys partially supported:** Error Recovery (Journey 2), Update & Redeploy (Journey 4), Cleanup (Journey 5).

**Journey explicitly NOT supported in v0.1:** Priya's unified multi-agent dashboard (Journey 3) — each deploy creates an independent stack. Priya can use MegaCenter but gets separate Grafana instances, not a unified view.

### Alpha Milestone (Week 4)

**Alpha = doctor + init + deploy.** Doctor is non-negotiable — it's the prerequisite check that prevents cryptic Docker errors. Status is deferred to weeks 5-8.

**Alpha success criteria:** Diego can run `megacenter doctor` → `megacenter init .` → `megacenter deploy` → open Grafana → see metrics. On macOS ARM64 only.

### Post-MVP Features

**Phase 2a — v0.2a (UX, ~2 months post-launch):**
- Unified multi-agent Grafana dashboard (Priya's journey)
- Alertmanager + Alerts dashboard + Slack/email webhooks
- Homebrew + apt installation
- `--json` output flag
- `megacenter dev` (hot-reload for development)

**Phase 2b — v0.2b (Depth, ~4 months post-launch):**
- Level 2: LangGraph-aware observability (traces, cost attribution)
- Second framework support (CrewAI or Mastra)
- 3 starter templates
- `megacenter quickstart` (scaffold starter projects)

**v0.2 migration consideration:** v0.1 uses independent stacks per directory. v0.2a introduces shared infrastructure (unified Grafana). This is an architectural change — users may need to re-deploy agents to migrate to shared infrastructure. Architecture doc must design v0.1 with this migration path in mind.

**Phase 3 — v0.3 (Operations Depth, ~6 months post-launch):**
- Multi-agent composition via MCP
- Audit trail, basic RBAC
- MCP Hub (service discovery)
- Advanced alerting (degradation detection)

**Phase 4 — v0.4+ (Vision):**
- MCP Gateway, Cost Autopilot, Agent Marketplace
- Custom control plane (replaces Prometheus)
- Custom Console UI (replaces Grafana)
- Multi-tenancy, Federation, Kubernetes native

### Scope Management Rules

**Scope (commands, features) is zero-sum:** No new command is added without removing one or explicit justification. The Agentfile has 6 fields — adding a 7th requires removing one or explicit justification.

**Polish (flags, error messages, UX) is free:** Adding `--verbose` for debugging, improving an error message, or handling a new edge case in detection is developer experience, not scope creep.

**Cut list (ordered, if time runs out):**

| Priority | Feature | Criterion | Rationale |
|----------|---------|-----------|-----------|
| Cut first | `--dry-run` | Diego doesn't need it for Journey 1 | CI/CD feature, not onboarding |
| Cut second | `megacenter status` | Diego can open Grafana instead | Nice-to-have, Grafana covers the need |
| Cut third | SHA256 verification | Security, not functionality | Can be added in patch release |
| Cut fourth | Error catalog (init/status) | Deploy errors are essential, others are nice-to-have | Core 5 deploy errors stay |
| Cut fifth | Detail dashboard | Overview is sufficient for MVP | Single dashboard still delivers value |
| **Never cut** | `doctor` | Prerequisite for Journey 1 | Without it, Docker errors are incomprehensible |
| **Never cut** | `init` | Core of Journey 1 | No Agentfile = no MegaCenter |
| **Never cut** | `deploy` | Core of Journey 1 | The product IS deploy |
| **Never cut** | Overview dashboard | The "aha moment" | Diego sees metrics = MegaCenter delivers |

### Risk Mitigation Strategy

**Technical Risks:**

| Risk | Impact | Mitigation |
|------|--------|------------|
| Dockerfile generation fails for edge cases | High | `dockerfile: custom` escape hatch. Error messages point to manual Dockerfile. |
| Detection accuracy < 90% for LangGraph | Medium | Generic fallback always works. Defaults are the safety net. |
| Docker Compose version incompatibilities | High | `megacenter doctor` checks version. Require Compose V2. |
| Large image sizes without multi-stage | Low | Documented trade-off. Functional but not optimized. v0.2 improvement. |
| 3 containers per project = resource pressure | Medium | Documented: ~500MB RAM per stack. Laptop with 16GB supports 3-4 concurrent projects. |

**Market Risks:**

| Risk | Impact | Mitigation |
|------|--------|------------|
| Central hypothesis is wrong | Fatal | 5 pre-code interviews. Go/No-Go gate at 3 months. |
| LangChain ships `langchain deploy` | High | Reposition as framework-agnostic. Accelerate to v0.2. |
| < 10 external users at 3 months | High | Pause and validate demand. Don't build v0.2 on hope. |

**Resource Risks:**

| Risk | Impact | Mitigation |
|------|--------|------------|
| 8 weeks not enough | Medium | Alpha at week 4. If alpha slips, apply cut list. |
| Solo developer burnout | Medium | Progressive launch. Week 4 alpha = early feedback loop. |
| Scope creep | High | Zero-sum rule for commands/features. Cut list pre-defined. |

## Functional Requirements

*This is the capability contract for MegaCenter v0.1. UX design, architecture, and epic breakdown will ONLY address capabilities listed here. Capabilities not listed will NOT exist in the product.*

*FRs marked [SHOULD] are aligned with the cut list — they may be deferred if time runs out. All unmarked FRs are MUST.*

### Environment Diagnosis

- **FR1:** Developer can run a pre-flight check that validates all prerequisites for MegaCenter to function: Docker installed, Docker running, Docker Compose V2 available, required ports free, Python ≥ 3.10 present, sufficient disk space, and MegaCenter version reported
- **FR2:** Developer can see a pass/fail result for each individual check with a specific fix suggestion for each failure
- **FR3:** Developer can determine from the command exit code whether all checks passed (0) or any failed (1)

### Project Detection & Configuration

- **FR4:** Developer can point MegaCenter at a Python project directory and receive an auto-generated Agentfile with a summary of detected values and instructions to customize
- **FR5:** System can detect the agent framework used (generic or LangGraph) by scanning dependency files and Python imports
- **FR6:** System can detect the application entry point by scanning for common filenames (main.py, app.py, server.py) in priority order
- **FR7:** System can detect the application port by scanning entry point code for server startup patterns
- **FR8:** System can detect health check endpoints by scanning route decorators
- **FR9:** System can detect environment variable references in agent code (os.environ, os.getenv patterns) to populate `.env.example`
- **FR10:** When detection is partial or ambiguous, system uses safe defaults and emits a warning with override instructions referencing the specific Agentfile field
- **FR11:** Developer can override any detected value by editing the generated Agentfile
- **FR12:** System generates a `.env.example` file listing detected environment variable names
- **FR13:** System adds `.megacenter/` and `.env` to the project's `.gitignore`
- **FR14:** Generated Agentfile includes a schema version field for forward compatibility
- **FR15:** When Agentfile already exists, system exits with error and suggests `--force` flag to overwrite

### Stack Generation & Deployment

- **FR16:** Developer can generate a complete deployment stack (Dockerfile, docker-compose.yml, Prometheus config, Grafana dashboards) from an Agentfile
- **FR17 [SHOULD]:** Developer can generate the deployment stack without executing it (dry-run mode)
- **FR18:** System generates a Dockerfile using auto-detection results, or uses an existing project Dockerfile when configured as custom mode
- **FR19:** System generates a docker-compose.yml that orchestrates 3 containers: agent, Prometheus, and Grafana
- **FR20:** System generates a Prometheus configuration that scrapes the agent's health endpoint directly, with an alert rule for agent-down detection (`up == 0` for configured duration)
- **FR21:** System generates Grafana dashboard definitions and provisioning configuration, including setting the Overview dashboard as the default landing page
- **FR22:** System executes docker compose to start all services after artifact generation
- **FR23:** System verifies agent health by polling the configured health endpoint with a default timeout
- **FR24:** Developer can see a structured deploy output listing: generated artifacts, service status with ports, summary URLs (agent + dashboard), and the exact command to stop services
- **FR25:** Developer can update a deployed agent by re-running deploy, which rebuilds the agent container without losing monitoring history
- **FR26:** System detects and reports common deployment failures with actionable error messages (Docker not running, port in use, OOM killed, health check timeout, build failure)

### Health Monitoring & Status

- **FR27 [SHOULD]:** Developer can check the current health status of a deployed agent and its supporting services
- **FR28 [SHOULD]:** Developer can see agent health state (healthy/unhealthy), uptime, and port assignment
- **FR29 [SHOULD]:** Developer can see the running state and port of each supporting service (Prometheus, Grafana)
- **FR30:** System detects when Docker daemon has restarted and reports that agents need redeployment

### Observability — Probe Metrics (Guaranteed)

- **FR31:** System provides health probe metrics (probe success/failure, probe latency) by configuring Prometheus to scrape the agent's health endpoint directly, without requiring agent-side Prometheus instrumentation
- **FR32:** Developer can view an Overview dashboard showing: agent status badge (healthy/unhealthy with visual red/green indicator), uptime (continuous), and current probe latency
- **FR33:** Developer can view a Detail dashboard showing: probe latency over time (line chart), health status timeline (green/red state chart), and uptime percentage (24h/7d)
- **FR34:** Overview dashboard displays a visual alert indicator (red status) when the agent's health check is failing

### Observability — Agent-Provided Metrics (Bonus)

- **FR35:** If the agent exposes Prometheus-format metrics on a `/metrics` endpoint, system automatically collects and displays them in dashboards alongside probe metrics, without additional configuration

### Observability — Dashboard UX

- **FR36:** Developer can open Grafana without authentication (anonymous access) and land directly on the Overview dashboard without navigation
- **FR37:** Dashboards clearly label all metrics as probe-based (e.g., "Probe Latency", "Health Checks") to distinguish from real user traffic metrics (available in v0.2)

### Installation & Distribution

- **FR38:** Developer can install MegaCenter via a single shell command that auto-detects OS and architecture
- **FR39 [SHOULD]:** Installation script verifies binary integrity via SHA256 checksum before placing the binary
- **FR40:** When the install location requires elevated permissions, system suggests an alternative user-local path with PATH configuration instructions
- **FR41:** Developer can check the installed MegaCenter version

## Non-Functional Requirements

### Performance

- **NFR1:** CLI commands (doctor, init, status) complete in < 5 seconds on target platforms
- **NFR2:** `megacenter deploy` first run (cold, including Docker base image pull) completes in < 5 minutes for a typical Python agent with < 20 pip dependencies
- **NFR3:** `megacenter deploy` subsequent runs (warm, base image cached) complete in < 3 minutes; re-deploy with no dependency changes completes in < 30 seconds
- **NFR4:** Grafana dashboards load and display metrics within 5 seconds of opening the browser
- **NFR5:** Prometheus scrapes agent health endpoint every 15 seconds (default interval) with a probe timeout of 10 seconds — health endpoint that does not respond within 10s counts as unhealthy (no "degraded" state in v0.1)
- **NFR6:** Total time-to-first-deploy (install + doctor + init + deploy) < 15 minutes — preconditions: Docker pre-installed and running, Python agent that starts an HTTP server, `requirements.txt` or `pyproject.toml` present

### Reliability

- **NFR7:** CLI commands complete without unexpected errors > 95% of the time on target platforms
- **NFR8:** Same Agentfile + same agent code produces functionally equivalent deployments across macOS ARM64, Ubuntu 24.04, and Debian 12: same container topology, same Prometheus config, same Grafana dashboards, same network behavior
- **NFR9:** Prometheus data volumes survive agent container rebuilds (monitoring history preserved across redeploys)
- **NFR10:** Generated artifacts (Dockerfile, docker-compose.yml, prometheus.yml, Grafana dashboards) are valid and functional without manual modification for standard Python HTTP agents
- **NFR11:** Health check probe detects agent-down state within 1 minute (2 consecutive failed probes at 15s interval + 30s alert rule evaluation)

### Compatibility

- **NFR12:** MegaCenter binary runs on macOS ARM64, Ubuntu 24.04 AMD64, and Debian 12 AMD64 without additional dependencies
- **NFR13:** Generated Docker Compose files are compatible with Docker Compose V2 (Compose Specification)
- **NFR14:** Generated Dockerfiles produce functional images on both AMD64 and ARM64 Docker hosts
- **NFR15:** CLI output renders correctly in terminals with and without emoji/Unicode support (respects `NO_COLOR` and `TERM=dumb`) and in terminals with minimum 60 columns width

### Security

- **NFR16:** Environment variable values (API keys, secrets) never appear in generated artifacts inside `.megacenter/` — only variable names are referenced, values come from `.env` in project root
- **NFR17:** `.env` file is automatically added to `.gitignore` to prevent accidental commit of secrets
- **NFR18 [SHOULD]:** Install script verifies binary integrity via SHA256 checksum before execution
- **NFR19:** Generated Grafana instance uses anonymous access (no credentials to manage) — acceptable for localhost-only v0.1 deployment
- **NFR20:** No telemetry, no phone-home, no data collection of any kind in v0.1

### Resource Efficiency

- **NFR21:** MegaCenter monitoring stack (Prometheus + Grafana, excluding agent container) uses < 500MB RAM at rest

### Developer Experience

- **NFR22:** Every user-facing error message follows the pattern: what happened + actionable fix instruction. No error exits without guidance
- **NFR23:** Every warning message follows the pattern: what happened + what was assumed + how to override
