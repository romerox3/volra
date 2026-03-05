"""{{.Name}} — MCP-compatible tool server, deployed with Volra."""

import re
from collections import Counter
from contextlib import asynccontextmanager

from fastapi import FastAPI
from mcp.server.fastmcp import FastMCP

# ---------------------------------------------------------------------------
# MCP Server
# ---------------------------------------------------------------------------

mcp = FastMCP("{{.Name}}")

# ---------------------------------------------------------------------------
# Sample data — replace with your own data source
# ---------------------------------------------------------------------------

SAMPLE_DATA = [
    {"id": 1, "title": "Getting Started with Volra", "content": "Volra deploys AI agents with monitoring using Docker Compose."},
    {"id": 2, "title": "Agentfile Reference", "content": "The Agentfile configures your agent: name, port, services, and observability."},
    {"id": 3, "title": "Monitoring with Grafana", "content": "Volra includes Prometheus and Grafana for real-time agent monitoring."},
    {"id": 4, "title": "Adding Custom Tools", "content": "Define Python functions and expose them as tools for your AI agent."},
    {"id": 5, "title": "Deploying to Production", "content": "Use volra deploy with Docker Compose for production-ready deployments."},
]

# ---------------------------------------------------------------------------
# Tools
# ---------------------------------------------------------------------------


@mcp.tool()
def text_length(text: str) -> int:
    """Count the number of characters in a text."""
    return len(text)


@mcp.tool()
def word_frequency(text: str) -> dict[str, int]:
    """Analyze word frequency in a text. Returns the top 20 most common words."""
    words = re.findall(r"\w+", text.lower())
    counts = Counter(words)
    return dict(counts.most_common(20))


@mcp.tool()
def search_data(query: str) -> list[dict]:
    """Search the sample data by keyword. Replace with your own data source."""
    query_lower = query.lower()
    return [
        item for item in SAMPLE_DATA
        if query_lower in item["title"].lower() or query_lower in item["content"].lower()
    ]


# ---------------------------------------------------------------------------
# Application
# ---------------------------------------------------------------------------


@asynccontextmanager
async def lifespan(_app: FastAPI):
    """Track available tool count on startup."""
    _app.state.tool_count = 3
    yield


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


@app.get("/health")
async def health():
    """Health check required by Volra (separate from MCP protocol)."""
    return {
        "status": "ok",
        "tools": app.state.tool_count,
    }


# Mount MCP on /mcp path
app.mount("/mcp", mcp.streamable_http_app())


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
