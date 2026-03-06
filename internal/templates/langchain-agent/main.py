"""{{.Name}} — LangChain ReAct agent with tool calling, deployed with Volra."""

import os
from contextlib import asynccontextmanager

from fastapi import FastAPI
from langchain import hub
from langchain.agents import AgentExecutor, create_react_agent
from langchain_core.tools import tool
from langchain_openai import ChatOpenAI
from pydantic import BaseModel

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")


@tool
def calculator(expression: str) -> str:
    """Evaluate a math expression safely.

    Args:
        expression: A math expression like '2 + 2'.
    """
    allowed = set("0123456789+-*/.() ")
    if not all(c in allowed for c in expression):
        return "Error: invalid characters"
    try:
        result = eval(expression, {"__builtins__": {}})  # noqa: S307
    except Exception as exc:
        return f"Error: {exc}"
    return str(result)


@tool
def web_search(query: str) -> str:
    """Search the web. Stub — replace with a real search API.

    Args:
        query: The search query.
    """
    return f"[Search stub] No results for '{query}'. Replace with Tavily/Serper."


TOOLS = [calculator, web_search]


def build_agent() -> AgentExecutor | None:
    if not OPENAI_API_KEY:
        return None
    llm = ChatOpenAI(model=OPENAI_MODEL, base_url=OPENAI_BASE_URL)
    prompt = hub.pull("hwchase17/react")
    agent = create_react_agent(llm, TOOLS, prompt)
    return AgentExecutor(agent=agent, tools=TOOLS, verbose=False, handle_parsing_errors=True)


@asynccontextmanager
async def lifespan(_app: FastAPI):
    _app.state.agent = build_agent()
    yield


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


class InvokeRequest(BaseModel):
    message: str


class InvokeResponse(BaseModel):
    response: str


@app.get("/health")
async def health():
    return {
        "status": "ok" if app.state.agent else "degraded",
        "llm_configured": app.state.agent is not None,
        "tools": [t.name for t in TOOLS],
    }


@app.post("/invoke", response_model=InvokeResponse)
async def invoke(req: InvokeRequest):
    if app.state.agent is None:
        return InvokeResponse(response="[No API key] Set OPENAI_API_KEY.")
    result = app.state.agent.invoke({"input": req.message})
    return InvokeResponse(response=result["output"])


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
