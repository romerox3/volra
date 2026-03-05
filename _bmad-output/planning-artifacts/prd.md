---
stepsCompleted: [step-01-init, step-02-discovery, step-03-success, step-04-journeys, step-05-domain-skipped, step-06-innovation, step-07-project-type, step-08-scoping, step-09-functional, step-10-nonfunctional, step-11-polish, step-e-strategic-pivot-2026-03-04]
inputDocuments:
  - '_bmad-output/planning-artifacts/product-brief-Volra-2026-03-02.md'
  - '_bmad-output/brainstorming/brainstorming-session-2026-03-02.md'
  - '_bmad-output/brainstorming/market-research-2026-03-04.md'
  - '_bmad-output/planning-artifacts/sprint-change-proposal-2026-03-04.md'
  - 'guide.md'
workflowType: 'prd'
executionMode: 'REFINE'
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

# Product Requirements Document - Volra

**Author:** Antonio
**Date:** 2026-03-02
**Last Updated:** 2026-03-04 (post-market-research strategic pivot)

## Executive Summary

**Volra** is an open-source CLI tool that gives developers production infrastructure for AI agents — on their own servers.

**Tagline:** Own your agent infrastructure.

**Subtitle:** Deploy, monitor, and operate AI agents on your own servers. CLI-first, open-source, framework-agnostic.

**What it does:** Volra generates a complete deployment stack (Docker Compose + Prometheus + Grafana) from a declarative Agentfile, then deploys and monitors the agent — all on localhost or any server the developer controls.

**What it doesn't do:** Volra does not build agents, host agents as SaaS, or require vendor accounts. It generates infrastructure artifacts that the developer owns, reads, and modifies.

**Competitive positioning (March 2026):**
- Railway ($100M, SaaS) and Vercel (SaaS) own the "deploy your agent" cloud space
- Aegra (OSS, Apache 2.0) targets LangGraph users with a self-hosted alternative
- Volra's unique combination: **self-hosted + framework-agnostic + CLI-first + open-source** does not exist in any competitor

**Target user:** Backend developers deploying Python AI agents who need production infrastructure without SaaS dependency.

## Success Criteria

### User Success

**Diego (Primary — Acquisition):**
- Completes first deploy (init → deploy → status healthy) without external help
- Time to first deploy < 15 minutes (given Docker installed + functional agent)
- Opens Grafana dashboard and sees agent metrics without any manual configuration
- Emotional signal: "I own everything — the monitoring, the config, the data — and it took 12 minutes instead of a week"

**Priya (Secondary — Retention):**
- Deploys 2nd and 3rd agent in < 5 minutes each using the same workflow
- All agents visible in a single unified dashboard
- Standardized health checks, metrics, and alerting across all agents regardless of how they were built
- Emotional signal: "Finally, I don't have to reinvent this for every agent"

**Success "aha!" moment:** Diego opens Grafana after `volra deploy` and sees — with zero configuration — his agent's health status, uptime, and probe latency. The dashboard is green, his agent is alive and monitored. Everything runs on his machine — no third-party accounts, no API keys shared with cloud services. The moment he realizes he owns the entire production stack and it took 12 minutes instead of a week. (Real traffic metrics — request count, latency percentiles — come in v0.2 with Level 2 observability.)

### Business Success

**3 months post-launch:**
- Evidence of organic interest: at least 1 unprompted discussion on HN/Reddit
- At least 5 successful deploys by external users (outside close circle)
- Install → Deploy Conversion Rate measurable (target > 50%)

**12 months:**
- At least 5 users with agents running in production > 7 days
- At least 1 user with 3+ agents on Volra
- At least 1 user willing to pay for support or hosted tier
- At least 1 meaningful external code contribution (not typo-fix)

**Go/No-Go gate:** If at 3 months < 10 people have tried the MVP outside close circle, pause and validate demand before building v0.2.

### Competitive Benchmarks (Post-Market-Research)

| Metric | Target | Timeframe | Why |
|--------|--------|-----------|-----|
| First non-team deploy | 1 successful | 2 weeks post-launch | Validates install + docs + DX |
| GitHub unique cloners | 100 | 3 months | Real adoption signal (not vanity stars) |
| Aegra feature parity exceeded | Level 2 shipped | 3 months | Must exceed direct competitor |
| MCP Server shipped | v0.2 released | 3 months | Competitive parity with Railway |
| First framework contribution | 1 merged PR | 6 months | Community health signal |
| Agents running > 7 days | 5 external users | 6 months | Production adoption proof |

### Technical Success

- **CLI reliability:** > 95% of commands complete without unexpected errors
- **Deploy reproducibility:** Same Agentfile + code produces same result on Mac ARM, Ubuntu 24.04, Debian 12
- **Framework detection accuracy:** > 90% correct identification for LangGraph projects; generic fallback for everything else
- **Generated artifacts quality:** All generated files (Dockerfile, docker-compose.yml, prometheus.yml, Grafana dashboards) are idempotent, human-readable, and functional without modification for the standard case
- **Smoke test:** `volra init` → `deploy` → `status healthy` → metrics in Grafana → cleanup passes on all 3 target environments

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

### MVP — v0.1 (BUILT — ready for early access launch)

**Status:** Code complete. 8,166 lines Go, ~60 files, 297+ unit tests, 33 E2E tests validated with 8 real Python agents. All 7 v2.1 limitations resolved.

**4 CLI commands:**
- `volra doctor` — Environment diagnosis (10 checks)
- `volra init .` — Auto-detect agent, generate Agentfile
- `volra deploy` — Generate Docker stack, deploy with monitoring
- `volra status` — Report agent health

**Agentfile v2.1** (~15 fields): version, name, framework, port, health_path, env, dockerfile, services, security, gpu, health_timeout, volumes, host_port, build

**Runtime** (Level 1 — universal): Docker Compose + Prometheus + Grafana (3+ containers per project)

**4 Grafana dashboard variants**: Agent Overview, Agent Detail, Agent Overview (LangGraph), Agent Detail (LangGraph)

