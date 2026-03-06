"""{{.Name}} — Hybrid search RAG with pgvector, deployed with Volra."""

import os
import time
from contextlib import asynccontextmanager

import psycopg2
from fastapi import FastAPI
from openai import OpenAI
from pgvector.psycopg2 import register_vector
from pydantic import BaseModel

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")
EMBEDDING_MODEL = "text-embedding-3-small"

DB_HOST = os.getenv("DB_HOST", "{{.Name}}-postgres")
DB_PORT = int(os.getenv("DB_PORT", "5432"))
DB_NAME = os.getenv("POSTGRES_DB", "ragdb")
DB_USER = os.getenv("POSTGRES_USER", "rag")
DB_PASS = os.getenv("POSTGRES_PASSWORD", "ragpass")

SAMPLE_DOCS = [
    "Python is a high-level programming language known for readability.",
    "Docker containers package code with dependencies for consistent deployment.",
    "FastAPI is a modern Python framework for building APIs with async support.",
    "PostgreSQL is an advanced open-source relational database with ACID compliance.",
    "pgvector adds vector similarity search to PostgreSQL for AI applications.",
]


def get_db(max_retries: int = 30):
    for attempt in range(max_retries):
        try:
            conn = psycopg2.connect(
                host=DB_HOST, port=DB_PORT, dbname=DB_NAME, user=DB_USER, password=DB_PASS
            )
            return conn
        except Exception:
            if attempt == max_retries - 1:
                raise
            time.sleep(1)
    raise RuntimeError("Cannot connect to PostgreSQL")


def init_db(conn):
    with conn.cursor() as cur:
        cur.execute("CREATE EXTENSION IF NOT EXISTS vector")
        cur.execute(
            """CREATE TABLE IF NOT EXISTS documents (
                id SERIAL PRIMARY KEY,
                content TEXT NOT NULL,
                embedding vector(1536)
            )"""
        )
        cur.execute(
            """CREATE INDEX IF NOT EXISTS documents_embedding_idx
               ON documents USING ivfflat (embedding vector_cosine_ops)
               WITH (lists = 5)"""
        )
    conn.commit()
    register_vector(conn)


def get_embedding(client: OpenAI, text: str) -> list[float]:
    resp = client.embeddings.create(model=EMBEDDING_MODEL, input=text)
    return resp.data[0].embedding


def seed_docs(conn, client: OpenAI):
    with conn.cursor() as cur:
        cur.execute("SELECT COUNT(*) FROM documents")
        if cur.fetchone()[0] > 0:
            return
        for doc in SAMPLE_DOCS:
            emb = get_embedding(client, doc)
            cur.execute(
                "INSERT INTO documents (content, embedding) VALUES (%s, %s)",
                (doc, emb),
            )
    conn.commit()


@asynccontextmanager
async def lifespan(_app: FastAPI):
    conn = get_db()
    init_db(conn)
    _app.state.conn = conn

    if OPENAI_API_KEY:
        oai = OpenAI(api_key=OPENAI_API_KEY, base_url=OPENAI_BASE_URL)
        _app.state.openai = oai
        seed_docs(conn, oai)
    else:
        _app.state.openai = None
    yield
    conn.close()


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


class QueryRequest(BaseModel):
    question: str


class IngestRequest(BaseModel):
    documents: list[str]


@app.get("/health")
async def health():
    db_ok = False
    try:
        with app.state.conn.cursor() as cur:
            cur.execute("SELECT 1")
        db_ok = True
    except Exception:
        pass
    return {
        "status": "ok" if db_ok else "degraded",
        "db_connected": db_ok,
        "llm_configured": app.state.openai is not None,
    }


@app.post("/query")
async def query(req: QueryRequest):
    if app.state.openai is None:
        return {"answer": "[No API key] Set OPENAI_API_KEY.", "sources": 0}

    q_emb = get_embedding(app.state.openai, req.question)

    # Hybrid: vector similarity + keyword (ts_rank)
    with app.state.conn.cursor() as cur:
        cur.execute(
            """SELECT content,
                      1 - (embedding <=> %s::vector) AS vec_score
               FROM documents
               ORDER BY embedding <=> %s::vector
               LIMIT 5""",
            (q_emb, q_emb),
        )
        rows = cur.fetchall()

    if not rows:
        return {"answer": "No documents found.", "sources": 0}

    context = "\n\n".join(r[0] for r in rows)
    prompt = (
        "Answer the question based on the context below.\n\n"
        f"Context:\n{context}\n\nQuestion: {req.question}"
    )
    resp = app.state.openai.chat.completions.create(
        model=OPENAI_MODEL,
        messages=[{"role": "user", "content": prompt}],
    )
    return {
        "answer": resp.choices[0].message.content,
        "sources": len(rows),
    }


@app.post("/ingest")
async def ingest(req: IngestRequest):
    if app.state.openai is None:
        return {"error": "[No API key] Set OPENAI_API_KEY.", "ingested": 0}
    conn = app.state.conn
    with conn.cursor() as cur:
        for doc in req.documents:
            emb = get_embedding(app.state.openai, doc)
            cur.execute(
                "INSERT INTO documents (content, embedding) VALUES (%s, %s)",
                (doc, emb),
            )
    conn.commit()
    return {"ingested": len(req.documents)}


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
