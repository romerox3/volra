"""
Polyglot Research Agent — Multi-provider, multi-service AI agent.

Combines:
- Dual LLM routing (OpenAI GPT-4o + Anthropic Claude Sonnet)
- RAG pipeline with ChromaDB vector store
- Conversation memory with PostgreSQL
- Response caching with Redis
- NLP preprocessing with NLTK
- Full observability via volra-observe
- Tool calling (web search, calculator, summarizer)
"""

import os
import json
import time
import hashlib
import logging
from contextlib import asynccontextmanager
from datetime import datetime

import nltk
from nltk.tokenize import sent_tokenize
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import psycopg2
from psycopg2.extras import RealDictCursor
import redis
import chromadb
import openai
import anthropic
import volra_observe

# --- Configuration ---

REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379")
PG_DSN = os.getenv("DATABASE_URL", "postgresql://postgres:changeme@localhost:5432/agent")
CHROMA_HOST = os.getenv("CHROMA_HOST", "localhost")
CHROMA_PORT = int(os.getenv("CHROMA_PORT", "8000"))
OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
ANTHROPIC_API_KEY = os.getenv("ANTHROPIC_API_KEY", "")
LLM_PROVIDER = os.getenv("LLM_PROVIDER", "openai")  # openai | anthropic | auto

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(name)s %(message)s")
logger = logging.getLogger("polyglot")

# --- Clients ---

redis_client = None
pg_conn = None
chroma_collection = None
openai_client = None
anthropic_client = None


def init_redis():
    global redis_client
    redis_client = redis.from_url(REDIS_URL, decode_responses=True)
    redis_client.ping()
    logger.info("Redis connected")


def init_postgres():
    global pg_conn
    pg_conn = psycopg2.connect(PG_DSN)
    pg_conn.autocommit = True
    with pg_conn.cursor() as cur:
        cur.execute("""
            CREATE TABLE IF NOT EXISTS conversations (
                id SERIAL PRIMARY KEY,
                session_id TEXT NOT NULL,
                role TEXT NOT NULL,
                content TEXT NOT NULL,
                provider TEXT,
                model TEXT,
                tokens_used INT DEFAULT 0,
                cost_dollars FLOAT DEFAULT 0.0,
                created_at TIMESTAMP DEFAULT NOW()
            )
        """)
        cur.execute("""
            CREATE TABLE IF NOT EXISTS documents (
                id SERIAL PRIMARY KEY,
                title TEXT NOT NULL,
                content TEXT NOT NULL,
                chunk_count INT DEFAULT 0,
                indexed_at TIMESTAMP DEFAULT NOW()
            )
        """)
    logger.info("PostgreSQL connected, tables ready")


def init_chroma():
    global chroma_collection
    client = chromadb.HttpClient(host=CHROMA_HOST, port=CHROMA_PORT)
    chroma_collection = client.get_or_create_collection(
        name="research_docs",
        metadata={"hnsw:space": "cosine"},
    )
    logger.info("ChromaDB connected, collection: research_docs")


def init_llm_clients():
    global openai_client, anthropic_client
    if OPENAI_API_KEY:
        openai_client = openai.OpenAI(api_key=OPENAI_API_KEY)
        logger.info("OpenAI client initialized")
    if ANTHROPIC_API_KEY:
        anthropic_client = anthropic.Anthropic(api_key=ANTHROPIC_API_KEY)
        logger.info("Anthropic client initialized")


# --- NLTK Preprocessing ---

def preprocess_text(text: str) -> list[str]:
    """Split text into sentences using NLTK punkt tokenizer."""
    sentences = sent_tokenize(text)
    return [s.strip() for s in sentences if len(s.strip()) > 10]


def chunk_document(text: str, chunk_size: int = 3) -> list[str]:
    """Chunk document into groups of sentences for RAG indexing."""
    sentences = preprocess_text(text)
    chunks = []
    for i in range(0, len(sentences), chunk_size):
        chunk = " ".join(sentences[i : i + chunk_size])
        chunks.append(chunk)
    return chunks


# --- LLM Routing ---

