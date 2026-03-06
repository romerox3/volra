"""{{.Name}} — Multi-agent group chat with AutoGen, deployed with Volra."""

import asyncio
import os
from contextlib import asynccontextmanager

from autogen_agentchat.agents import AssistantAgent
from autogen_agentchat.conditions import TextMentionTermination
from autogen_agentchat.teams import RoundRobinGroupChat
from autogen_ext.models.openai import OpenAIChatCompletionClient
from fastapi import FastAPI
from pydantic import BaseModel

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")


def create_model_client() -> OpenAIChatCompletionClient | None:
    if not OPENAI_API_KEY:
        return None
    kwargs = {"model": OPENAI_MODEL, "api_key": OPENAI_API_KEY}
    if OPENAI_BASE_URL:
        kwargs["base_url"] = OPENAI_BASE_URL
    return OpenAIChatCompletionClient(**kwargs)


async def run_group(topic: str, client: OpenAIChatCompletionClient) -> str:
    planner = AssistantAgent(
        name="planner",
        model_client=client,
        system_message=(
            "You are a planner. Break down problems into actionable steps. "
            "When the group reaches consensus, say CONSENSUS."
        ),
    )
    coder = AssistantAgent(
        name="coder",
        model_client=client,
        system_message=(
            "You are a coder. Propose implementations for the planner's steps. "
            "When the plan is approved, say CONSENSUS."
        ),
    )
    critic = AssistantAgent(
        name="critic",
        model_client=client,
        system_message=(
            "You are a critic. Evaluate proposals for flaws and suggest improvements. "
            "When satisfied, say CONSENSUS."
        ),
    )

    termination = TextMentionTermination("CONSENSUS")
    team = RoundRobinGroupChat(
        [planner, coder, critic], termination_condition=termination, max_turns=9
    )
    result = await team.run(task=topic)
    messages = [m.content for m in result.messages if hasattr(m, "content")]
    return messages[-1] if messages else "No output"


@asynccontextmanager
async def lifespan(_app: FastAPI):
    _app.state.model_client = create_model_client()
    yield


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


class DiscussRequest(BaseModel):
    topic: str


class DiscussResponse(BaseModel):
    result: str
    topic: str


@app.get("/health")
async def health():
    configured = app.state.model_client is not None
    return {
        "status": "ok" if configured else "degraded",
        "llm_configured": configured,
        "agents": ["planner", "coder", "critic"],
    }


@app.post("/discuss", response_model=DiscussResponse)
async def discuss(req: DiscussRequest):
    if app.state.model_client is None:
        return DiscussResponse(result="[No API key] Set OPENAI_API_KEY.", topic=req.topic)
    result = await run_group(req.topic, app.state.model_client)
    return DiscussResponse(result=result, topic=req.topic)


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
