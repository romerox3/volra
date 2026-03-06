"""{{.Name}} — Two-agent coder + reviewer with AutoGen, deployed with Volra."""

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


async def run_duo(problem: str, client: OpenAIChatCompletionClient) -> str:
    coder = AssistantAgent(
        name="coder",
        model_client=client,
        system_message=(
            "You are an expert coder. Write clean, correct solutions. "
            "When the reviewer approves, respond with APPROVED."
        ),
    )
    reviewer = AssistantAgent(
        name="reviewer",
        model_client=client,
        system_message=(
            "You are a code reviewer. Check the coder's solution for bugs and improvements. "
            "If the solution is good, respond with APPROVED."
        ),
    )

    termination = TextMentionTermination("APPROVED")
    team = RoundRobinGroupChat(
        [coder, reviewer], termination_condition=termination, max_turns=6
    )
    result = await team.run(task=problem)
    messages = [m.content for m in result.messages if hasattr(m, "content")]
    return messages[-1] if messages else "No output"


@asynccontextmanager
async def lifespan(_app: FastAPI):
    _app.state.model_client = create_model_client()
    yield


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


class SolveRequest(BaseModel):
    problem: str


class SolveResponse(BaseModel):
    result: str
    problem: str


@app.get("/health")
async def health():
    configured = app.state.model_client is not None
    return {
        "status": "ok" if configured else "degraded",
        "llm_configured": configured,
        "agents": ["coder", "reviewer"],
    }


@app.post("/solve", response_model=SolveResponse)
async def solve(req: SolveRequest):
    if app.state.model_client is None:
        return SolveResponse(result="[No API key] Set OPENAI_API_KEY.", problem=req.problem)
    result = await run_duo(req.problem, app.state.model_client)
    return SolveResponse(result=result, problem=req.problem)


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