def select_provider(query: str) -> str:
    """Route to the best LLM provider based on query characteristics."""
    if LLM_PROVIDER != "auto":
        return LLM_PROVIDER

    # Heuristic: use Anthropic for longer, analytical queries; OpenAI for shorter, creative ones
    if len(query) > 500 or any(kw in query.lower() for kw in ["analyze", "compare", "research", "explain in detail"]):
        return "anthropic" if anthropic_client else "openai"
    return "openai" if openai_client else "anthropic"


@volra_observe.track_llm("gpt-4o")
def call_openai(messages: list[dict], model: str = "gpt-4o") -> dict:
    """Call OpenAI with observability tracking."""
    response = openai_client.chat.completions.create(
        model=model,
        messages=messages,
        max_tokens=2048,
    )
    return {
        "content": response.choices[0].message.content,
        "model": response.model,
        "tokens": response.usage.total_tokens,
        "input_tokens": response.usage.prompt_tokens,
        "output_tokens": response.usage.completion_tokens,
        "provider": "openai",
    }


def call_anthropic(messages: list[dict], model: str = "claude-sonnet-4-20250514") -> dict:
    """Call Anthropic with observability tracking."""
    with volra_observe.llm_context(model):
        # Convert OpenAI-style messages to Anthropic format
        system_msg = ""
        user_messages = []
        for msg in messages:
            if msg["role"] == "system":
                system_msg = msg["content"]
            else:
                user_messages.append(msg)

        response = anthropic_client.messages.create(
            model=model,
            max_tokens=2048,
            system=system_msg,
            messages=user_messages,
        )
        return {
            "content": response.content[0].text,
            "model": response.model,
            "tokens": response.usage.input_tokens + response.usage.output_tokens,
            "input_tokens": response.usage.input_tokens,
            "output_tokens": response.usage.output_tokens,
            "provider": "anthropic",
        }


def call_llm(messages: list[dict], provider: str = None) -> dict:
    """Route LLM call to the appropriate provider."""
    provider = provider or select_provider(messages[-1]["content"])

    if provider == "anthropic" and anthropic_client:
        return call_anthropic(messages)
    elif provider == "openai" and openai_client:
        return call_openai(messages)
    else:
        raise HTTPException(status_code=503, detail=f"No LLM client available for provider: {provider}")


# --- Tools ---

def tool_calculator(expression: str) -> str:
    """Safe math expression evaluator."""
    volra_observe.record_tool_call("calculator")
    allowed = set("0123456789+-*/.() ")
    if not all(c in allowed for c in expression):
        return "Error: invalid characters in expression"
    try:
        result = eval(expression)  # Safe: only digits and operators allowed
        return str(result)
    except Exception as e:
        return f"Error: {e}"


def tool_summarize(text: str, max_sentences: int = 3) -> str:
    """Extractive summarizer using NLTK sentence tokenization."""
    volra_observe.record_tool_call("summarizer")
    sentences = preprocess_text(text)
    return " ".join(sentences[:max_sentences])


def tool_search_documents(query: str, n_results: int = 3) -> list[dict]:
    """Search indexed documents via ChromaDB."""
    volra_observe.record_tool_call("document_search")
    if chroma_collection is None:
        return [{"error": "ChromaDB not available"}]

    results = chroma_collection.query(query_texts=[query], n_results=n_results)
    docs = []
    for i, doc_text in enumerate(results["documents"][0]):
        docs.append({
            "text": doc_text,
            "distance": results["distances"][0][i] if results["distances"] else None,
            "metadata": results["metadatas"][0][i] if results["metadatas"] else None,
        })
    return docs


TOOLS = {
    "calculator": tool_calculator,
    "summarize": tool_summarize,
    "search": tool_search_documents,
}

# --- Cache ---

def cache_key(query: str, provider: str) -> str:
    """Generate a cache key for a query + provider combination."""
    return f"polyglot:cache:{hashlib.sha256(f'{provider}:{query}'.encode()).hexdigest()[:16]}"


def get_cached_response(query: str, provider: str) -> dict | None:
    """Check Redis cache for a previous response."""
    if redis_client is None:
        return None
    key = cache_key(query, provider)
    cached = redis_client.get(key)
    if cached:
        logger.info(f"Cache HIT for {key}")
        return json.loads(cached)
    return None


def set_cached_response(query: str, provider: str, response: dict, ttl: int = 300):
    """Cache a response in Redis with TTL."""
    if redis_client is None:
        return
    key = cache_key(query, provider)
    redis_client.setex(key, ttl, json.dumps(response))


