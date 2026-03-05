"""{{.Name}} — RAG agent with ChromaDB and Redis cache, deployed with Volra."""

import json
import os
import time
from contextlib import asynccontextmanager

import chromadb
import redis
from fastapi import FastAPI
from openai import OpenAI
from pydantic import BaseModel

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")

CHROMA_HOST = os.getenv("CHROMA_HOST", "{{.Name}}-vectordb")
CHROMA_PORT = int(os.getenv("CHROMA_PORT", "8000"))

REDIS_HOST = os.getenv("REDIS_HOST", "{{.Name}}-cache")
REDIS_PORT = int(os.getenv("REDIS_PORT", "6379"))

COLLECTION_NAME = "knowledge"
TOP_K = 3
CACHE_TTL = 300

# ---------------------------------------------------------------------------
# Sample knowledge base — replace with your own documents
# ---------------------------------------------------------------------------

SAMPLE_DOCUMENTS = [
    "Python is a high-level programming language known for readability, "
    "versatility, and a rich ecosystem of libraries for web, data, and AI.",
    "Docker is a platform for building, shipping, and running applications "
    "in lightweight containers that package code with all dependencies.",
    "FastAPI is a modern Python web framework for building APIs with automatic "
    "OpenAPI docs, type validation via Pydantic, and async support.",
    "Kubernetes orchestrates containerized applications across clusters, "
    "providing automated deployment, scaling, and self-healing.",
    "PostgreSQL is an advanced open-source relational database known for "
    "extensibility, ACID compliance, and support for JSON and full-text search.",
    "Redis is an in-memory data store used as a cache, message broker, "
    "and database, supporting strings, hashes, lists, sets, and streams.",
    "LangChain is a framework for building LLM-powered applications with "
    "chains, agents, retrieval, and memory components.",
    "Prometheus collects time-series metrics via a pull model and provides "
    "a powerful query language (PromQL) for monitoring and alerting.",
    "Grafana is an open-source visualization platform for dashboards, "
    "supporting Prometheus, PostgreSQL, Elasticsearch, and many data sources.",
    "CI/CD pipelines automate building, testing, and deploying code changes, "
    "enabling faster and more reliable software delivery.",
]

# ---------------------------------------------------------------------------
# Dependencies
# ---------------------------------------------------------------------------


def connect_chromadb(max_retries: int = 30) -> chromadb.HttpClient:
    """Connect to ChromaDB with retry logic for container startup."""
    for attempt in range(max_retries):
        try:
            client = chromadb.HttpClient(host=CHROMA_HOST, port=CHROMA_PORT)
            client.heartbeat()
            return client
        except Exception:
            if attempt == max_retries - 1:
                raise
            time.sleep(1)
    msg = f"Cannot connect to ChromaDB after {max_retries} retries"
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
# Knowledge indexing
# ---------------------------------------------------------------------------


def index_documents(
    collection: chromadb.Collection,
    documents: list[str],
) -> None:
    """Index documents into ChromaDB if collection is empty."""
    if collection.count() > 0:
        return
    collection.add(
        documents=documents,
        ids=[f"doc-{i}" for i in range(len(documents))],
    )


# ---------------------------------------------------------------------------
# RAG pipeline
# ---------------------------------------------------------------------------


def retrieve(collection: chromadb.Collection, question: str) -> list[str]:
    """Retrieve relevant documents from ChromaDB."""
    results = collection.query(query_texts=[question], n_results=TOP_K)
    return results["documents"][0] if results["documents"] else []


def generate_answer(
    client: OpenAI | None,
    question: str,
    context: list[str],
) -> dict:
    """Generate an answer using retrieved context."""
    context_text = "\n\n".join(context)

    if client is None:
        return {
            "answer": context_text,
            "sources": len(context),
            "generated": False,
        }

    prompt = (
        "Answer the question based on the context below. "
        "If the context doesn't contain relevant information, say so.\n\n"
        f"Context:\n{context_text}\n\n"
        f"Question: {question}"
    )
    resp = client.chat.completions.create(
        model=OPENAI_MODEL,
        messages=[{"role": "user", "content": prompt}],
    )
    return {
        "answer": resp.choices[0].message.content,
        "sources": len(context),
        "generated": True,
    }


# ---------------------------------------------------------------------------
# Models
# ---------------------------------------------------------------------------


class QueryRequest(BaseModel):
    question: str


# ---------------------------------------------------------------------------
# Application
# ---------------------------------------------------------------------------


@asynccontextmanager
async def lifespan(_app: FastAPI):
    """Initialize ChromaDB, Redis, and OpenAI on startup."""
    chroma = connect_chromadb()
    collection = chroma.get_or_create_collection(COLLECTION_NAME)
    index_documents(collection, SAMPLE_DOCUMENTS)

    _app.state.collection = collection
    _app.state.redis = connect_redis()
    _app.state.openai = create_openai_client()
    yield


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


@app.get("/health")
async def health():
    """Check connectivity to all dependencies."""
    redis_ok = False
    chroma_ok = False
    try:
        redis_ok = app.state.redis.ping()
    except Exception:
        pass
    try:
        app.state.collection.count()
        chroma_ok = True
    except Exception:
        pass
    llm_configured = app.state.openai is not None
    is_healthy = redis_ok and chroma_ok
    return {
        "status": "ok" if is_healthy else "degraded",
        "redis_connected": redis_ok,
        "chromadb_connected": chroma_ok,
        "llm_configured": llm_configured,
    }


@app.post("/query")
async def query(req: QueryRequest):
    """Query the knowledge base with semantic search and optional LLM generation."""
    r = app.state.redis
    cache_key = f"rag:{req.question.lower().strip()}"

    cached = r.get(cache_key)
    if cached:
        return {**json.loads(cached), "cached": True}

    context = retrieve(app.state.collection, req.question)
    if not context:
        return {"answer": "No relevant documents found.", "sources": 0, "cached": False}

    result = generate_answer(app.state.openai, req.question, context)
    r.setex(cache_key, CACHE_TTL, json.dumps(result))
    return {**result, "cached": False}


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
