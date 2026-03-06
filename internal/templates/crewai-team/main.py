"""{{.Name}} — 3-agent dev team with CrewAI, deployed with Volra."""

import asyncio
import os
from contextlib import asynccontextmanager

from crewai import Agent, Crew, Process, Task
from fastapi import FastAPI
from pydantic import BaseModel

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
LLM_MODEL = os.getenv("CREWAI_LLM", "openai/gpt-4o-mini")

if OPENAI_BASE_URL:
    os.environ.setdefault("OPENAI_API_BASE", OPENAI_BASE_URL)


# ---------------------------------------------------------------------------
# Agents
# ---------------------------------------------------------------------------


def create_pm() -> Agent:
    return Agent(
        role="Product Manager",
        goal="Break down the task into clear requirements and acceptance criteria",
        backstory="You are a seasoned PM who writes crisp specs that developers love.",
        llm=LLM_MODEL,
        verbose=False,
    )


def create_developer() -> Agent:
    return Agent(
        role="Software Developer",
        goal="Implement the solution based on the PM's requirements",
        backstory="You are a senior developer who writes clean, tested code.",
        llm=LLM_MODEL,
        verbose=False,
    )


def create_qa() -> Agent:
    return Agent(
        role="QA Engineer",
        goal="Review the implementation for bugs, edge cases, and improvements",
        backstory="You are a meticulous QA engineer who catches issues others miss.",
        llm=LLM_MODEL,
        verbose=False,
    )


# ---------------------------------------------------------------------------
# Crew
# ---------------------------------------------------------------------------


def run_dev_crew(task_description: str) -> str:
    pm = create_pm()
    dev = create_developer()
    qa = create_qa()

    spec_task = Task(
        description=f"Create requirements for: {task_description}",
        expected_output="Requirements with acceptance criteria as bullet points",
        agent=pm,
    )
    impl_task = Task(
        description="Implement the solution based on the requirements",
        expected_output="Implementation plan or pseudocode with key decisions explained",
        agent=dev,
    )
    review_task = Task(
        description="Review the implementation for quality, bugs, and edge cases",
        expected_output="Review report with findings and suggestions",
        agent=qa,
    )

    crew = Crew(
        agents=[pm, dev, qa],
        tasks=[spec_task, impl_task, review_task],
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


class KickoffRequest(BaseModel):
    task: str


class KickoffResponse(BaseModel):
    result: str
    task: str


@app.get("/health")
async def health():
    return {
        "status": "ok" if app.state.llm_configured else "degraded",
        "llm_configured": app.state.llm_configured,
        "agents": ["pm", "developer", "qa"],
    }


@app.post("/kickoff", response_model=KickoffResponse)
async def kickoff(req: KickoffRequest):
    if not app.state.llm_configured:
        return KickoffResponse(
            result="[No API key] Set OPENAI_API_KEY.", task=req.task
        )
    result = await asyncio.to_thread(run_dev_crew, req.task)
    return KickoffResponse(result=result, task=req.task)


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
