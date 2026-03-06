"""{{.Name}} — SSE streaming chatbot with session memory, deployed with Volra."""

import os
import uuid
from contextlib import asynccontextmanager

from fastapi import FastAPI, Request
from openai import OpenAI
from pydantic import BaseModel
from sse_starlette.sse import EventSourceResponse

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")

SESSIONS: dict[str, list[dict]] = {}
MAX_HISTORY = 20


def get_client() -> OpenAI | None:
    if not OPENAI_API_KEY:
        return None
    return OpenAI(api_key=OPENAI_API_KEY, base_url=OPENAI_BASE_URL)


@asynccontextmanager
async def lifespan(_app: FastAPI):
    _app.state.client = get_client()
    yield


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


class ChatRequest(BaseModel):
    message: str
    session_id: str | None = None


class ChatResponse(BaseModel):
    response: str
    session_id: str


@app.get("/health")
async def health():
    return {
        "status": "ok" if app.state.client else "degraded",
        "llm_configured": app.state.client is not None,
    }


@app.post("/chat", response_model=ChatResponse)
async def chat(req: ChatRequest):
    """Send a message and get a response."""
    sid = req.session_id or str(uuid.uuid4())
    history = SESSIONS.setdefault(sid, [])
    history.append({"role": "user", "content": req.message})

    if app.state.client is None:
        reply = "[No API key] Set OPENAI_API_KEY to enable chat."
    else:
        resp = app.state.client.chat.completions.create(
            model=OPENAI_MODEL,
            messages=history[-MAX_HISTORY:],
        )
        reply = resp.choices[0].message.content

    history.append({"role": "assistant", "content": reply})
    return ChatResponse(response=reply, session_id=sid)


@app.get("/chat/stream")
async def chat_stream(message: str, session_id: str | None = None):
    """Stream a response via Server-Sent Events."""
    sid = session_id or str(uuid.uuid4())
    history = SESSIONS.setdefault(sid, [])
    history.append({"role": "user", "content": message})

    async def generate():
        if app.state.client is None:
            yield {"data": "[No API key] Set OPENAI_API_KEY to enable chat."}
            return
        stream = app.state.client.chat.completions.create(
            model=OPENAI_MODEL,
            messages=history[-MAX_HISTORY:],
            stream=True,
        )
        full_reply = ""
        for chunk in stream:
            delta = chunk.choices[0].delta.content
            if delta:
                full_reply += delta
                yield {"data": delta}
        history.append({"role": "assistant", "content": full_reply})

    return EventSourceResponse(generate())


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
