"""{{.Name}} — OpenAI Assistants API agent, deployed with Volra."""

import os
import time
from contextlib import asynccontextmanager

from fastapi import FastAPI
from openai import OpenAI
from pydantic import BaseModel

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")

ASSISTANT_INSTRUCTIONS = (
    "You are a helpful assistant. You can analyze data, write code, "
    "and answer questions. Use code interpreter when calculations are needed."
)


@asynccontextmanager
async def lifespan(_app: FastAPI):
    if OPENAI_API_KEY:
        client = OpenAI(api_key=OPENAI_API_KEY, base_url=OPENAI_BASE_URL)
        assistant = client.beta.assistants.create(
            name="{{.Name}}",
            instructions=ASSISTANT_INSTRUCTIONS,
            model=OPENAI_MODEL,
            tools=[{"type": "code_interpreter"}],
        )
        _app.state.client = client
        _app.state.assistant_id = assistant.id
    else:
        _app.state.client = None
        _app.state.assistant_id = None
    yield


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


class AskRequest(BaseModel):
    message: str
    thread_id: str | None = None


class AskResponse(BaseModel):
    response: str
    thread_id: str


@app.get("/health")
async def health():
    configured = app.state.client is not None
    return {
        "status": "ok" if configured else "degraded",
        "llm_configured": configured,
    }


@app.post("/ask", response_model=AskResponse)
async def ask(req: AskRequest):
    if app.state.client is None:
        return AskResponse(response="[No API key] Set OPENAI_API_KEY.", thread_id="")

    client = app.state.client
    if req.thread_id:
        thread_id = req.thread_id
    else:
        thread = client.beta.threads.create()
        thread_id = thread.id

    client.beta.threads.messages.create(
        thread_id=thread_id, role="user", content=req.message
    )
    run = client.beta.threads.runs.create(
        thread_id=thread_id, assistant_id=app.state.assistant_id
    )

    while run.status in ("queued", "in_progress"):
        time.sleep(0.5)
        run = client.beta.threads.runs.retrieve(thread_id=thread_id, run_id=run.id)

    if run.status != "completed":
        return AskResponse(response=f"Run failed: {run.status}", thread_id=thread_id)

    messages = client.beta.threads.messages.list(thread_id=thread_id, limit=1)
    reply = messages.data[0].content[0].text.value
    return AskResponse(response=reply, thread_id=thread_id)


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
