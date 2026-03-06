"""{{.Name}} — LangChain chatbot with conversation memory, deployed with Volra."""

import os
from contextlib import asynccontextmanager

from fastapi import FastAPI
from langchain.chains import ConversationChain
from langchain.memory import ConversationBufferWindowMemory
from langchain_openai import ChatOpenAI
from pydantic import BaseModel

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")

SESSIONS: dict[str, ConversationChain] = {}


def create_chain() -> ConversationChain | None:
    if not OPENAI_API_KEY:
        return None
    llm = ChatOpenAI(model=OPENAI_MODEL, base_url=OPENAI_BASE_URL)
    memory = ConversationBufferWindowMemory(k=10)
    return ConversationChain(llm=llm, memory=memory, verbose=False)


@asynccontextmanager
async def lifespan(_app: FastAPI):
    _app.state.llm_configured = bool(OPENAI_API_KEY)
    yield


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


class ChatRequest(BaseModel):
    message: str
    session_id: str = "default"


class ChatResponse(BaseModel):
    response: str
    session_id: str


@app.get("/health")
async def health():
    return {
        "status": "ok" if app.state.llm_configured else "degraded",
        "llm_configured": app.state.llm_configured,
    }


@app.post("/chat", response_model=ChatResponse)
async def chat(req: ChatRequest):
    """Chat with conversation memory per session."""
    if not app.state.llm_configured:
        return ChatResponse(
            response="[No API key] Set OPENAI_API_KEY to enable chat.",
            session_id=req.session_id,
        )
    if req.session_id not in SESSIONS:
        SESSIONS[req.session_id] = create_chain()
    chain = SESSIONS[req.session_id]
    result = chain.invoke({"input": req.message})
    return ChatResponse(response=result["response"], session_id=req.session_id)


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
