"""{{.Name}} — OpenAI Agents SDK with tools and handoffs, deployed with Volra."""

import os
from contextlib import asynccontextmanager
from datetime import datetime, timezone

from agents import Agent, OpenAIChatCompletionsModel, Runner, function_tool, handoff
from fastapi import FastAPI
from openai import AsyncOpenAI
from pydantic import BaseModel

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")

# ---------------------------------------------------------------------------
# Tools
# ---------------------------------------------------------------------------


@function_tool
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


@function_tool
def current_datetime() -> str:
    """Get the current date and time in ISO format."""
    return datetime.now(tz=timezone.utc).isoformat()


# ---------------------------------------------------------------------------
# LLM model (chat completions for broad provider compatibility)
# ---------------------------------------------------------------------------


def create_model() -> OpenAIChatCompletionsModel | None:
    """Create a chat completions model compatible with any OpenAI-compatible API."""
    if not OPENAI_API_KEY:
        return None
    client = AsyncOpenAI(api_key=OPENAI_API_KEY, base_url=OPENAI_BASE_URL)
    return OpenAIChatCompletionsModel(model=OPENAI_MODEL, openai_client=client)


# ---------------------------------------------------------------------------
# Models
# ---------------------------------------------------------------------------


class RunRequest(BaseModel):
    message: str


class RunResponse(BaseModel):
    response: str


# ---------------------------------------------------------------------------
# Application
# ---------------------------------------------------------------------------


@asynccontextmanager
async def lifespan(_app: FastAPI):
    """Build agents on startup."""
    model = create_model()
    _app.state.llm_configured = model is not None

    if model is not None:
        _app.state.math_agent = Agent(
            name="Math Specialist",
            instructions=(
                "You are a math specialist. Use the calculator tool to solve "
                "mathematical problems. Show your work step by step."
            ),
            tools=[calculator],
            model=model,
        )
        _app.state.primary_agent = Agent(
            name="Assistant",
            instructions=(
                "You are a helpful assistant. For math questions, hand off to the "
                "Math Specialist. Use the current_datetime tool when asked about "
                "the current time or date."
            ),
            tools=[current_datetime],
            handoffs=[handoff(_app.state.math_agent)],
            model=model,
        )
    yield


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


@app.get("/health")
async def health():
    """Check if the agents are ready."""
    return {
        "status": "ok" if app.state.llm_configured else "degraded",
        "llm_configured": app.state.llm_configured,
        "agents": ["assistant", "math-specialist"],
    }


@app.post("/run", response_model=RunResponse)
async def run(req: RunRequest):
    """Run the agent pipeline with automatic handoffs."""
    if not app.state.llm_configured:
        return RunResponse(
            response="[No API key configured] Set OPENAI_API_KEY to enable the agents.",
        )
    result = await Runner.run(app.state.primary_agent, req.message)
    return RunResponse(response=result.final_output)


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
