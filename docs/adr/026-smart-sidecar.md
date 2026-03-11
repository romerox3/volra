# ADR-26: Smart Sidecar — Go Reverse Proxy for A2A Task Execution

## Status

Accepted

## Context

Volra v1.1 ships A2A agent cards, federation discovery, and MCP Gateway routing. However, agents cannot receive A2A `Tasks/send` calls because the nginx:alpine proxy only serves the static agent card and reverse-proxies HTTP — it has no understanding of the A2A JSON-RPC protocol.

To close this gap, agents need an A2A-aware proxy that can:
1. Serve the agent card at `/.well-known/agent-card.json`
2. Handle `Tasks/send` JSON-RPC requests and translate them to HTTP calls to the agent
3. Reverse proxy all other traffic to the agent

## Decision

Replace `nginx:alpine` with a custom Go reverse proxy (`volra-proxy`) that runs as a Docker sidecar alongside the agent. The proxy is built as a separate binary (`cmd/volra-proxy`) and published as `ghcr.io/romerox3/volra-proxy:<version>`.

Three operation modes via Agentfile `a2a` section:

| Mode | Behavior |
|------|----------|
| `default` (zero-config) | `Tasks/send` text → `POST /ask {"question":"<text>"}` → extracts `"answer"` |
| `declarative` | Maps skills to endpoints with configurable request/response field mapping |
| `passthrough` | Forwards raw JSON-RPC to agent's `/a2a` endpoint |

Configuration is passed via environment variables: `VOLRA_AGENT_URL`, `VOLRA_CARD_PATH`, `VOLRA_A2A_MODE`, `VOLRA_A2A_SKILLS`.

## Alternatives Considered

### 1. SDK approach — agent implements A2A directly
Rejected: Requires framework-specific SDKs (Python, Node, etc.), breaks the framework-agnostic principle. Agents would need to understand A2A protocol.

### 2. nginx + Lua sidecar
Rejected: Adds Lua dependency, harder to test, limited JSON-RPC handling capability.

### 3. Separate sidecar container (nginx + Go proxy)
Rejected: Two containers instead of one, increased complexity and resource usage for the same functionality.

## Consequences

### Positive
- Zero-config A2A task execution for any agent with an `/ask` endpoint
- Declarative skill mapping without agent code changes
- Passthrough mode for agents that natively speak A2A
- Single binary, small Docker image (~10MB vs nginx:alpine ~40MB)
- Full test coverage with httptest
- Same reverse proxy behavior for non-A2A traffic

### Negative
- Custom binary to maintain (vs off-the-shelf nginx)
- Docker image must be published to GHCR on each release
- Adds `internal/proxy` package to codebase

### Neutral
- Removes `a2a-proxy.conf.tmpl` nginx template
- Removes `GenerateA2AProxy()` function
- Adds `a2a` section to Agentfile schema