# --- Conversation Memory ---

def save_message(session_id: str, role: str, content: str, provider: str = None, model: str = None, tokens: int = 0, cost: float = 0.0):
    """Save a message to PostgreSQL conversation history."""
    if pg_conn is None:
        return
    with pg_conn.cursor() as cur:
        cur.execute(
            "INSERT INTO conversations (session_id, role, content, provider, model, tokens_used, cost_dollars) VALUES (%s, %s, %s, %s, %s, %s, %s)",
            (session_id, role, content, provider, model, tokens, cost),
        )


def get_history(session_id: str, limit: int = 20) -> list[dict]:
    """Retrieve conversation history from PostgreSQL."""
    if pg_conn is None:
        return []
    with pg_conn.cursor(cursor_factory=RealDictCursor) as cur:
        cur.execute(
            "SELECT role, content FROM conversations WHERE session_id = %s ORDER BY created_at DESC LIMIT %s",
            (session_id, limit),
        )
        rows = cur.fetchall()
    return [{"role": r["role"], "content": r["content"]} for r in reversed(rows)]


# --- RAG Pipeline ---

def index_document(title: str, content: str) -> dict:
    """Index a document into ChromaDB for RAG retrieval."""
    chunks = chunk_document(content)
    if not chunks:
        return {"error": "No valid chunks extracted from document"}

    ids = [f"{hashlib.sha256(title.encode()).hexdigest()[:8]}_{i}" for i in range(len(chunks))]
    metadatas = [{"title": title, "chunk_index": i, "indexed_at": datetime.now().isoformat()} for i in range(len(chunks))]

    chroma_collection.upsert(documents=chunks, ids=ids, metadatas=metadatas)

    # Also save to PostgreSQL for persistence
    if pg_conn:
        with pg_conn.cursor() as cur:
            cur.execute(
                "INSERT INTO documents (title, content, chunk_count) VALUES (%s, %s, %s) ON CONFLICT DO NOTHING",
                (title, content, len(chunks)),
            )

    return {"title": title, "chunks_indexed": len(chunks), "chunk_ids": ids}


def rag_query(query: str, session_id: str = "default") -> dict:
    """Full RAG pipeline: retrieve context → augment prompt → generate response."""
    # Step 1: Retrieve relevant documents
    context_docs = tool_search_documents(query, n_results=5)
    context_text = "\n---\n".join(d["text"] for d in context_docs if "error" not in d)

    # Step 2: Build augmented prompt
    history = get_history(session_id, limit=10)
    provider = select_provider(query)

    messages = [
        {"role": "system", "content": f"You are a research assistant. Use the following context to answer questions.\n\nContext:\n{context_text}\n\nIf the context doesn't contain relevant information, say so clearly."},
    ]
    messages.extend(history)
    messages.append({"role": "user", "content": query})

    # Step 3: Check cache
    cached = get_cached_response(query, provider)
    if cached:
        return {**cached, "cached": True, "context_docs": len(context_docs)}

    # Step 4: Call LLM
    result = call_llm(messages, provider)

    # Step 5: Save to memory and cache
    save_message(session_id, "user", query)
    save_message(session_id, "assistant", result["content"], result["provider"], result["model"], result["tokens"])
    set_cached_response(query, provider, result)

    return {
        **result,
        "cached": False,
        "context_docs": len(context_docs),
        "session_id": session_id,
    }


