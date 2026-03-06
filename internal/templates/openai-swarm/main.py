"""{{.Name}} — Multi-agent handoffs via function calling, deployed with Volra."""

import json
import os
from contextlib import asynccontextmanager

from fastapi import FastAPI
from openai import OpenAI
from pydantic import BaseModel

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")

# ---------------------------------------------------------------------------
# Agent definitions
# ---------------------------------------------------------------------------

AGENTS = {
    "triage": {
        "system": (
            "You are a triage agent. Route the user to the right specialist:\n"
            "- For technical questions, call transfer_to_technical\n"
            "- For billing questions, call transfer_to_billing\n"
            "- For general questions, answer directly"
        ),
        "tools": [
            {
                "type": "function",
                "function": {
                    "name": "transfer_to_technical",
                    "description": "Transfer to technical support agent",
                    "parameters": {"type": "object", "properties": {}},
                },
            },
            {
                "type": "function",
                "function": {
                    "name": "transfer_to_billing",
                    "description": "Transfer to billing support agent",
                    "parameters": {"type": "object", "properties": {}},
                },
            },
        ],
    },
    "technical": {
        "system": "You are a technical support agent. Help with technical issues. Be concise.",
        "tools": [],
    },
    "billing": {
        "system": "You are a billing agent. Help with invoices, payments, and subscriptions. Be concise.",
        "tools": [],
    },
}

TRANSFER_MAP = {
    "transfer_to_technical": "technical",
    "transfer_to_billing": "billing",
}


# ---------------------------------------------------------------------------
# Orchestrator
# ---------------------------------------------------------------------------


def run_swarm(client: OpenAI, message: str) -> tuple[str, str]:
    """Run the swarm orchestration loop. Returns (response, agent_name)."""
    current_agent = "triage"
    messages = [{"role": "user", "content": message}]

    for _ in range(5):  # max handoff depth
        agent_def = AGENTS[current_agent]
        resp = client.chat.completions.create(
            model=OPENAI_MODEL,
            messages=[{"role": "system", "content": agent_def["system"]}] + messages,
            tools=agent_def["tools"] or None,
        )
        choice = resp.choices[0]

        if choice.finish_reason == "tool_calls":
            tool_call = choice.message.tool_calls[0]
            next_agent = TRANSFER_MAP.get(tool_call.function.name)
            if next_agent:
                current_agent = next_agent
                continue

        return choice.message.content or "", current_agent

    return "Max handoff depth reached.", current_agent


# ---------------------------------------------------------------------------
# Application
# ---------------------------------------------------------------------------


@asynccontextmanager
async def lifespan(_app: FastAPI):
    if OPENAI_API_KEY:
        _app.state.client = OpenAI(api_key=OPENAI_API_KEY, base_url=OPENAI_BASE_URL)
    else:
        _app.state.client = None
    yield


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


class ChatRequest(BaseModel):
    message: str


class ChatResponse(BaseModel):
    response: str
    agent: str


@app.get("/health")
async def health():
    configured = app.state.client is not None
    return {
        "status": "ok" if configured else "degraded",
        "llm_configured": configured,
        "agents": list(AGENTS.keys()),
    }


@app.post("/chat", response_model=ChatResponse)
async def chat(req: ChatRequest):
    if app.state.client is None:
        return ChatResponse(response="[No API key] Set OPENAI_API_KEY.", agent="none")
    response, agent = run_swarm(app.state.client, req.message)
    return ChatResponse(response=response, agent=agent)


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
