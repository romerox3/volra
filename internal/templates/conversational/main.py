"""{{.Name}} — conversational agent with session memory, deployed with Volra."""

import json
import os
import time
import uuid
from contextlib import asynccontextmanager

import psycopg2
import redis
from fastapi import FastAPI
from openai import OpenAI
from pydantic import BaseModel
from sse_starlette.sse import EventSourceResponse

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")

PG_HOST = os.getenv("PG_HOST", "{{.Name}}-postgres")
PG_PORT = int(os.getenv("PG_PORT", "5432"))
PG_PASSWORD = os.getenv("POSTGRES_PASSWORD", "changeme")
PG_DB = os.getenv("POSTGRES_DB", "conversations")

REDIS_HOST = os.getenv("REDIS_HOST", "{{.Name}}-redis")
REDIS_PORT = int(os.getenv("REDIS_PORT", "6379"))

HISTORY_LIMIT = 10
CACHE_TTL = 600

# ---------------------------------------------------------------------------
# Dependencies
# ---------------------------------------------------------------------------


def connect_postgres() -> psycopg2.extensions.connection:
    """Connect to PostgreSQL with retry logic for container startup."""
    max_retries = 30
    for attempt in range(max_retries):
        try:
            conn = psycopg2.connect(
                host=PG_HOST,
                port=PG_PORT,
                user="postgres",
                password=PG_PASSWORD,
                dbname=PG_DB,
            )
            conn.autocommit = True
            return conn
        except psycopg2.OperationalError:
            if attempt == max_retries - 1:
                raise
            time.sleep(1)
    msg = f"Cannot connect to Postgres after {max_retries} retries"
    raise RuntimeError(msg)


def connect_redis() -> redis.Redis:
    """Create a Redis client."""
    return redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)


def create_openai_client() -> OpenAI | None:
    """Create an OpenAI client if API key is configured."""
    if not OPENAI_API_KEY:
        return None
    return OpenAI(api_key=OPENAI_API_KEY, base_url=OPENAI_BASE_URL)


# ---------------------------------------------------------------------------
# Database
# ---------------------------------------------------------------------------


def init_schema(conn: psycopg2.extensions.connection) -> None:
    """Create tables if they don't exist."""
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


def ensure_session(conn: psycopg2.extensions.connection, session_id: str) -> None:
    """Create session if it doesn't exist."""
    with conn.cursor() as cur:
        cur.execute(
            "INSERT INTO sessions (id) VALUES (%s) ON CONFLICT DO NOTHING",
            (session_id,),
        )


def save_message(
    conn: psycopg2.extensions.connection,
    session_id: str,
    role: str,
    content: str,
) -> None:
    """Persist a message to the database."""
    with conn.cursor() as cur:
        cur.execute(
            "INSERT INTO messages (session_id, role, content) VALUES (%s, %s, %s)",
            (session_id, role, content),
        )


def fetch_history(
    conn: psycopg2.extensions.connection,
    session_id: str,
    limit: int = HISTORY_LIMIT,
) -> list[dict[str, str]]:
    """Retrieve recent conversation history for a session."""
    with conn.cursor() as cur:
        cur.execute(
            "SELECT role, content FROM messages "
            "WHERE session_id = %s ORDER BY id DESC LIMIT %s",
            (session_id, limit),
        )
        rows = cur.fetchall()
    return [{"role": r, "content": c} for r, c in reversed(rows)]


def count_messages(conn: psycopg2.extensions.connection, session_id: str) -> int:
    """Count messages in a session."""
    with conn.cursor() as cur:
        cur.execute(
            "SELECT COUNT(*) FROM messages WHERE session_id = %s",
            (session_id,),
        )
        return cur.fetchone()[0]


# ---------------------------------------------------------------------------
# LLM
# ---------------------------------------------------------------------------

SYSTEM_PROMPT = "You are a helpful assistant."