# --- FastAPI App ---

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Initialize all services on startup."""
    # Start volra-observe metrics server
    volra_observe.init(port=9101, auto_patch=True)
    logger.info("volra-observe initialized on :9101")

    # Initialize services (with graceful degradation)
    for name, init_fn in [("Redis", init_redis), ("PostgreSQL", init_postgres), ("ChromaDB", init_chroma), ("LLM clients", init_llm_clients)]:
        try:
            init_fn()
        except Exception as e:
            logger.warning(f"Failed to init {name}: {e} — continuing with degraded functionality")

    yield

    # Cleanup
    if pg_conn:
        pg_conn.close()


app = FastAPI(
    title="Polyglot Research Agent",
    description="Multi-provider, multi-service AI research agent",
    version="1.0.0",
    lifespan=lifespan,
)


# --- Request/Response Models ---

class AskRequest(BaseModel):
    query: str
    session_id: str = "default"
    provider: str | None = None
    use_rag: bool = True

class AskResponse(BaseModel):
    content: str
    provider: str
    model: str
    tokens: int
    cached: bool
    context_docs: int = 0
    session_id: str = "default"

class IndexRequest(BaseModel):
    title: str
    content: str

class ToolRequest(BaseModel):
    tool: str
    input: str


# --- Endpoints ---

@app.get("/health")
def health():
    """Health check endpoint for Volra monitoring."""
    checks = {
        "status": "healthy",
        "timestamp": datetime.now().isoformat(),
        "services": {},
    }

    # Check Redis
    try:
        if redis_client and redis_client.ping():
            checks["services"]["redis"] = "connected"
    except Exception:
        checks["services"]["redis"] = "disconnected"

    # Check PostgreSQL
    try:
        if pg_conn and not pg_conn.closed:
            checks["services"]["postgres"] = "connected"
    except Exception:
        checks["services"]["postgres"] = "disconnected"

    # Check ChromaDB
    try:
        if chroma_collection:
            checks["services"]["chromadb"] = "connected"
    except Exception:
        checks["services"]["chromadb"] = "disconnected"

    # Check LLM providers
    checks["services"]["openai"] = "configured" if openai_client else "not configured"
    checks["services"]["anthropic"] = "configured" if anthropic_client else "not configured"

    return checks


@app.post("/ask", response_model=AskResponse)
def ask(req: AskRequest):
    """Main query endpoint — routes to RAG or direct LLM based on request."""
    start = time.time()

    if req.use_rag and chroma_collection:
        result = rag_query(req.query, req.session_id)
    else:
        provider = req.provider or select_provider(req.query)
        cached = get_cached_response(req.query, provider)
        if cached:
            result = {**cached, "cached": True, "context_docs": 0}
        else:
            messages = [
                {"role": "system", "content": "You are a helpful research assistant."},
            ]
            history = get_history(req.session_id, limit=10)
            messages.extend(history)
            messages.append({"role": "user", "content": req.query})
            result = call_llm(messages, provider)
            result["cached"] = False
            result["context_docs"] = 0
            save_message(req.session_id, "user", req.query)
            save_message(req.session_id, "assistant", result["content"], result["provider"], result["model"], result["tokens"])
            set_cached_response(req.query, provider, result)

    result["session_id"] = req.session_id
    elapsed = time.time() - start
    logger.info(f"Ask completed in {elapsed:.2f}s | provider={result.get('provider')} cached={result.get('cached')}")
    return AskResponse(**result)


@app.post("/index")
def index_doc(req: IndexRequest):
    """Index a document for RAG retrieval."""
    if chroma_collection is None:
        raise HTTPException(status_code=503, detail="ChromaDB not available")
    result = index_document(req.title, req.content)
    return result


@app.post("/tool")
def use_tool(req: ToolRequest):
    """Execute a tool by name."""
    if req.tool not in TOOLS:
        raise HTTPException(status_code=400, detail=f"Unknown tool: {req.tool}. Available: {list(TOOLS.keys())}")
    result = TOOLS[req.tool](req.input)
    return {"tool": req.tool, "result": result}


@app.get("/history/{session_id}")
def get_conversation_history(session_id: str, limit: int = 50):
    """Retrieve conversation history for a session."""
    history = get_history(session_id, limit)
    return {"session_id": session_id, "messages": history, "count": len(history)}


@app.get("/stats")
def stats():
    """Agent statistics and metrics."""
    result = {"uptime": "running"}

    if redis_client:
        try:
            info = redis_client.info("stats")
            result["redis"] = {
                "total_commands": info.get("total_commands_processed", 0),
                "cache_keys": redis_client.dbsize(),
            }
        except Exception:
            result["redis"] = {"status": "error"}

    if pg_conn and not pg_conn.closed:
        try:
            with pg_conn.cursor() as cur:
                cur.execute("SELECT COUNT(*) FROM conversations")
                result["conversations"] = cur.fetchone()[0]
                cur.execute("SELECT COUNT(*) FROM documents")
                result["documents"] = cur.fetchone()[0]
        except Exception:
            pass

    if chroma_collection:
        try:
            result["vectors"] = chroma_collection.count()
        except Exception:
            pass

    return result


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