**Design constraints:**
- Generates, doesn't abstract (all artifacts human-readable, no lock-in)
- No plugins, no hooks (override = edit generated files)
- Level 1 only (no framework introspection — Level 2 in v0.2)

**Launch strategy:** Early access launch this week (March 2026). See [Launch Strategy](#launch-strategy).

### Growth Features (Post-MVP)

**v0.2 — Differentiation & Developer Experience:**
- `volra quickstart` (scaffold from templates) + 3 starter templates (basic, rag, conversational)
- `volra logs` (streaming log access for deployed agents)
- Level 2: Framework-agnostic agent observability (token counting, cost tracking, LLM latency) via `volra-observe` Python package
- Volra MCP Server (deploy and manage agents from MCP-compatible editors)
- Basic LLM cost tracking dashboard panels

**v0.3 — Composition & Framework Depth:**
- Level 2 addons: Optional framework-specific extensions (LangGraph graph state, CrewAI crew metrics) + auto-injected instrumentation
- Multi-agent composition via MCP (agent-to-agent communication)
- Second framework support (demand-driven: CrewAI or Mastra)
- Basic audit trail (append-only execution log)
- EU AI Act compliance documentation

### Vision (Future)

**v0.4:**
- `volra dev` (hot-reload local development)
- RBAC, MCP Hub/Gateway, Advanced alerting
- Agent Marketplace (community templates)

**v0.5+:**
- Custom control plane (replaces direct Prometheus/Alertmanager)
- Custom Console UI (replaces Grafana)
- Multi-tenancy, Federation, Kubernetes native
- MCP Gateway (security policies), Cost Autopilot

**2-3 year vision:** Volra is the de facto standard for self-hosted AI agent operations. Open-source, framework-agnostic, with an ecosystem of templates, Level 2+ framework integrations, and MCP composition capabilities. Supabase model: everything free self-hosted, monetize hosted convenience + enterprise support. The competitive positioning is "own your infrastructure" — serving the segment that Railway/Vercel cannot enter (self-hosted, data sovereignty, compliance).

## User Journeys

### Journey 1: Diego's First Deploy (Primary — Success Path)

**Narrative:**

Diego is a freelance backend developer in Buenos Aires. He built an AI agent for a client — a Python script using the OpenAI API to answer questions about product documentation. It works in his terminal. The client says: "Can you have it running 24/7 by Friday?"

Diego stares at his `main.py`. Running 24/7 means Docker, a server, health checks, monitoring, cost awareness. He's stressed — deadline is Friday, and he estimates a week of infrastructure work. He googles "how to deploy AI agent production" and finds Volra on a Hacker News thread.

He installs, runs `volra doctor` (all green), runs `volra init .` (detects his agent, generates an Agentfile — he sees the CLI explain "Generated Agentfile (your agent config) — you can customize it later"). Runs `volra deploy`. 47 seconds later, everything is running.

He opens `localhost:3001`. No login screen (anonymous access enabled by default). Grafana loads with "Agent Overview" dashboard. Status: healthy. Uptime: 2 minutes. Probe latency: 45ms. His agent is alive and monitored.

He expected this to take a week. It took 12 minutes. The emotional contrast — from anxiety to relief — is what makes him tweet about it.

**Functional Steps:**

1. **`volra doctor`**
   - Input: none
   - System checks: Docker installed + running, Docker Compose V2, ports 9090/3001 free, Python ≥ 3.10, disk space
   - Output: pass/fail list with fix suggestions per check
   - Exit code: 0 (all pass) or 1 (failures found)

2. **`volra init .`**
   - Input: current directory path
   - System scans: requirements.txt, Python imports, app entry points
   - System detects: framework (generic/langgraph), port, health_path
   - System generates: `./Agentfile` (YAML, 6 fields)
   - Output: detection summary + "Generated Agentfile (your agent config) — you can customize it later"

3. **`volra deploy`**
   - System generates: `.volra/` directory containing Dockerfile, docker-compose.yml, prometheus.yml, grafana dashboards
   - System executes: `docker compose -f .volra/docker-compose.yml up -d`
   - System verifies: health check (HTTP 200 on configured health_path) with default timeout
   - Output (see Output Spec below)

4. **User opens `localhost:3001` (Grafana)**
   - Anonymous access enabled by default (no login screen)
   - "Agent Overview" dashboard auto-selected
   - Real-time metrics from Prometheus displayed

**Deploy Output Spec:**

```
$ volra deploy

  Volra Deploy v0.1.0

  Generating artifacts...
    ✓ Dockerfile                → .volra/Dockerfile
    ✓ docker-compose.yml        → .volra/docker-compose.yml
    ✓ prometheus.yml            → .volra/prometheus.yml
    ✓ grafana dashboards        → .volra/grafana/

  Starting services...
    ✓ agent         (port 8000)    healthy
    ✓ prometheus    (port 9090)    running
    ✓ grafana       (port 3001)    running

  ──────────────────────────────────────────

  ✅ Deploy complete (34s)

  Your agent:     http://localhost:8000
  Dashboard:      http://localhost:3001
  Test it:        curl http://localhost:8000/health

  Stop services:  docker compose -f .volra/docker-compose.yml down
```

---

### Journey 2: Diego's Deploy Fails (Primary — Error Recovery)

**Narrative:**

Diego's second client has a RAG agent using `chromadb`. `volra init` works fine. `volra deploy` starts building, the container comes up, then exits immediately. Volra detects exit code 137, reports OOM with the memory limit, and tells Diego exactly where to fix it and what command to retry. Diego opens the generated docker-compose.yml (readable YAML), increases the memory limit, runs `volra deploy` again. It works.

The key insight: Volra didn't hide the problem. It gave a generic but actionable error message — no pretense of "smart" diagnosis, just facts and fix instructions.

**Error Catalog (v0.1):**

| Error | Cause | Output |
|-------|-------|--------|
| Docker not running | Daemon stopped | `❌ Docker is not running. Start Docker and try again.` |
| Port in use | Another service on required port | `❌ Port 3001 is in use (PID 1234: node). Free it or edit .volra/docker-compose.yml` |
| OOM killed | Container exceeds memory limit | `❌ Container killed: out of memory (512MB limit). Increase in .volra/docker-compose.yml → deploy.resources.limits.memory` |
| Health check timeout | Agent doesn't respond within 60s | `❌ Health check failed: no response on :8000/health after 60s. Check agent logs: docker logs volra-agent-1` |
| Build failure | pip install fails | `❌ Dockerfile build failed. See build logs above. Verify: pip install -r requirements.txt works locally` |

**Error pattern:** `❌ What happened. How to fix it.` Every error is actionable.

**Functional Steps:**

1. `volra deploy` executes normally through artifact generation
2. `docker compose up -d` starts containers
3. System monitors container status for 60s
4. Container exits with code 137 (OOM)
5. System interprets exit code → "out of memory"
6. System reads memory limit from docker-compose.yml → "512MB"
7. System outputs error with: what happened + current limit + where to change + retry command
8. User edits `.volra/docker-compose.yml` (human-readable, editable)
9. User runs `volra deploy` again (idempotent — detects existing `.volra/`, rebuilds)
10. Deploy succeeds

---

### Journey 3: Priya Standardizes Three Agents (Operations)

**Narrative:**

Priya is the most senior engineer at a 20-person startup. Three agents deployed three different ways — raw Docker, spare EC2, tmux session. The support chatbot went down for 6 hours unnoticed. The CEO found out from a customer. Priya got the blame.

She installs Volra. Runs `volra init` + `volra deploy` in each of the three project directories. 40 minutes later: three separate Grafana instances, each showing its agent's health status. It's not a unified dashboard yet (that's v0.2), but each agent is monitored identically. She sends the CEO three screenshots. He replies: "This is exactly what I wanted to see."

**Functional Steps:**

1. For each of 3 agent directories: `volra init .` → Agentfile generated
2. For each: `volra deploy` → `.volra/` generated, independent stack started (3 containers each)
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

Two weeks later, Diego updates his agent's system prompt. He runs `volra deploy` in the same directory. Volra detects `.volra/` exists → update mode. Rebuilds only the agent container, restarts services. 8 seconds of downtime. Prometheus metrics history preserved. Same command, same workflow, no new concepts.

**Functional Steps:**

1. User modifies agent code (e.g., `main.py`)
2. User runs `volra deploy`
3. System detects `.volra/` directory exists → update mode
4. System regenerates `.volra/Dockerfile` (in case dependencies changed)
5. System runs `docker compose -f .volra/docker-compose.yml up -d --build`
6. Only agent container rebuilds; Prometheus, Grafana untouched
7. System verifies health check on new container
8. Output: same format as initial deploy, with note "Updated existing deployment"
9. Prometheus data volumes persist across redeploy

**Key architectural requirement:** `.volra/` is disposable and regenerable. To migrate to a new server: copy project (with Agentfile) → run `volra deploy` → `.volra/` regenerated for new environment.

---

### Journey 5: Diego Cleans Up

**Narrative:**

Diego finishes a project. He wants to stop the agent and free resources. There is no `volra stop` command in v0.1 — Volra generates standard Docker Compose files, and standard Docker Compose commands stop them.

The deploy output already told him: `Stop services: docker compose -f .volra/docker-compose.yml down`

He runs it. Services stop. To also remove metrics data: `docker compose -f .volra/docker-compose.yml down -v`

To fully clean up: `rm -rf .volra/`

**Requirements revealed:**
- Deploy output must include stop/cleanup commands
- `.volra/` can be safely deleted (no state that can't be regenerated)
- No dedicated cleanup command in v0.1 (workaround is documented standard Docker Compose)

---

### `--dry-run` Flag

`volra deploy --dry-run` generates the `.volra/` directory with all artifacts but does NOT execute `docker compose up`. Allows inspection of generated files before deployment.

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
├── .volra/         # generated artifacts (gitignored, disposable)
│   ├── Dockerfile
│   ├── docker-compose.yml
│   ├── prometheus.yml
│   └── grafana/
│       ├── dashboards/
│       └── provisioning/
└── .gitignore           # includes .volra/ and .env
```

- **Agentfile** lives in project root — it's project config, versioned with git
- **`.volra/`** lives in project root — generated artifacts, gitignored, disposable and regenerable
- `volra init` generates Agentfile + adds `.volra/` to .gitignore
- `volra deploy` generates `.volra/` contents + executes

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
| Diego Returns | `.volra/` detection (always regenerate), selective rebuild (agent only), metrics persistence, `--dry-run` flag |
| Diego Cleans Up | Cleanup hints in deploy output, `.volra/` is disposable, no dedicated command needed |

**Cross-Journey Requirements:**
- `volra deploy` is idempotent: works for first deploy, update, and retry after failure
- All generated artifacts are human-readable, editable, and live in `.volra/`
- Error messages always follow: ❌ What happened. How to fix it.
- Grafana dashboards display probe-based metrics honestly labeled
- `.volra/` is disposable and regenerable — Agentfile is the portable config

### OSS Project Setup Requirements

*(Derived from contributor journey, not a product user journey)*

- `README.md` with installation from source
- `CONTRIBUTING.md` with PR rules (conventional commits, tests required)
- GitHub Issue templates (bug report, feature request)
- CI that runs tests on PRs automatically
- Clear module boundaries in codebase for easy contribution

## Innovation & Novel Patterns

### Innovation Type: Positional (v0.1), Technical (v0.2+)

Volra v0.1 does not introduce novel technology. Every component exists: Docker Compose, Prometheus, Grafana, health checks, YAML configuration. The innovation is **positional** — the combination of self-hosted + framework-agnostic + CLI-first + open-source does not exist in any competitor as of March 2026.

**Why this niche exists:** Cloud platforms (Railway $100M, Vercel) serve the "deploy for me" segment but cannot serve teams that need data sovereignty, compliance control, or vendor independence. Framework-specific tools (LangGraph Platform, CrewAI AMP) lock users into one ecosystem. Generic PaaS (Render, Fly.io) require significant configuration per agent. Volra fills the gap: opinionated agent operations for developers who need to own their infrastructure.

**Honest update (March 2026):** Aegra (Apache 2.0) exists as a self-hosted LangGraph Platform alternative. Volra's differentiation over Aegra is framework-agnosticism and CLI-first DX. The competitive window is 6-12 months, not the 12-18 originally assumed.

### Central Hypothesis

> "Developers who need self-hosted agent infrastructure abandon projects because of deployment friction."

**Original hypothesis (pre-market-research):** "Developers abandon because of infrastructure friction, not framework limitations."

**Updated (March 2026):** McKinsey data shows the primary production failure cause is workflow/agent quality (~75%), not infrastructure. However, for the segment that HAS working agents and NEEDS self-hosted deployment, infrastructure friction is the blocker. Volra targets this specific segment — not all agent developers, but those who need production infrastructure they own.

This narrower hypothesis is more defensible and aligns with the "own your infrastructure" positioning.

### Moat Analysis (Updated March 2026)

| Timeframe | Moat Type | Strength | Notes |
|-----------|-----------|----------|-------|
| v0.1 (launch) | Niche positioning | Weak | Self-hosted + framework-agnostic + CLI-first is unique but easy to replicate |
| v0.1 (6 months) | Agentfile standard + community | Moderate | Opinionated schema with adoption creates switching cost |
| v0.2 | Agent introspection (Level 2) | Strong | Framework-specific observability (LangGraph traces, cost attribution) requires deep integration — neither Railway nor Aegra have this |
| v0.3 | MCP composition runtime | Strong | Self-hosted MCP runtime for agent-to-agent communication — no competitor offers this |

**Honest assessment:** v0.1 has no technical moat. Aegra already exists in the adjacent space. The bet is that CLI-first DX, framework-agnosticism, and rapid iteration create enough momentum to reach v0.2, where Level 2 observability becomes a genuine technical differentiator that competitors cannot easily replicate.

**Defensive strategy:**
1. Do NOT compete with Railway/Vercel on deployment ease — they have $100M+
2. DO compete on: ownership, compliance readiness, framework-agnosticism, transparency
3. Level 2 observability is the moat — invest heavily here
4. MCP Server is competitive parity — must ship by v0.2 (Railway has it since Aug 2025)

### Validation Plan (COMPLETED)

*Originally planned for pre-code validation. Code is now complete (v2.1). Validation shifted to post-launch user feedback.*

| # | Claim | Original Plan | Actual Outcome |
|---|-------|--------------|----------------|
| 1 | Infrastructure friction is a blocker | 5 interviews | Validated by market research: ~75% of agent projects fail to reach production |
| 2 | "15 minutes to deploy" is achievable | Prototype | Validated: E2E tests show deploy in 4-8 seconds (warm), TTFD < 15 min confirmed |
| 3 | Grafana dashboards provide value | Show mock | Validated: 4 dashboard variants generated automatically |
| 4 | Developers trust generated Dockerfiles | Dry-run review | Validated: multi-stage builds, security contexts, all human-readable |

**Post-launch validation:** The real validation is now external adoption. Go/No-Go gate at 3 months post-launch.

### Risk: Competitive Landscape (Updated March 2026)

**Primary risks:**

| Risk | Impact | Mitigation |
|------|--------|------------|
| Railway ($100M) accelerates agent tooling | HIGH | Don't compete on cloud deploy — own the self-hosted niche |
| Aegra gains LangGraph community traction | HIGH | Ship Level 2 (deeper than Aegra) + framework-agnostic differentiator |
| Vercel expands AI Gateway to self-hosted | MEDIUM | Unlikely (SaaS business model), but monitor |
| MCP standard shifts away from current spec | LOW | Foundation governance makes radical changes unlikely |
| < 10 external users at 3 months | HIGH | Pause v0.2 development, validate demand, consider pivot |

## Competitive Landscape (Verified March 2026)

### Direct Competitors

| Competitor | Type | Funding | Self-Hosted | Framework | Threat Level |
|------------|------|---------|-------------|-----------|-------------|
| Railway | SaaS PaaS | $100M (Jan 2026) | No | Agnostic | HIGH |
| Vercel | SaaS PaaS | Public | No | Agnostic (AI SDK) | MEDIUM |
| Aegra | OSS | N/A | Yes | LangGraph only | HIGH |
| LangGraph Platform | Freemium | LangChain-backed | Partial | LangGraph only | MEDIUM |
| CrewAI AMP | Paid SaaS/Self-hosted | Funded | Yes ($99+/mo) | CrewAI only | LOW |

### Competitive Positioning

Volra's unique combination: **self-hosted + framework-agnostic + CLI-first + open-source** does not exist in any competitor as of March 2026.

| Dimension | Railway | Vercel | Aegra | LangGraph Platform | Volra |
|-----------|---------|--------|-------|-------------------|-------|
| Self-hosted | No | No | Yes | Partial | **Yes** |
| Framework-agnostic | Yes | Yes | No | No | **Yes** |
| Open-source | No | No | Yes | Partial | **Yes** |
| CLI-first DX | No | No | No | No | **Yes** |
| MCP integration | Yes | Yes | No | No | **v0.2** |
| Agent observability | Yes (SaaS) | Yes | Partial | Yes | **v0.2 (Level 2)** |
| Cost tracking | No | Yes | No | Yes | **v0.2** |

### Competitive Strategy

1. **Do NOT compete** with Railway/Vercel on "easy cloud deploy" — they have $100M+ and established platforms
2. **DO compete** on: infrastructure ownership, compliance readiness, framework-agnosticism, full transparency
3. **Differentiate** via Level 2 observability (agent-specific traces, cost attribution) — neither Railway nor Aegra offer this
4. **Achieve parity** with MCP Server by v0.2 — Railway has had MCP since August 2025
5. **Target segment**: Developers and teams that NEED self-hosted (data sovereignty, compliance, vendor independence, cost control)

### Market Context

- AI Agents market: $7.6B (2025) → $10.9B (2026), 49.6% CAGR (Grand View Research)
- ~75% of agent projects fail to reach production (McKinsey, Gartner)
- Primary failure cause: workflow/agent quality (~75%), NOT infrastructure
- MCP is de facto standard: Linux Foundation governance, 97M SDK downloads/month
- EU AI Act full enforcement: August 2026 — self-hosted facilitates compliance

### Non-Competitor: Docker Model Runner

Docker Model Runner (DMR) is an inference engine (runs LLMs locally via llama.cpp/vLLM), not an agent operations platform. DMR operates at the MODEL layer; Volra operates at the AGENT layer. They are complementary — a Volra-deployed agent could use DMR as its model backend.

## CLI Tool + Developer Platform Requirements

### Deployment Model

**v0.1 is explicitly single-machine.** All agents, monitoring services, and dashboards run on `localhost`. There is no remote deployment, no SSH, no multi-machine orchestration.

**v0.1 is one stack per project directory.** Each `volra deploy` in a separate directory launches its own independent Prometheus + Grafana stack (3 containers: agent + prometheus + grafana). Priya with 3 agents opens 3 Grafana tabs (ports 3001, 3011, 3021). A unified multi-agent dashboard is a v0.2 feature.

**Docker requirement:** Docker Compose V2 (Compose Specification, no `version` field in generated files). Minimum Docker Engine version: current stable at time of v0.1 release. Checked by `volra doctor`.

### Command Structure

Volra v0.1 exposes 4 commands with a flat, no-subcommand structure:

| Command | Purpose | Input | Output | Exit Code |
|---------|---------|-------|--------|-----------|
| `volra doctor` | Environment diagnosis | none | Pass/fail checklist with fix suggestions | 0 (all pass) / 1 (failures) |
| `volra init .` | Auto-detect agent, generate config | directory path | Detection summary + Agentfile | 0 (success) / 1 (detection failed) |
| `volra deploy` | Generate stack + deploy | none (reads Agentfile from cwd) | Artifact list + service status + URLs | 0 (healthy) / 1 (failed) |
| `volra status` | Report agent health | none | Health summary (agents + services) | 0 (all healthy) / 1 (unhealthy) |

**Agentfile lookup:** Current working directory only. No recursive search upward. If not found: `❌ No Agentfile found. Run volra init . first.`

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

**`volra status` output spec:**

```
$ volra status

  Volra Status

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
# Agentfile — generated by volra init, versioned with git
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

**`env` field semantics:** Lists environment variable *names* (not values). Values come from a `.env` file in the project root. `volra init` creates a `.env.example` listing the detected variables. `volra deploy` generates `docker-compose.yml` with `env_file: ../.env`. The `.env` file lives in the project root (next to Agentfile), NOT inside `.volra/`, ensuring `.volra/` remains fully disposable. `volra init` adds both `.volra/` and `.env` to `.gitignore`.

**Dockerfile mode:**
- `auto` (default): Generate Dockerfile from detection results
- `custom`: Use existing `./Dockerfile` in project root (user-managed)

No implicit `skip` mode. If a Dockerfile exists and Agentfile says `auto`, Volra overwrites the generated one in `.volra/Dockerfile` (not the project root). If Agentfile says `custom`, Volra uses the project root Dockerfile directly.

### Detection Logic (volra init)

Detection is **best-effort with safe defaults.** When detection fails partially, Volra uses defaults and emits warnings — never fails silently, never blocks without reason.

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
curl -fsSL https://get.volra.dev | sh
```

- Detects OS (macOS/Linux) and architecture (ARM64/AMD64)
- Verifies SHA256 checksum before installing binary
- Default install location: `/usr/local/bin/volra`
- If no write permission: suggests `--prefix ~/.local/bin` and PATH instructions
- No runtime dependencies (Go compiles to static binary)
- Docker is a runtime dependency (checked by `volra doctor`), not an install dependency

**Future (v0.2+):** Homebrew, apt repository, GitHub Releases.

### Platform Support

Volra v0.1 is tested on 3 platforms:

| Platform | Architecture | Tested in CI |
|----------|-------------|--------------|
| macOS | ARM64 (Apple Silicon) | Yes |
| Ubuntu 24.04 | AMD64 | Yes |
| Debian 12 | AMD64 | Yes |

Other platforms may work but are not tested or supported in v0.1.

### Generated Artifacts

Volra generates, it does not abstract. All artifacts are human-readable, editable, idempotent, and disposable.

| Artifact | Location | Purpose |
|----------|----------|---------|
| Dockerfile | `.volra/Dockerfile` | Container image for the agent |
| docker-compose.yml | `.volra/docker-compose.yml` | Full stack (agent + monitoring) |
| prometheus.yml | `.volra/prometheus.yml` | Metrics scraping config + alert rules |
| Grafana dashboards | `.volra/grafana/dashboards/` | 2 JSON dashboard definitions (Overview + Detail) |
| Grafana provisioning | `.volra/grafana/provisioning/` | Auto-configured Prometheus datasource + default dashboard |

**Regeneration policy:** `volra deploy` ALWAYS regenerates everything in `.volra/`. No merge logic, no edit detection. If you edited a generated file and want to preserve the change: either (a) modify the Agentfile so generation produces what you want, or (b) switch to `dockerfile: custom` for the Dockerfile case.

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

**`volra doctor`:**

| Error | Output |
|-------|--------|
| Docker not installed | `❌ Docker not installed. Install: https://docs.docker.com/get-docker/` |
| Docker not running | `❌ Docker is not running. Start Docker Desktop and try again.` |
| Port in use | `❌ Port 3001 in use (PID 1234: node). Free it or change port.` |

**`volra init`:**

| Error | Output |
|-------|--------|
| No Python project | `❌ No Python project detected. Volra requires requirements.txt or pyproject.toml.` |
| No entry point | `❌ No entry point found. Create main.py or specify in Agentfile.` |
| Multiple entry points | `⚠️ Multiple entry points detected (main.py, app.py). Using main.py. Override in Agentfile.` |
| Partial port detection | `⚠️ Could not detect port. Using default: 8000. Override in Agentfile → port.` |

**`volra deploy`:** (see Journey 2 error catalog: OOM, health check timeout, build failure, port in use, Docker not running)

**`volra status`:**

| Error | Output |
|-------|--------|
| No deployment | `❌ No deployment found. Run volra deploy first.` |
| Docker not running | `❌ Docker is not running. Start Docker and try again.` |
| Agent unhealthy | `⚠️ Agent unhealthy: no response on :8000/health. Check logs: docker logs volra-agent-1` |
| Docker daemon restarted | `❌ Docker daemon restarted. Your agents stopped. Run volra deploy to restart.` |

### Deferred to Architecture Document

The following technical decisions are out of scope for the PRD and will be resolved in the Architecture document:

- Internal Go package structure and CLI framework selection
- Testing strategy (unit, integration, golden file approach)
- Prometheus scraping topology for multi-agent scenarios (v0.2)
- Docker Compose generation internals (templating approach)
- CI/CD pipeline design for Volra itself

## Project Scoping & Phased Development

### MVP Strategy & Philosophy

**MVP Approach:** Problem-solving MVP — prove that the core workflow (init → deploy → dashboard) solves a real problem (infrastructure friction) for a specific user (Diego, backend dev deploying his first agent).

**Resource Reality:** 1 developer, 6-8 weeks. Every scope decision passes the filter: "Can one person build, test, and ship this in 8 weeks?"

**Central bet:** Developers will use Volra because the alternative (manual Docker + Prometheus + Grafana setup) takes a week. If the alternative only takes an hour for a competent dev, Volra's value proposition collapses.

**Measurement definitions:**
- **Install** = curl install script completes successfully (exit code 0)
- **Deploy** = `volra deploy` completes with health check passing (exit code 0)
- **Install → Deploy Conversion Rate** = Deploy count / Install count (target > 50%)

### MVP Feature Set (Phase 1 — v0.1)

**4 commands, 1 workflow, 1 persona, 3 containers per project:**

| Feature | v0.1 Scope (Built) | NOT in v0.1 |
|---------|-----------|----------------------|
| `volra doctor` | Environment checks with fix suggestions | No auto-fix, no plugin checks |
| `volra init` | Python project detection, Agentfile generation | No interactive wizard, no non-Python support |
| `volra deploy` | Full stack generation + docker compose up + GPU pre-flight | No remote deploy, no K8s, no hot-reload |
| `volra status` | Health check + service status | No historical metrics, no alerting config |
| Agentfile | ~15 fields (v2.1), services, security, gpu, build | No conditional logic, no includes |
| Dockerfile | auto (multi-stage) + custom modes + build-time setup | No multi-language |
| Services | Redis, PostgreSQL, ChromaDB with auto-healthchecks + resource limits | Max 5 services |
| Security | read_only, no_new_privileges, drop_capabilities, auto-tmpfs | No RBAC, no audit trail |
| Monitoring | Prometheus + Blackbox Exporter + Grafana | No Alertmanager, no tracing |
| Dashboards | 4 Grafana dashboards (Overview + Detail × generic/LangGraph) | No custom dashboards |
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

**Journey explicitly NOT supported in v0.1:** Priya's unified multi-agent dashboard (Journey 3) — each deploy creates an independent stack. Priya can use Volra but gets separate Grafana instances, not a unified view.

### Milestones (COMPLETED)

**Alpha (Week 4) — COMPLETED:** doctor + init + deploy working on macOS ARM64.

**v0.1 (Week 8) — COMPLETED:** Full CLI (doctor, init, deploy, status), 4 dashboard variants, services, security contexts, GPU support, 297+ tests, 33 E2E.

**v2.1 (post-E2E) — COMPLETED:** 7 limitations resolved (healthchecks, port separation, resource limits, GPU pre-flight, build-time setup, secrets separation, read_only + tmpfs).

### Post-MVP Features (Internal Development History)

*Internal versions v1.0 → v2.1 represent development iterations. Public release versioning starts at v0.1.*

**v1.0 → v1.2 (internal):** health_timeout, multi-stage Dockerfiles, /metrics auto-detection, persistent volumes, LLM token tracking panels

**v2.0 → v2.1 (internal):** Services (Redis, PostgreSQL, ChromaDB), security contexts, GPU support, service healthchecks, port separation, resource limits, secrets separation, build-time model downloads

**All internal versions are included in public v0.1.** The next public release is v0.2.

### Launch Strategy

**v0.1 is code-complete and ready for early access launch.**

**Launch package (minimum viable):**
1. README.md with clear value proposition ("own your agent infrastructure")
2. Install script (macOS ARM64, Linux AMD64) — `curl | sh`
3. GitHub Release with cross-compiled binaries + SHA256 checksums
4. 3 example projects as quick-start references
5. GitHub repo rename: MegaCenter → volra
6. Launch announcements: Hacker News, Reddit r/MachineLearning, Twitter/X

**Timeline:** This week (March 2026).

**Early access expectations:**
- "Early access" label signals the product works but is evolving
- Feedback drives v0.2 priorities (Level 2 vs MCP Server vs templates)
- Bug reports and feature requests welcome via GitHub Issues

### v0.2 Detailed Roadmap

**v0.2 — Differentiation & Developer Experience (6 weeks post-launch):**

| Priority | Feature | Effort | Dependencies | Status |
|----------|---------|--------|-------------|--------|
| P0 | `volra quickstart` + 3 templates (basic, rag, conversational) | 1 week | Convert E2E examples | **DONE** (Epic 12) |
| P0 | `volra logs` (streaming log access) | 2 days | None | **DONE** (Epic 12) |
| P1 | Level 2: Framework-agnostic agent observability (`volra-observe` package) | 2-3 weeks | New Python package | **DONE** (Epic 13) |
| P1 | Volra MCP Server (deploy from editors) | 1-2 weeks | Direct protocol impl | **DONE** (Epic 14) |
| P2 | LLM cost tracking dashboard panels | 1 week | Level 2 | **DONE** (Epic 13.4) |
| P2 | `volra quickstart` interactive mode | 2 days | Templates | **DONE** (Epic 15) |
| P2 | `--json` output flag for CI/CD | 2 days | None | **DONE** (Epic 15) |

**v0.2 Status: COMPLETE** — All features implemented. 384 Go tests + 25 Python tests = 409 total, all passing.

### v0.3 Detailed Roadmap

**v0.3 — Composition & Framework Depth (~3 months post-v0.2):**

| Priority | Feature | Effort | Dependencies |
|----------|---------|--------|-------------|
| P0 | Framework-specific addons: LangGraph graph state + CrewAI crew metrics | 2-3 weeks | Level 2 |
| P0 | Auto-injected instrumentation (zero-config Level 2) | 1 week | Level 2 |
| P1 | Multi-agent composition via MCP | 3-4 weeks | MCP Server |
| P1 | Second framework support (demand-driven) | 2-3 weeks | User feedback |
| P2 | Basic audit trail (execution log) | 2 days | None |
| P2 | EU AI Act compliance documentation | 1 week | None |

### v0.4+ Roadmap

- `volra dev` (hot-reload local development)
- RBAC, MCP Hub/Gateway
- Advanced alerting (degradation detection)
- Agent Marketplace (community templates)
- Custom control plane + Console UI
- Multi-tenancy, Federation, Kubernetes native

### Scope Management Rules

**Scope (commands, features) is zero-sum:** No new command is added without removing one or explicit justification. The Agentfile has 7 fields in v1.0. New fields in v1.1+ are justified by E2E testing limitations (L1-L10) — each new field maps to a specific operational gap discovered during real-world agent deployment testing. v1.1 adds 1 field (health_timeout), v1.2 adds 1 field (volumes) and extends existing dashboard variants (no new detection mechanisms), v2.0 adds 3 fields (services, security, gpu) with a schema version bump.

**Polish (flags, error messages, UX) is free:** Adding `--verbose` for debugging, improving an error message, or handling a new edge case in detection is developer experience, not scope creep.

**Cut list (ordered, if time runs out):**

| Priority | Feature | Criterion | Rationale |
|----------|---------|-----------|-----------|
| Cut first | `--dry-run` | Diego doesn't need it for Journey 1 | CI/CD feature, not onboarding |
| Cut second | `volra status` | Diego can open Grafana instead | Nice-to-have, Grafana covers the need |
| Cut third | SHA256 verification | Security, not functionality | Can be added in patch release |
| Cut fourth | Error catalog (init/status) | Deploy errors are essential, others are nice-to-have | Core 5 deploy errors stay |
| Cut fifth | Detail dashboard | Overview is sufficient for MVP | Single dashboard still delivers value |
| **Never cut** | `doctor` | Prerequisite for Journey 1 | Without it, Docker errors are incomprehensible |
| **Never cut** | `init` | Core of Journey 1 | No Agentfile = no Volra |
| **Never cut** | `deploy` | Core of Journey 1 | The product IS deploy |
| **Never cut** | Overview dashboard | The "aha moment" | Diego sees metrics = Volra delivers |

### Risk Mitigation Strategy

**Technical Risks:**

| Risk | Impact | Mitigation |
|------|--------|------------|
| Dockerfile generation fails for edge cases | High | `dockerfile: custom` escape hatch. Error messages point to manual Dockerfile. |
| Detection accuracy < 90% for LangGraph | Medium | Generic fallback always works. Defaults are the safety net. |
| Docker Compose version incompatibilities | High | `volra doctor` checks version. Require Compose V2. |
| Large image sizes without multi-stage | Low | Documented trade-off. Functional but not optimized. v0.2 improvement. |
| 3 containers per project = resource pressure | Medium | Documented: ~500MB RAM per stack. Laptop with 16GB supports 3-4 concurrent projects. |

**Market Risks (Updated March 2026):**

| Risk | Impact | Mitigation |
|------|--------|------------|
| Railway ($100M) accelerates agent tooling | HIGH | Don't compete on cloud deploy — own the self-hosted niche |
| Aegra gains LangGraph community traction | HIGH | Ship Level 2 (deeper than Aegra) + framework-agnostic differentiator |
| < 10 external users at 3 months | HIGH | Pause v0.2 development, validate demand, consider pivot |
| Vercel expands AI Gateway to self-hosted | MEDIUM | Unlikely (SaaS model), but monitor |
| Agent production failures are mostly workflow, not infra | MEDIUM | Honest positioning — Volra helps the segment that HAS good agents |

**Resource Risks:**

| Risk | Impact | Mitigation |
|------|--------|------------|
| Solo developer for v0.2 (6+ weeks) | Medium | Prioritize ruthlessly. P0 first, P2 only if time allows. |
| Community doesn't form | High | Launch early, iterate on feedback, contribute to adjacent projects. |
| Scope creep from user requests | Medium | Sprint Change Proposal process. Changes go through PM review. |

## Functional Requirements

*This is the capability contract for Volra v0.1. UX design, architecture, and epic breakdown will ONLY address capabilities listed here. Capabilities not listed will NOT exist in the product.*

*FRs marked [SHOULD] are aligned with the cut list — they may be deferred if time runs out. All unmarked FRs are MUST.*

### Environment Diagnosis

- **FR1:** Developer can run a pre-flight check that validates all prerequisites for Volra to function: Docker installed, Docker running, Docker Compose V2 available, required ports free, Python ≥ 3.10 present, sufficient disk space, and Volra version reported
- **FR2:** Developer can see a pass/fail result for each individual check with a specific fix suggestion for each failure
- **FR3:** Developer can determine from the command exit code whether all checks passed (0) or any failed (1)

### Project Detection & Configuration

- **FR4:** Developer can point Volra at a Python project directory and receive an auto-generated Agentfile with a summary of detected values and instructions to customize
- **FR5:** System can detect the agent framework used (generic or LangGraph) by scanning dependency files and Python imports
- **FR6:** System can detect the application entry point by scanning for common filenames (main.py, app.py, server.py) in priority order
- **FR7:** System can detect the application port by scanning entry point code for server startup patterns
- **FR8:** System can detect health check endpoints by scanning route decorators
- **FR9:** System can detect environment variable references in agent code (os.environ, os.getenv patterns) to populate `.env.example`
- **FR10:** When detection is partial or ambiguous, system uses safe defaults and emits a warning with override instructions referencing the specific Agentfile field
- **FR11:** Developer can override any detected value by editing the generated Agentfile
- **FR12:** System generates a `.env.example` file listing detected environment variable names
- **FR13:** System adds `.volra/` and `.env` to the project's `.gitignore`
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

- **FR38:** Developer can install Volra via a single shell command that auto-detects OS and architecture
- **FR39 [SHOULD]:** Installation script verifies binary integrity via SHA256 checksum before placing the binary
- **FR40:** When the install location requires elevated permissions, system suggests an alternative user-local path with PATH configuration instructions
- **FR41:** Developer can check the installed Volra version

### Configurable Health & Build Optimization (v1.1)

- **FR42:** Developer can configure health check timeout in Agentfile via `health_timeout` field (integer, seconds, range 10-600, default 60) to support ML models and agents with long startup times
- **FR43:** System generates multi-stage Dockerfiles (builder + runtime stages) to reduce image size, with pip cache mounts for faster rebuilds
- **FR44:** When the agent exposes a `/metrics` endpoint, system auto-detects it and adds custom metrics panels to Grafana dashboards (request count, latency histogram, custom counters) alongside existing probe panels

### Persistent Data & LLM Observability (v1.2)

- **FR45:** Developer can declare persistent volume mounts in Agentfile via `volumes` field to preserve agent data (model weights, databases, caches) across container rebuilds
- **FR46:** Dashboard panels for LLM token tracking — when the agent exposes `prometheus_client` metrics, the metrics dashboard variants include LLM-specific panels alongside generic request panels: token consumption rate (`llm_tokens_total`), cost trending (`llm_request_cost_dollars_total`), and per-model request breakdown (`llm_model_requests_total`). Panels display "No data" gracefully when the agent does not expose these metrics. No additional detection or configuration required beyond existing HasMetrics (FR44). The Volra LLM Metrics Convention defines the recommended metric names (documented in deploy guide)

### Infrastructure Services & Security (v2.0)

- **FR47:** Developer can declare infrastructure services in Agentfile via `services` field as a map of service name to service definition. Each service definition includes: `image` (required), `port` (optional, host-exposed port), `volumes` (optional, container mount paths), `env` (optional, environment variable names). System generates corresponding docker-compose service definitions with automatic volra network membership, depends_on from agent service, and container naming convention `{projectName}-{serviceName}`. Maximum 5 services. Service names must be DNS labels and cannot conflict with reserved names (agent, prometheus, grafana, blackbox)
- **FR48:** System generates security context for agent container with configurable options via `security` field in Agentfile: `read_only` (read-only root filesystem), `no_new_privileges` (prevent privilege escalation), `drop_capabilities` (list of Linux capabilities to drop). Maps to docker-compose `read_only`, `security_opt`, and `cap_drop` directives
- **FR49:** ~~System monitors WebSocket/SSE streaming connections~~ — **DEFERRED** to backlog. No current E2E test agent uses streaming connections. Will be planned when a real use case emerges
- **FR50:** Developer can declare GPU/hardware acceleration requirements in Agentfile via `gpu` field (boolean or string specifying runtime). System configures docker-compose `deploy.resources.reservations.devices` with GPU capabilities. Requires nvidia-container-toolkit on host

### Explicitly Out of Scope

- **Horizontal scaling** (L9): Multi-instance orchestration and load balancing are Kubernetes concerns. Volra targets single-instance deployments. Users needing horizontal scaling should use Kubernetes directly.

## Non-Functional Requirements

### Performance

- **NFR1:** CLI commands (doctor, init, status) complete in < 5 seconds on target platforms
- **NFR2:** `volra deploy` first run (cold, including Docker base image pull) completes in < 5 minutes for a typical Python agent with < 20 pip dependencies
- **NFR3:** `volra deploy` subsequent runs (warm, base image cached) complete in < 3 minutes; re-deploy with no dependency changes completes in < 30 seconds
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

- **NFR12:** Volra binary runs on macOS ARM64, Ubuntu 24.04 AMD64, and Debian 12 AMD64 without additional dependencies
- **NFR13:** Generated Docker Compose files are compatible with Docker Compose V2 (Compose Specification)
- **NFR14:** Generated Dockerfiles produce functional images on both AMD64 and ARM64 Docker hosts
- **NFR15:** CLI output renders correctly in terminals with and without emoji/Unicode support (respects `NO_COLOR` and `TERM=dumb`) and in terminals with minimum 60 columns width

### Security

- **NFR16:** Environment variable values (API keys, secrets) never appear in generated artifacts inside `.volra/` — only variable names are referenced, values come from `.env` in project root
- **NFR17:** `.env` file is automatically added to `.gitignore` to prevent accidental commit of secrets
- **NFR18 [SHOULD]:** Install script verifies binary integrity via SHA256 checksum before execution
- **NFR19:** Generated Grafana instance uses anonymous access (no credentials to manage) — acceptable for localhost-only v0.1 deployment
- **NFR20:** No telemetry, no phone-home, no data collection of any kind in v0.1

### Resource Efficiency

- **NFR21:** Volra monitoring stack (Prometheus + Grafana, excluding agent container) uses < 500MB RAM at rest

### Developer Experience

- **NFR22:** Every user-facing error message follows the pattern: what happened + actionable fix instruction. No error exits without guidance
- **NFR23:** Every warning message follows the pattern: what happened + what was assumed + how to override

### Backward Compatibility

- **NFR24:** Agentfiles created for v1.0 (version: 1, 7 fields) must work without modification on Volra v1.1 and v1.2. New optional fields use zero-value defaults that preserve v1.0 behavior
- **NFR25:** Generated artifacts from v1.1 must be functionally equivalent to v1.0 when no new optional fields are specified in the Agentfile
