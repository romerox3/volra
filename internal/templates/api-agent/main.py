"""{{.Name}} — function-calling agent without any framework, deployed with Volra."""

import json
import math
import os
from contextlib import asynccontextmanager

import httpx
from fastapi import FastAPI
from openai import OpenAI
from pydantic import BaseModel

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")
MAX_TOOL_STEPS = 5

# ---------------------------------------------------------------------------
# Tool definitions (OpenAI function-calling schema)
# ---------------------------------------------------------------------------

TOOL_SCHEMAS = [
    {
        "type": "function",
        "function": {
            "name": "calculator",
            "description": "Evaluate a mathematical expression. Supports basic arithmetic and math functions.",
            "parameters": {
                "type": "object",
                "properties": {"expression": {"type": "string", "description": "e.g. '2 + 2' or 'math.sqrt(144)'"}},
                "required": ["expression"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "weather",
            "description": "Get the current weather for a city. This is a stub — replace with a real API.",
            "parameters": {
                "type": "object",
                "properties": {"city": {"type": "string", "description": "City name"}},
                "required": ["city"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "web_fetch",
            "description": "Fetch the content of a URL and return the first 500 characters.",
            "parameters": {
                "type": "object",
                "properties": {"url": {"type": "string", "description": "Full URL to fetch"}},
                "required": ["url"],
            },
        },
    },
]

# ---------------------------------------------------------------------------
# Tool implementations
# ---------------------------------------------------------------------------


def tool_calculator(expression: str) -> str:
    """Evaluate a math expression safely."""
    try:
        result = eval(expression, {"__builtins__": {}, "math": math})  # noqa: S307
    except Exception as exc:
        return f"Error: {exc}"
    return str(result)


def tool_weather(city: str) -> str:
    """Return stub weather data."""
    return f"Weather in {city}: 22°C, sunny (stub — replace with a real weather API)"


def tool_web_fetch(url: str) -> str:
    """Fetch a URL and return truncated content."""
    try:
        resp = httpx.get(url, timeout=5, follow_redirects=True)
        return resp.text[:500]
    except httpx.HTTPError as exc:
        return f"Error fetching {url}: {exc}"


TOOL_IMPLEMENTATIONS = {
    "calculator": tool_calculator,
    "weather": tool_weather,
    "web_fetch": tool_web_fetch,
}

# ---------------------------------------------------------------------------
# Agent loop
# ---------------------------------------------------------------------------


def run_agent(client: OpenAI, message: str) -> str:
    """Execute the tool-calling loop until the LLM produces a final response."""
    messages = [{"role": "user", "content": message}]

    for _ in range(MAX_TOOL_STEPS):
        response = client.chat.completions.create(
            model=OPENAI_MODEL,
            messages=messages,
            tools=TOOL_SCHEMAS,
        )
        assistant_msg = response.choices[0].message

        if not assistant_msg.tool_calls:
            return assistant_msg.content or ""

        messages.append(assistant_msg)
        for call in assistant_msg.tool_calls:
            fn = TOOL_IMPLEMENTATIONS[call.function.name]
            args = json.loads(call.function.arguments)
            result = fn(**args)
            messages.append({
                "role": "tool",
                "tool_call_id": call.id,
                "content": result,
            })

    last = messages[-1]
    return last.get("content", "") if isinstance(last, dict) else last.content


# ---------------------------------------------------------------------------
# Models
# ---------------------------------------------------------------------------


class AskRequest(BaseModel):
    message: str


class AskResponse(BaseModel):
    response: str


# ---------------------------------------------------------------------------
# Application
# ---------------------------------------------------------------------------


@asynccontextmanager
async def lifespan(_app: FastAPI):
    """Initialize OpenAI client on startup."""
    _app.state.openai = OpenAI(api_key=OPENAI_API_KEY, base_url=OPENAI_BASE_URL) if OPENAI_API_KEY else None
    yield


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


@app.get("/health")
async def health():
    """Check if the agent is ready."""
    llm_configured = app.state.openai is not None
    return {
        "status": "ok" if llm_configured else "degraded",
        "llm_configured": llm_configured,
        "tools": [t["function"]["name"] for t in TOOL_SCHEMAS],
    }


@app.post("/ask", response_model=AskResponse)
async def ask(req: AskRequest):
    """Run the agent with a message. The agent will use tools as needed."""
    if app.state.openai is None:
        return AskResponse(
            response="[No API key configured] Set OPENAI_API_KEY to enable the agent.",
        )
    result = run_agent(app.state.openai, req.message)
    return AskResponse(response=result)


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
