"""RAG Knowledge Base — in-memory KB with Redis query cache."""

import json
import os
import time

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

import redis

app = FastAPI(title="RAG Knowledge Base")

# In-memory knowledge base.
KNOWLEDGE = {
    "volra": "Volra is an AI agent deployment tool that uses Agentfiles to define agent configurations.",
    "agentfile": "An Agentfile is a YAML manifest that declares an agent's framework, port, services, and deployment options.",
    "langgraph": "LangGraph is a framework for building stateful, multi-actor applications with LLMs.",
    "docker": "Docker is a platform for building, shipping, and running applications in containers.",
    "fastapi": "FastAPI is a modern Python web framework for building APIs with automatic OpenAPI documentation.",
}

REDIS_HOST = os.getenv("REDIS_HOST", "rag-kb-redis-cache")
REDIS_PORT = int(os.getenv("REDIS_PORT", "6379"))

_redis: redis.Redis | None = None


def get_redis() -> redis.Redis:
    global _redis
    if _redis is None:
        _redis = redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)
    return _redis


class Query(BaseModel):
    question: str


class KBResult(BaseModel):
    answer: str
    source: str
    cached: bool


@app.get("/health")
async def health():
    redis_ok = False
    try:
        r = get_redis()
        redis_ok = r.ping()
    except Exception:
        pass
    return {
        "status": "ok" if redis_ok else "degraded",
        "redis_connected": redis_ok,
        "kb_entries": len(KNOWLEDGE),
    }


@app.post("/query", response_model=KBResult)
async def query(q: Query):
    r = get_redis()
    cache_key = f"kb:{q.question.lower().strip()}"

    # Check Redis cache.
    cached = r.get(cache_key)
    if cached:
        data = json.loads(cached)
        return KBResult(answer=data["answer"], source=data["source"], cached=True)

    # Simple keyword search in knowledge base.
    question_lower = q.question.lower()
    for key, value in KNOWLEDGE.items():
        if key in question_lower:
            result = {"answer": value, "source": key}
            r.setex(cache_key, 300, json.dumps(result))
            return KBResult(answer=value, source=key, cached=False)

    raise HTTPException(status_code=404, detail="No relevant knowledge found")


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
