"""Document Summarizer — mock summarization with volume-backed cache."""

import hashlib
import os
from pathlib import Path

from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI(title="Document Summarizer")

CACHE_DIR = Path("/data/cache")


class Document(BaseModel):
    text: str
    max_length: int = 100


class Summary(BaseModel):
    summary: str
    cached: bool
    original_length: int


def _cache_key(text: str) -> str:
    return hashlib.sha256(text.encode()).hexdigest()[:16]


def _mock_summarize(text: str, max_length: int) -> str:
    """Mock summarization: return first max_length chars with ellipsis."""
    if len(text) <= max_length:
        return text
    return text[:max_length].rsplit(" ", 1)[0] + "..."


@app.get("/health")
async def health():
    # Verify that the volume mount is writable.
    try:
        CACHE_DIR.mkdir(parents=True, exist_ok=True)
        probe = CACHE_DIR / ".health_probe"
        probe.write_text("ok")
        probe.unlink()
        volume_ok = True
    except OSError:
        volume_ok = False
    return {
        "status": "ok" if volume_ok else "degraded",
        "cache_writable": volume_ok,
        "api_key_set": bool(os.getenv("OPENAI_API_KEY")),
    }


@app.post("/summarize", response_model=Summary)
async def summarize(doc: Document):
    key = _cache_key(doc.text)
    cache_file = CACHE_DIR / f"{key}.txt"

    # Check cache first.
    if cache_file.exists():
        return Summary(
            summary=cache_file.read_text(),
            cached=True,
            original_length=len(doc.text),
        )

    result = _mock_summarize(doc.text, doc.max_length)

    # Write to cache.
    CACHE_DIR.mkdir(parents=True, exist_ok=True)
    cache_file.write_text(result)

    return Summary(
        summary=result,
        cached=False,
        original_length=len(doc.text),
    )


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
