"""{{.Name}} — Research agent with web scraping, deployed with Volra."""

import asyncio
import os
from contextlib import asynccontextmanager

from crewai import Agent, Crew, Process, Task
from crewai.tools import tool
from fastapi import FastAPI
from pydantic import BaseModel

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
LLM_MODEL = os.getenv("CREWAI_LLM", "openai/gpt-4o-mini")

if OPENAI_BASE_URL:
    os.environ.setdefault("OPENAI_API_BASE", OPENAI_BASE_URL)


# ---------------------------------------------------------------------------
# Tools
# ---------------------------------------------------------------------------


@tool("web_search")
def web_search(query: str) -> str:
    """Search the web for information. Stub — replace with a real API.

    Args:
        query: The search query.
    """
    return f"[Search stub] No results for '{query}'. Replace with SerperDevTool."


@tool("fetch_url")
def fetch_url(url: str) -> str:
    """Fetch content from a URL.

    Args:
        url: The URL to fetch.
    """
    import httpx

    try:
        resp = httpx.get(url, timeout=10, follow_redirects=True)
        resp.raise_for_status()
        text = resp.text[:3000]
        return text
    except Exception as exc:
        return f"Error fetching {url}: {exc}"


# ---------------------------------------------------------------------------
# Crew
# ---------------------------------------------------------------------------


def run_research(topic: str) -> str:
    researcher = Agent(
        role="Research Analyst",
        goal=f"Research '{topic}' thoroughly and produce a comprehensive report",
        backstory="Expert research analyst who finds and synthesizes information.",
        tools=[web_search, fetch_url],
        llm=LLM_MODEL,
        verbose=False,
    )

    task = Task(
        description=f"Research the topic: {topic}. Find key facts, trends, and insights.",
        expected_output="A structured research report with sections, key findings, and sources",
        agent=researcher,
    )

    crew = Crew(
        agents=[researcher],
        tasks=[task],
        process=Process.sequential,
        verbose=False,
    )
    return str(crew.kickoff())


# ---------------------------------------------------------------------------
# Application
# ---------------------------------------------------------------------------


@asynccontextmanager
async def lifespan(_app: FastAPI):
    _app.state.llm_configured = bool(OPENAI_API_KEY)
    yield


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


class ResearchRequest(BaseModel):
    topic: str


class ResearchResponse(BaseModel):
    result: str
    topic: str


@app.get("/health")
async def health():
    return {
        "status": "ok" if app.state.llm_configured else "degraded",
        "llm_configured": app.state.llm_configured,
    }


@app.post("/research", response_model=ResearchResponse)
async def research(req: ResearchRequest):
    if not app.state.llm_configured:
        return ResearchResponse(
            result="[No API key] Set OPENAI_API_KEY.", topic=req.topic
        )
    result = await asyncio.to_thread(run_research, req.topic)
    return ResearchResponse(result=result, topic=req.topic)


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
