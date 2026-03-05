# Sprint Change Proposal: Framework-Agnostic Observability Pivot

**Date:** 2026-03-04
**Triggered by:** Strategic review — user identified contradiction between positioning and Epic 13
**Scope:** Moderate (3 planning artifacts)
**Status:** APPROVED

---

## 1. Issue Summary

Epic 13 ("LangGraph Level 2a Observability") dedicates the largest v0.2 effort (2-3 weeks, P1) to a LangGraph-specific feature. This directly contradicts Volra's #1 competitive differentiator: **framework-agnostic**.

Evidence:
- PRD comparison table: "Framework-agnostic: Yes" is a key advantage vs Railway, Vercel, and Aegra
- Aegra is already LangGraph-only — our advantage IS being agnostic
- Architecture constraint "level: 2 only valid when framework: langgraph" excludes all non-LangGraph users from v0.2's main feature
- LangGraph-specific observability is not a defensible moat (Aegra, Railway, LangSmith can all replicate it)

## 2. Impact Analysis

### Epic Impact
- **Epic 13**: Full rewrite required (scope, goal, all 6 stories)
- **Epics 11-12**: Completed, no impact
- **Epics 14-15**: Independent, no impact

### Artifact Conflicts
- **PRD**: v0.2 roadmap line references "volra-langgraph" — must change to "volra-observe"
- **Architecture**: Extension 8 is entirely LangGraph-specific — full rewrite
- **Epics**: Epic 13 stories reference LangGraph callbacks, LangGraph dashboard — full rewrite

### What does NOT change
- v0.1 code (BUILT, no modifications)
- Agentfile `framework` field (stays — used for dashboard selection, not observability gating)
- Prometheus + Grafana architecture (same infrastructure)
- `observability` struct concept (same approach, different scope)

## 3. Recommended Approach: Direct Adjustment

**Zero code has been written for Epic 13.** Only planning artifacts need updating.

### Key Changes

| Before (LangGraph-specific) | After (Framework-agnostic) |
|---|---|
| `volra-langgraph` Python package | `volra-observe` Python package |
| LangGraph callback handler | OpenAI/Anthropic SDK decorators |
| `level: 2 requires framework: langgraph` | `level: 2` works with any framework |
| Graph node metrics (core) | LLM token/cost/latency metrics (core) |
| Graph metrics as optional addon | Graph metrics as optional addon |
| `langgraph-level2-overview.json` | `agent-level2-overview.json` |

### Core Metrics (universal, any agent)
1. `volra_llm_tokens_total{model, direction}` — token count per request
2. `volra_llm_cost_dollars_total{model}` — estimated cost
3. `volra_llm_request_duration_seconds{model, status}` — LLM call latency
4. `volra_llm_errors_total{model, error_type}` — errors by type
5. `volra_tool_calls_total{tool_name}` — tool/function calls

### Optional Graph Addon (when LangGraph detected)
6. `volra_graph_node_executions_total{node}` — node transitions
7. `volra_graph_execution_duration_seconds{graph}` — graph execution time

### Why This Is Better
- **10x users served**: Works with ALL Python agents, not just LangGraph subset
- **Reinforces positioning**: "Framework-agnostic" in marketing AND in features
- **Harder to replicate**: Generic LLM instrumentation across providers is more valuable than framework-specific hooks
- **Same effort**: ~2-3 weeks, similar complexity
- **No code impact**: Zero Epic 13 code exists

## 4. Detailed Change Proposals

### PRD Changes
- Line ~147: "Level 2a: LangGraph basic observability... via `volra-langgraph`" → "Level 2: Framework-agnostic agent observability (token counting, cost tracking, LLM latency) via `volra-observe` Python package"
- Any reference to "volra-langgraph" → "volra-observe"

### Architecture Changes
- Extension 8: Full rewrite — title, ADR, struct validation, package, metrics, dashboard, doctor checks

### Epic Changes
- Epic 13: Rename + rewrite all 6 stories

## 5. Implementation Handoff

- **Scope**: Moderate
- **Agent**: Lisa (PM) for PRD + Epics, Frink (Architect) for Architecture
- **Next steps**: Apply changes → proceed to Sprint Planning → implement Epic 13 (revised)
