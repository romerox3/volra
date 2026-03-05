"""Conversational Agent — sessions in Postgres, cache in Redis, mock LLM."""

import json
import os
import time
import uuid

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

import psycopg2
import redis

app = FastAPI(title="Conversational Agent")

PG_HOST = os.getenv("PG_HOST", "conv-agent-postgres")
PG_PORT = int(os.getenv("PG_PORT", "5432"))
PG_PASSWORD = os.getenv("POSTGRES_PASSWORD", "volra_test")
PG_DB = os.getenv("POSTGRES_DB", "conversations")

REDIS_HOST = os.getenv("REDIS_HOST", "conv-agent-redis")
REDIS_PORT = int(os.getenv("REDIS_PORT", "6379"))

_pg_conn = None
_redis_client: redis.Redis | None = None


def get_pg():
    """Get Postgres connection with retry for startup race condition."""
    global _pg_conn
    if _pg_conn is not None and not _pg_conn.closed:
        return _pg_conn

    for attempt in range(30):
        try:
            _pg_conn = psycopg2.connect(
                host=PG_HOST,
                port=PG_PORT,
                user="postgres",
                password=PG_PASSWORD,
                dbname=PG_DB,
            )
            _pg_conn.autocommit = True
            return _pg_conn
        except psycopg2.OperationalError:
            time.sleep(1)
    raise RuntimeError("Cannot connect to Postgres after 30 retries")


def get_redis() -> redis.Redis:
    global _redis_client
    if _redis_client is None:
        _redis_client = redis.Redis(
            host=REDIS_HOST, port=REDIS_PORT, decode_responses=True
        )
    return _redis_client


def init_db():
    conn = get_pg()
    with conn.cursor() as cur:
        cur.execute("""
            CREATE TABLE IF NOT EXISTS sessions (
                id TEXT PRIMARY KEY,
                created_at TIMESTAMP DEFAULT NOW()
            )
        """)
        cur.execute("""
            CREATE TABLE IF NOT EXISTS messages (
                id SERIAL PRIMARY KEY,
                session_id TEXT REFERENCES sessions(id),
                role TEXT NOT NULL,
                content TEXT NOT NULL,
                created_at TIMESTAMP DEFAULT NOW()
            )
        """)


class ChatRequest(BaseModel):
    session_id: str | None = None
    message: str


class ChatResponse(BaseModel):
    session_id: str
    response: str
    history_length: int


def mock_llm_response(message: str) -> str:
    """Mock LLM response based on keywords."""
    lower = message.lower()
    if "hello" in lower or "hi" in lower:
        return "Hello! How can I help you today?"
    if "help" in lower:
        return "I'm a conversational agent. Ask me anything!"
    if "bye" in lower:
        return "Goodbye! Have a great day!"
    return f"I received your message: '{message}'. This is a mock response."


@app.on_event("startup")
async def startup():
    init_db()


@app.get("/health")
async def health():
    pg_ok = False
    redis_ok = False
    try:
        conn = get_pg()
        with conn.cursor() as cur:
            cur.execute("SELECT 1")
        pg_ok = True
    except Exception:
        pass
    try:
        r = get_redis()
        redis_ok = r.ping()
    except Exception:
        pass
    return {
        "status": "ok" if (pg_ok and redis_ok) else "degraded",
        "postgres_connected": pg_ok,
        "redis_connected": redis_ok,
    }


@app.post("/chat", response_model=ChatResponse)
async def chat(req: ChatRequest):
    conn = get_pg()
    r = get_redis()

    session_id = req.session_id or str(uuid.uuid4())

    # Create session if new.
    with conn.cursor() as cur:
        cur.execute(
            "INSERT INTO sessions (id) VALUES (%s) ON CONFLICT DO NOTHING",
            (session_id,),
        )
        # Store user message.
        cur.execute(
            "INSERT INTO messages (session_id, role, content) VALUES (%s, %s, %s)",
            (session_id, "user", req.message),
        )

    # Check response cache.
    cache_key = f"chat:{session_id}:{req.message}"
    cached = r.get(cache_key)
    if cached:
        response = cached
    else:
        response = mock_llm_response(req.message)
        r.setex(cache_key, 600, response)

    # Store assistant message.
    with conn.cursor() as cur:
        cur.execute(
            "INSERT INTO messages (session_id, role, content) VALUES (%s, %s, %s)",
            (session_id, "assistant", response),
        )
        cur.execute(
            "SELECT COUNT(*) FROM messages WHERE session_id = %s", (session_id,)
        )
        count = cur.fetchone()[0]

    return ChatResponse(
        session_id=session_id,
        response=response,
        history_length=count,
    )


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
