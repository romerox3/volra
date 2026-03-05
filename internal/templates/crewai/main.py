"""{{.Name}} — CrewAI multi-agent research crew, deployed with Volra."""

import asyncio
import os
from contextlib import asynccontextmanager

from crewai import Agent, Crew, Process, Task
from crewai.tools import tool
from fastapi import FastAPI
from pydantic import BaseModel

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
LLM_MODEL = os.getenv("CREWAI_LLM", "openai/gpt-4o-mini")

# Propagate base URL for LiteLLM (used by CrewAI internally)
if OPENAI_BASE_URL:
    os.environ.setdefault("OPENAI_API_BASE", OPENAI_BASE_URL)

# ---------------------------------------------------------------------------
# Tools
# ---------------------------------------------------------------------------


@tool("web_search")
def web_search(query: str) -> str:
    """Search the web for information. This is a stub — replace with a real API.

    Args:
        query: The search query.
    """
    return (
        f"[Search stub] Results for '{query}': "
        "No real results. Replace this tool with SerperDevTool or similar."
    )


# ---------------------------------------------------------------------------
# Agents
# ---------------------------------------------------------------------------


def create_researcher() -> Agent:
    """Create the research analyst agent."""
    return Agent(
        role="Research Analyst",
        goal="Find accurate, relevant information on the given topic",
        backstory=(
            "You are an expert research analyst with a keen eye for detail. "
            "You gather comprehensive information and identify key insights."
        ),
        tools=[web_search],
        llm=LLM_MODEL,
        verbose=False,
    )


def create_writer() -> Agent:
    """Create the content writer agent."""
    return Agent(
        role="Content Writer",
        goal="Write a clear, well-structured summary based on the research",
        backstory=(
            "You are a skilled technical writer who transforms complex research "
            "into readable, actionable summaries."
        ),
        llm=LLM_MODEL,
        verbose=False,
    )


# ---------------------------------------------------------------------------
# Crew
# ---------------------------------------------------------------------------


def run_research_crew(topic: str) -> str:
    """Assemble and execute the research crew for a given topic."""
    researcher = create_researcher()
    writer = create_writer()

    research_task = Task(
        description=f"Research the following topic thoroughly: {topic}",
        expected_output="Key findings as a structured list of bullet points",
        agent=researcher,
    )
    writing_task = Task(
        description="Write a concise summary based on the research findings",
        expected_output="A 2-3 paragraph summary in clear, professional language",
        agent=writer,
    )

    crew = Crew(
        agents=[researcher, writer],
        tasks=[research_task, writing_task],
        process=Process.sequential,
        verbose=False,
    )

    result = crew.kickoff()
    return str(result)


# ---------------------------------------------------------------------------
# Models
# ---------------------------------------------------------------------------


class KickoffRequest(BaseModel):
    topic: str


class KickoffResponse(BaseModel):
    result: str
    topic: str


# ---------------------------------------------------------------------------
# Application
# ---------------------------------------------------------------------------


@asynccontextmanager
async def lifespan(_app: FastAPI):
    """Verify configuration on startup."""
    _app.state.llm_configured = bool(OPENAI_API_KEY)
    yield


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


@app.get("/health")
async def health():
    """Check if the crew is ready to run."""
    return {
        "status": "ok" if app.state.llm_configured else "degraded",
        "llm_configured": app.state.llm_configured,
        "agents": ["researcher", "writer"],
    }


@app.post("/kickoff", response_model=KickoffResponse)
async def kickoff(req: KickoffRequest):
    """Run the research crew on a topic. This may take 30-60 seconds."""
    if not app.state.llm_configured:
        return KickoffResponse(
            result="[No API key configured] Set OPENAI_API_KEY to enable the crew.",
            topic=req.topic,
        )
    result = await asyncio.to_thread(run_research_crew, req.topic)
    return KickoffResponse(result=result, topic=req.topic)


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
