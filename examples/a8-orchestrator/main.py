"""Multi-Agent Orchestrator — all Volra features combined."""

import json
import os
import time
import uuid
from pathlib import Path

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

import httpx
import psycopg2
import redis

app = FastAPI(title="Multi-Agent Orchestrator")

# Service hostnames (based on Volra naming: {name}-{service}).
PG_HOST = os.getenv("PG_HOST", "orchestrator-postgres")
PG_PORT = int(os.getenv("PG_PORT", "5432"))
PG_PASSWORD = os.getenv("POSTGRES_PASSWORD", "volra_test")
PG_DB = os.getenv("POSTGRES_DB", "orchestrator")

REDIS_HOST = os.getenv("REDIS_HOST", "orchestrator-redis")
REDIS_PORT = int(os.getenv("REDIS_PORT", "6379"))

CHROMA_HOST = os.getenv("CHROMA_HOST", "orchestrator-chromadb")
CHROMA_PORT = int(os.getenv("CHROMA_PORT", "8100"))

ORCHESTRATOR_MODE = os.getenv("ORCHESTRATOR_MODE", "standard")

# Volume paths.
SESSIONS_DIR = Path("/data/sessions")
EMBEDDINGS_DIR = Path("/data/embeddings")
LOGS_DIR = Path("/data/logs")

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
            CREATE TABLE IF NOT EXISTS tasks (
                id TEXT PRIMARY KEY,
                title TEXT NOT NULL,
                status TEXT DEFAULT 'pending',
                result TEXT,
                created_at TIMESTAMP DEFAULT NOW(),
                updated_at TIMESTAMP DEFAULT NOW()
            )
        """)


def _log_event(event: str, data: dict):
    """Write event to logs volume."""
    LOGS_DIR.mkdir(parents=True, exist_ok=True)
    log_file = LOGS_DIR / "events.jsonl"
    entry = {"timestamp": time.time(), "event": event, **data}
    with open(log_file, "a") as f:
        f.write(json.dumps(entry) + "\n")


class TaskCreate(BaseModel):
    title: str
    context: str = ""


class TaskResponse(BaseModel):
    task_id: str
    title: str
    status: str
    result: str | None


class OrchestratorStatus(BaseModel):
    mode: str
    postgres: bool
    redis: bool
    chromadb: bool
    volumes_writable: bool
    task_count: int


@app.on_event("startup")
async def startup():
    init_db()
    for d in [SESSIONS_DIR, EMBEDDINGS_DIR, LOGS_DIR]:
        d.mkdir(parents=True, exist_ok=True)
    _log_event("startup", {"mode": ORCHESTRATOR_MODE})


@app.get("/health")
async def health():
    pg_ok = False
    redis_ok = False
    chroma_ok = False
    volumes_ok = False
    task_count = 0

    try:
        conn = get_pg()
        with conn.cursor() as cur:
            cur.execute("SELECT COUNT(*) FROM tasks")
            task_count = cur.fetchone()[0]
        pg_ok = True
    except Exception:
        pass

    try:
        r = get_redis()
        redis_ok = r.ping()
    except Exception:
        pass

    try:
        resp = httpx.get(f"http://{CHROMA_HOST}:{CHROMA_PORT}/api/v1/heartbeat", timeout=5)
        chroma_ok = resp.status_code == 200
    except Exception:
        pass

    try:
        for d in [SESSIONS_DIR, EMBEDDINGS_DIR, LOGS_DIR]:
            probe = d / ".health_probe"
            probe.write_text("ok")
            probe.unlink()
        volumes_ok = True
    except OSError:
        pass

    all_ok = pg_ok and redis_ok and chroma_ok and volumes_ok
    return OrchestratorStatus(
        mode=ORCHESTRATOR_MODE,
        postgres=pg_ok,
        redis=redis_ok,
        chromadb=chroma_ok,
        volumes_writable=volumes_ok,
        task_count=task_count,
    )


@app.post("/tasks", response_model=TaskResponse)
async def create_task(req: TaskCreate):
    conn = get_pg()
    r = get_redis()
    task_id = str(uuid.uuid4())

    # Store task in Postgres.
    with conn.cursor() as cur:
        cur.execute(
            "INSERT INTO tasks (id, title) VALUES (%s, %s)",
            (task_id, req.title),
        )

    # Cache in Redis.
    r.setex(f"task:{task_id}", 3600, json.dumps({"title": req.title, "status": "pending"}))

    # Store context as mock embedding.
    if req.context:
        embedding_file = EMBEDDINGS_DIR / f"{task_id}.json"
        embedding_file.write_text(json.dumps({"context": req.context, "task_id": task_id}))

    # Save session.
    session_file = SESSIONS_DIR / f"{task_id}.json"
    session_file.write_text(json.dumps({"task_id": task_id, "title": req.title}))

    _log_event("task_created", {"task_id": task_id, "title": req.title})

    # Mock: simulate orchestration result.
    result = f"[{ORCHESTRATOR_MODE}] Processed: {req.title}"
    with conn.cursor() as cur:
        cur.execute(
            "UPDATE tasks SET status = 'completed', result = %s, updated_at = NOW() WHERE id = %s",
            (result, task_id),
        )

    r.setex(f"task:{task_id}", 3600, json.dumps({"title": req.title, "status": "completed"}))
    _log_event("task_completed", {"task_id": task_id})

    return TaskResponse(task_id=task_id, title=req.title, status="completed", result=result)


@app.get("/tasks/{task_id}", response_model=TaskResponse)
async def get_task(task_id: str):
    r = get_redis()

    # Try cache first.
    cached = r.get(f"task:{task_id}")
    if cached:
        data = json.loads(cached)
        return TaskResponse(
            task_id=task_id,
            title=data["title"],
            status=data["status"],
            result=data.get("result"),
        )

    # Fallback to Postgres.
    conn = get_pg()
    with conn.cursor() as cur:
        cur.execute("SELECT title, status, result FROM tasks WHERE id = %s", (task_id,))
        row = cur.fetchone()
        if not row:
            raise HTTPException(status_code=404, detail="Task not found")
        return TaskResponse(task_id=task_id, title=row[0], status=row[1], result=row[2])


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
