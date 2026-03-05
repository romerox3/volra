"""Vision Classifier — mock image classification with GPU requirement."""

import os
from pathlib import Path

from fastapi import FastAPI, UploadFile, File
from pydantic import BaseModel

app = FastAPI(title="Vision Classifier")

MODELS_DIR = Path("/models")

# Mock classification labels.
LABELS = [
    ("cat", 0.85),
    ("dog", 0.10),
    ("bird", 0.03),
    ("other", 0.02),
]


class ClassificationResult(BaseModel):
    label: str
    confidence: float
    all_scores: dict[str, float]
    gpu_available: bool


def _check_gpu() -> bool:
    """Check if NVIDIA GPU device is available."""
    return os.path.exists("/dev/nvidia0")


@app.get("/health")
async def health():
    gpu_ok = _check_gpu()
    models_writable = False
    try:
        MODELS_DIR.mkdir(parents=True, exist_ok=True)
        probe = MODELS_DIR / ".health_probe"
        probe.write_text("ok")
        probe.unlink()
        models_writable = True
    except OSError:
        pass
    return {
        "status": "ok",
        "gpu_available": gpu_ok,
        "models_writable": models_writable,
    }


@app.post("/classify", response_model=ClassificationResult)
async def classify(file: UploadFile = File(...)):
    # Read file to simulate processing.
    content = await file.read()
    file_size = len(content)

    # Mock classification — adjust confidence by file size for variety.
    scores = {label: round(score * (1 + (file_size % 10) / 100), 4) for label, score in LABELS}
    top_label = max(scores, key=scores.get)

    return ClassificationResult(
        label=top_label,
        confidence=scores[top_label],
        all_scores=scores,
        gpu_available=_check_gpu(),
    )


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