def build_messages(
    user_message: str,
    history: list[dict[str, str]],
) -> list[dict[str, str]]:
    """Build the message list for the LLM call."""
    messages = [{"role": "system", "content": SYSTEM_PROMPT}]
    messages.extend(history)
    messages.append({"role": "user", "content": user_message})
    return messages


def generate_response(
    client: OpenAI | None,
    user_message: str,
    history: list[dict[str, str]],
) -> str:
    """Generate a response using the LLM, or echo if no API key."""
    if client is None:
        return f"[No API key configured] Echo: {user_message}"
    messages = build_messages(user_message, history)
    resp = client.chat.completions.create(model=OPENAI_MODEL, messages=messages)
    return resp.choices[0].message.content


# ---------------------------------------------------------------------------
# Models
# ---------------------------------------------------------------------------


class ChatRequest(BaseModel):
    session_id: str | None = None
    message: str


class ChatResponse(BaseModel):
    session_id: str
    response: str
    history_length: int


# ---------------------------------------------------------------------------
# Application
# ---------------------------------------------------------------------------


@asynccontextmanager
async def lifespan(_app: FastAPI):
    """Initialize database schema on startup."""
    conn = connect_postgres()
    init_schema(conn)
    _app.state.pg = conn
    _app.state.redis = connect_redis()
    _app.state.openai = create_openai_client()
    yield


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


@app.get("/health")
async def health():
    """Check connectivity to all dependencies."""
    pg_ok = False
    redis_ok = False
    try:
        with app.state.pg.cursor() as cur:
            cur.execute("SELECT 1")
        pg_ok = True
    except Exception:
        pass
    try:
        redis_ok = app.state.redis.ping()
    except Exception:
        pass
    llm_configured = app.state.openai is not None
    is_healthy = pg_ok and redis_ok
    return {
        "status": "ok" if is_healthy else "degraded",
        "postgres_connected": pg_ok,
        "redis_connected": redis_ok,
        "llm_configured": llm_configured,
    }


@app.post("/chat", response_model=ChatResponse)
async def chat(req: ChatRequest):
    """Send a message and get a response."""
    conn = app.state.pg
    r = app.state.redis
    session_id = req.session_id or str(uuid.uuid4())

    ensure_session(conn, session_id)
    save_message(conn, session_id, "user", req.message)

    cache_key = f"chat:{session_id}:{req.message}"
    cached = r.get(cache_key)
    if cached:
        response = cached
    else:
        history = fetch_history(conn, session_id)
        response = generate_response(app.state.openai, req.message, history)
        r.setex(cache_key, CACHE_TTL, response)

    save_message(conn, session_id, "assistant", response)
    return ChatResponse(
        session_id=session_id,
        response=response,
        history_length=count_messages(conn, session_id),
    )


@app.post("/chat/stream")
async def chat_stream(req: ChatRequest):
    """Send a message and stream the response via SSE."""
    conn = app.state.pg
    client = app.state.openai
    session_id = req.session_id or str(uuid.uuid4())

    ensure_session(conn, session_id)
    save_message(conn, session_id, "user", req.message)
    history = fetch_history(conn, session_id)

    async def event_generator():
        if client is None:
            fallback = f"[No API key configured] Echo: {req.message}"
            yield {"data": json.dumps({"content": fallback, "done": True})}
            save_message(conn, session_id, "assistant", fallback)
            return

        messages = build_messages(req.message, history)
        full_response: list[str] = []

        stream = client.chat.completions.create(
            model=OPENAI_MODEL,
            messages=messages,
            stream=True,
        )
        for chunk in stream:
            delta = chunk.choices[0].delta.content
            if delta:
                full_response.append(delta)
                yield {"data": json.dumps({"content": delta, "done": False})}

        yield {
            "data": json.dumps(
                {"content": "", "done": True, "session_id": session_id}
            )
        }
        save_message(conn, session_id, "assistant", "".join(full_response))

    return EventSourceResponse(event_generator())


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
