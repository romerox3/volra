# Sprint Change Proposal — Strategic Pivot Post-Market Research

**Date:** 2026-03-04
**Trigger:** Market research reveals competitive landscape significantly different from PRD assumptions
**Scope:** MAJOR — affects PRD, Architecture, Epics, Roadmap, Positioning
**Approved by:** Antonio (Parallel Track + "Own Your Infrastructure" positioning)

---

## Section 1: Issue Summary

### Problem Statement

The v0.2 roadmap defined in the PRD (March 2) was designed assuming Volra had no direct competition and a 12-18 month window. Market research conducted March 4 reveals:

1. **Railway** raised $100M (January 2026) and has Agent Skills + MCP Server, positioning as "the new Heroku for AI agent stacks"
2. **Vercel** launched agent-specific infrastructure (Sandboxes, Fluid Compute, AI Gateway, Workflows) with the blog post "Anyone can build agents, but it takes a platform to run them"
3. **Aegra** exists as direct OSS competitor (Apache 2.0, Docker Compose, self-hosted LangGraph Platform replacement)
4. **MCP** has won as de facto standard (Linux Foundation, 97M SDK downloads/month, adopted by OpenAI/Google/AWS/Microsoft)
5. The **production failure root cause** is 75% workflow/agent quality, not infrastructure deployment

### Evidence

See full research document: `_bmad-output/brainstorming/market-research-2026-03-04.md` (25 verified sources).

### What This Means

- The positioning "deploy your agent" is **occupied** by Railway ($100M) and Vercel
- The competitive window is **6-12 months**, not 12-18
- v0.1 is **built but not launched** — every day without users is competitive ground lost
- The unique positioning is **"own your agent infrastructure"** (self-hosted + framework-agnostic + open-source)

---

## Section 2: Impact Analysis

### Epic Impact

| Epics 1-10 | Status | Impact |
|---|---|---|
| All completed | v2.1 shipped | **No change** — code is correct |

| New Epics Needed | Priority | Reason |
|---|---|---|
| Epic 11: Launch Readiness | **P0 (immediate)** | Cannot compete without users |
| Epic 12: Adoption Accelerators | **P0 (parallel with launch)** | Templates + quickstart reduce Day 0 friction |
| Epic 13: LangGraph Level 2 | **P1 (v0.2)** | Technical moat — neither Railway nor Aegra have this |
| Epic 14: MCP Integration | **P1 (v0.2)** | Competitive parity — Railway has MCP Server since Aug 2025 |
| Epic 15: Day 2 Operations | **P2 (v0.2)** | Cost tracking + logs — where agents actually die |

### PRD Changes Required

| Section | Change | Type |
|---|---|---|
| Positioning/Tagline | "Own your agent infrastructure" replaces "deploy your agent" | **REWRITE** |
| Product Scope > v0.2 | Re-prioritized: launch + templates + Level 2 + MCP | **REWRITE** |
| Product Scope > v0.3 | MCP composition + `volra dev` + second framework | **UPDATE** |
| Competitive Analysis | NEW section with verified landscape | **ADD** |
| Known Risks | Add Railway/Vercel/Aegra competitive risks | **UPDATE** |
| Launch Strategy | v0.1 exists — launch is NOW, not Week 8 | **UPDATE** |
| Success Criteria | Add competitive benchmarks | **UPDATE** |

### Architecture Changes Required

| Area | Change |
|---|---|
| New package `internal/mcp/` | MCP Server implementation (v0.2) |
| New package `internal/trace/` | LangGraph trace collection (v0.2) |
| New dependency: OpenTelemetry | For Level 2 trace export |
| New dependency: MCP SDK | For MCP server protocol |
| Extend Grafana dashboards | LangGraph-specific trace panels |
| HTTP server mode | `volra serve` for MCP Server (not just CLI) |

### Other Artifacts

| Artifact | Status | Action |
|---|---|---|
| README.md (user-facing) | Does not exist | **CREATE** — critical for launch |
| Install script (`install.sh`) | Does not exist | **CREATE** — `curl \| sh` |
| GitHub Release pipeline | Not configured | **CREATE** — cross-compile + checksums |
| GitHub repo name | "MegaCenter" | **RENAME** to "volra" |
| Examples as templates | Test fixtures only | **CONVERT** to user-facing templates |
| Landing page / docs site | Does not exist | **CREATE** — minimal, SEO-optimized |

