"""{{.Name}} — HuggingFace smolagents code agent, deployed with Volra."""

import asyncio
import os
from contextlib import asynccontextmanager

from fastapi import FastAPI
from pydantic import BaseModel
from smolagents import CodeAgent, DuckDuckGoSearchTool, OpenAIServerModel, tool

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")

# ---------------------------------------------------------------------------
# Tools
# ---------------------------------------------------------------------------


@tool
def calculator(expression: str) -> str:
    """Evaluate a mathematical expression safely.

    Args:
        expression: A math expression like '2 + 2' or '(10 * 5) / 3'.
    """
    allowed = set("0123456789+-*/.() ")
    if not all(c in allowed for c in expression):
        return "Error: expression contains invalid characters"
    try:
        result = eval(expression, {"__builtins__": {}})  # noqa: S307
    except Exception as exc:
        return f"Error: {exc}"
    return str(result)


TOOLS = [calculator, DuckDuckGoSearchTool()]

# ---------------------------------------------------------------------------
# Models
# ---------------------------------------------------------------------------


class SolveRequest(BaseModel):
    task: str


class SolveResponse(BaseModel):
    result: str


# ---------------------------------------------------------------------------
# Application
# ---------------------------------------------------------------------------


def create_agent() -> CodeAgent | None:
    """Create a smolagents CodeAgent if API key is available."""
    if not OPENAI_API_KEY:
        return None
    model = OpenAIServerModel(model_id=OPENAI_MODEL, api_base=OPENAI_BASE_URL)
    return CodeAgent(tools=TOOLS, model=model, max_steps=5)


def run_task(agent: CodeAgent, task: str) -> str:
    """Run a task with the agent and return the result as a string."""
    result = agent.run(task)
    return str(result)


@asynccontextmanager
async def lifespan(_app: FastAPI):
    """Build the agent on startup."""
    _app.state.agent = create_agent()
    yield


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


@app.get("/health")
async def health():
    """Check if the agent is ready."""
    llm_configured = app.state.agent is not None
    return {
        "status": "ok" if llm_configured else "degraded",
        "llm_configured": llm_configured,
        "tools": [t.name for t in TOOLS],
    }


@app.post("/solve", response_model=SolveResponse)
async def solve(req: SolveRequest):
    """Solve a task using the code agent. The agent writes Python code to find the answer."""
    if app.state.agent is None:
        return SolveResponse(
            result="[No API key configured] Set OPENAI_API_KEY to enable the agent.",
        )
    result = await asyncio.to_thread(run_task, app.state.agent, req.task)
    return SolveResponse(result=result)


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
