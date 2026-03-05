"""{{.Name}} — RAG agent with Redis cache, deployed with Volra."""

import json
import os

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

import redis

app = FastAPI(title="{{.Name}}")

# In-memory knowledge base. Replace with your own data source.
KNOWLEDGE = {
    "python": "Python is a high-level programming language known for readability and versatility.",
    "docker": "Docker is a platform for building, shipping, and running applications in containers.",
    "fastapi": "FastAPI is a modern Python web framework for building APIs with automatic OpenAPI documentation.",
}

REDIS_HOST = os.getenv("REDIS_HOST", "{{.Name}}-cache")
REDIS_PORT = int(os.getenv("REDIS_PORT", "6379"))

_redis: redis.Redis | None = None


def get_redis() -> redis.Redis:
    global _redis
    if _redis is None:
        _redis = redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)
    return _redis


class Query(BaseModel):
    question: str


@app.get("/health")
async def health():
    redis_ok = False
    try:
        redis_ok = get_redis().ping()
    except Exception:
        pass
    return {"status": "ok" if redis_ok else "degraded", "redis_connected": redis_ok}


@app.post("/query")
async def query(q: Query):
    r = get_redis()
    cache_key = f"kb:{q.question.lower().strip()}"

    # Check cache.
    cached = r.get(cache_key)
    if cached:
        data = json.loads(cached)
        return {**data, "cached": True}

    # Simple keyword search.
    for key, value in KNOWLEDGE.items():
        if key in q.question.lower():
            result = {"answer": value, "source": key}
            r.setex(cache_key, 300, json.dumps(result))
            return {**result, "cached": False}

    raise HTTPException(status_code=404, detail="No relevant knowledge found")


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
