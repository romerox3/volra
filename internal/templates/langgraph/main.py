"""{{.Name}} — LangGraph ReAct agent with tool use, deployed with Volra."""

import os
from contextlib import asynccontextmanager
from datetime import datetime, timezone

from fastapi import FastAPI
from langchain_core.messages import HumanMessage
from langchain_core.tools import tool
from langchain_openai import ChatOpenAI
from langgraph.checkpoint.memory import MemorySaver
from langgraph.graph import END, MessagesState, StateGraph, START
from langgraph.prebuilt import ToolNode, tools_condition
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


@tool
def current_datetime() -> str:
    """Get the current date and time in ISO format."""
    return datetime.now(tz=timezone.utc).isoformat()


@tool
def web_search(query: str) -> str:
    """Search the web for information. This is a stub — replace with a real search API.

    Args:
        query: The search query.
    """
    return (
        f"[Search stub] No real results for: '{query}'. "
        "Replace this tool with a real search API (Tavily, Serper, etc.)."
    )


TOOLS = [calculator, current_datetime, web_search]

# ---------------------------------------------------------------------------
# Graph
# ---------------------------------------------------------------------------


def build_agent():
    """Build and compile the LangGraph ReAct agent."""
    llm = ChatOpenAI(model=OPENAI_MODEL, base_url=OPENAI_BASE_URL).bind_tools(TOOLS)

    def call_model(state: MessagesState) -> dict:
        return {"messages": [llm.invoke(state["messages"])]}

    graph = StateGraph(MessagesState)
    graph.add_node("agent", call_model)
    graph.add_node("tools", ToolNode(TOOLS))
    graph.add_edge(START, "agent")
    graph.add_conditional_edges("agent", tools_condition)
    graph.add_edge("tools", "agent")

    return graph.compile(checkpointer=MemorySaver())


# ---------------------------------------------------------------------------
# Models
# ---------------------------------------------------------------------------


class InvokeRequest(BaseModel):
    message: str
    thread_id: str = "default"


class InvokeResponse(BaseModel):
    response: str
    thread_id: str


# ---------------------------------------------------------------------------
# Application
# ---------------------------------------------------------------------------


@asynccontextmanager
async def lifespan(_app: FastAPI):
    """Build the agent graph on startup."""
    if OPENAI_API_KEY:
        _app.state.agent = build_agent()
    else:
        _app.state.agent = None
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


@app.post("/invoke", response_model=InvokeResponse)
async def invoke(req: InvokeRequest):
    """Invoke the agent with a message. Supports thread-based memory."""
    if app.state.agent is None:
        return InvokeResponse(
            response="[No API key configured] Set OPENAI_API_KEY to enable the agent.",
            thread_id=req.thread_id,
        )
    config = {"configurable": {"thread_id": req.thread_id}}
    result = app.state.agent.invoke(
        {"messages": [HumanMessage(content=req.message)]},
        config,
    )
    last_message = result["messages"][-1]
    return InvokeResponse(
        response=last_message.content,
        thread_id=req.thread_id,
    )


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