---

## Section 3: Recommended Approach

### Selected: Parallel Track with "Own Your Infrastructure" Positioning

**Track A — Launch (Epic 11, immediate):**
Launch v0.1 as early access this week. Minimum viable launch package:
- README with clear value proposition ("own your agent infrastructure")
- Install script (macOS ARM64, Linux AMD64)
- GitHub Release with binaries
- 3 example templates (basic, RAG, conversational)
- HN/Reddit/Twitter launch posts

**Track B — v0.2 Development (Epics 12-15, parallel):**
Build differentiation features with feedback from early users:
- P0: Starter templates + `volra quickstart`
- P1: LangGraph Level 2 (traces, cost attribution, graph state)
- P1: Volra MCP Server (deploy from editors)
- P2: `volra logs` + basic cost tracking

**Rationale:**
1. Railway has $100M but is SaaS-only. We don't compete head-to-head — we serve the segment that NEEDS self-hosted.
2. Aegra is LangGraph-only. Our framework-agnostic position is the differentiator.
3. Early users provide signal for v0.2 priorities — "Level 2 vs MCP vs cost tracking" should be user-demand-driven, not assumption-driven.
4. The "early access" label sets expectations while establishing presence.

### Effort & Risk

| Track | Effort | Risk | Timeline |
|---|---|---|---|
| A: Launch | 2-3 days | Low (code exists, need packaging) | This week |
| B: Templates + quickstart | 1 week | Low | Week 1-2 |
| B: LangGraph Level 2 | 2-3 weeks | Medium (new domain) | Week 2-5 |
| B: MCP Server | 1-2 weeks | Medium (new protocol) | Week 3-5 |
| B: Cost tracking + logs | 1 week | Low | Week 5-6 |

---

## Section 4: Detailed Change Proposals

### 4.1 PRD — Positioning Change

**OLD (Product Brief):**
> "We don't build agents. We make yours work in production."
> Tagline: "From notebook to production in 5 minutes"

**NEW:**
> "We don't build agents. We give you the infrastructure to run them — on YOUR servers."
> Tagline: "Own your agent infrastructure. Deploy, monitor, and operate AI agents on your own servers — open-source, framework-agnostic, zero vendor lock-in."

**Rationale:** "Deploy in 5 minutes" competes with Railway ($100M). "Own your infrastructure" creates a category Railway/Vercel cannot enter (they are SaaS by design).

### 4.2 PRD — v0.2 Scope Rewrite

**OLD:**
```
v0.2 — Adoption Unlock:
- volra dev (hot-reload)
- volra quickstart (scaffolding)
- Level 2: LangGraph-aware observability
- Second framework support
- 3 starter templates
```

**NEW:**
```
v0.2 — Differentiation & Ecosystem:
- P0: volra quickstart (scaffolding) + 5 starter templates
- P1: Level 2 — LangGraph-aware observability (traces, cost attribution, graph state inspection)
- P1: Volra MCP Server — deploy and manage agents from any MCP-compatible editor
- P2: volra logs — streaming log access for deployed agents
- P2: Basic LLM cost tracking (token usage, per-model attribution)
- DEFERRED to v0.3: volra dev (hot-reload), second framework support
```

**Rationale:**
- `volra dev` is nice-to-have, not a differentiator (developers have their own dev workflows)
- Second framework without users = guessing. Let demand signal drive priority.
- MCP Server moves up because Railway already has one (competitive parity)
- Cost tracking added because Vercel has AI Gateway with budget controls
- Templates are P0 because reducing Day 0 friction is critical for adoption

### 4.3 PRD — v0.3 Scope Update

**OLD:**
```
v0.3 — Operations Depth:
- Multi-agent composition via MCP
- Audit trail, basic RBAC
- MCP Hub (service discovery)
- Advanced alerting
```

**NEW:**
```
v0.3 — Composition & Developer Experience:
- Multi-agent composition via MCP (agent-to-agent communication)
- volra dev (hot-reload local development)
- Second framework support (demand-driven: CrewAI or Mastra)
- Basic audit trail (execution log, decision log)
- EU AI Act compliance helpers (data residency documentation, conformity templates)
- DEFERRED to v0.4: RBAC, MCP Hub/Gateway, Advanced alerting
```

