"""AI Gateway — rate-limited mock LLM proxy with security context."""

import time
from collections import defaultdict

from fastapi import FastAPI, HTTPException, Request
from pydantic import BaseModel

app = FastAPI(title="AI Gateway")

# In-memory rate limiting: {ip: [(timestamp, ...)]}.
_rate_limits: dict[str, list[float]] = defaultdict(list)
RATE_LIMIT = 10  # requests per window
RATE_WINDOW = 60  # seconds


class CompletionRequest(BaseModel):
    model: str = "mock-gpt"
    prompt: str
    max_tokens: int = 100


class CompletionResponse(BaseModel):
    model: str
    text: str
    tokens_used: int


def _check_rate_limit(client_ip: str) -> bool:
    now = time.time()
    window_start = now - RATE_WINDOW
    # Prune old entries.
    _rate_limits[client_ip] = [
        t for t in _rate_limits[client_ip] if t > window_start
    ]
    if len(_rate_limits[client_ip]) >= RATE_LIMIT:
        return False
    _rate_limits[client_ip].append(now)
    return True


@app.get("/health")
async def health():
    return {"status": "ok", "security": "hardened"}


@app.post("/v1/completions", response_model=CompletionResponse)
async def completions(req: CompletionRequest, request: Request):
    client_ip = request.client.host if request.client else "unknown"
    if not _check_rate_limit(client_ip):
        raise HTTPException(
            status_code=429,
            detail=f"Rate limit exceeded: {RATE_LIMIT} requests per {RATE_WINDOW}s",
        )

    # Mock LLM response.
    mock_text = f"[Mock {req.model}] Response to: {req.prompt[:50]}"
    tokens = min(len(req.prompt.split()) + 10, req.max_tokens)

    return CompletionResponse(
        model=req.model,
        text=mock_text,
        tokens_used=tokens,
    )


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