**Rationale:** EU AI Act enforcement August 2026 creates urgency for compliance features. RBAC is enterprise — defer until there's enterprise demand signal.

### 4.4 PRD — New Section: Competitive Landscape

**ADD after "Known Risks":**

```markdown
## Competitive Landscape (Verified March 2026)

### Direct Competitors
| Competitor | Type | Self-Hosted | Framework | Threat |
|---|---|---|---|---|
| Railway ($100M) | SaaS PaaS | No | Agnostic | HIGH — same "Heroku for agents" but SaaS |
| Vercel | SaaS PaaS | No | Agnostic (AI SDK) | MEDIUM — frontend-oriented |
| Aegra | OSS | Yes | LangGraph only | HIGH — same stack, same pitch |
| CrewAI Factory | Paid | Yes (cloud) | CrewAI only | LOW — framework-specific, expensive |

### Volra's Differentiation
The combination of open-source + self-hosted + framework-agnostic + CLI-first
does not exist in any competitor. This is our defensible position.

### Competitive Strategy
- Do NOT compete with Railway/Vercel on "deploy ease" — they have $100M+
- DO compete on: ownership, compliance, framework-agnosticism, transparency
- Level 2 observability is the technical moat — neither Railway nor Aegra have it
- MCP Server is competitive parity — must have by v0.2
```

### 4.5 Architecture — New Extensions Needed

**ADD to architecture document:**

```markdown
## Extension 8: LangGraph Level 2 Observability (v0.2)

New package: internal/trace/
- LangGraph callback handler that exports traces via OpenTelemetry
- Graph state inspection (node execution, checkpoint detection)
- Cost attribution per LLM call (token counting)
- New Grafana dashboard variant: overview_langgraph.json, detail_langgraph.json

New dependency: go.opentelemetry.io/otel (for trace export)
Alternative: Direct Prometheus metrics export (lighter, no new dep)

Decision needed: OpenTelemetry vs Prometheus-native for traces.

## Extension 9: Volra MCP Server (v0.2)

New package: internal/mcp/
- MCP Server exposing Volra operations as tools
- Tools: volra_deploy, volra_status, volra_logs, volra_doctor
- Transport: stdio (for editor integration) or HTTP/SSE
- Enables: "deploy from Cursor/VS Code/Claude Code"

New dependency: MCP Go SDK (or implement protocol directly — it's simple)

## Extension 10: Cost Tracking (v0.2)

Approach: Proxy or scrape LLM API usage
- Option A: Transparent proxy (high effort, full visibility)
- Option B: Parse agent logs for token usage (medium effort)
- Option C: Prometheus metrics if agent exposes them (low effort, user-dependent)

Decision needed: Which approach for v0.2 MVP.
```

---

## Section 5: Implementation Handoff

### Scope Classification: MAJOR

Requires PM (Lisa) + Architect (Frink) involvement before development.

### Handoff Plan

| Step | Agent | Workflow | Deliverable |
|---|---|---|---|
| 1 | **Lisa (PM)** | EP (Edit PRD) | Updated PRD with competitive landscape, v0.2 re-scope, positioning change |
| 2 | **Frink (Architect)** | CA (Create Architecture) | Architecture extensions 8-10 (Level 2, MCP, Cost Tracking) |
| 3 | **Lisa (PM)** | CE (Create Epics) | Epics 11-15 with stories and GWT acceptance criteria |
| 4 | **Frink (Architect)** | IR (Implementation Readiness) | Validate all new epics have architecture backing |
| 5 | **Ned (SM)** | SP (Sprint Planning) | Sprint plan for Epic 11 (launch) + Epic 12-15 (v0.2) |
| 6 | **Homer (Dev)** | DS (Dev Story) | Implementation |

### Success Criteria

1. v0.1 launched as early access within 1 week
2. First external user deploys an agent within 2 weeks of launch
3. v0.2 features (Level 2 OR MCP Server) shipped within 6 weeks
4. At least 1 GitHub star from non-team member within 1 month

---

## Approval

**Status:** APPROVED by Antonio (Parallel Track + "Own Your Infrastructure")
**Next step:** Lisa EP (Edit PRD) with competitive landscape and v0.2 re-scope
