"""Sentiment Analyzer — keyword-based sentiment analysis agent.

Uses a simple keyword approach instead of NLTK to avoid runtime data
downloads inside containers (Volra's auto Dockerfile does not support
build-time model/data downloads — see FINDINGS.md H4).
"""

import os
import re

from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI(title="Sentiment Analyzer")

model_name = os.getenv("MODEL_NAME", "keyword_v1")

# Simple keyword-based sentiment lexicon.
_POSITIVE = {"good", "great", "excellent", "amazing", "wonderful", "fantastic",
             "love", "happy", "best", "awesome", "nice", "perfect", "beautiful"}
_NEGATIVE = {"bad", "terrible", "awful", "horrible", "hate", "worst", "ugly",
             "poor", "sad", "angry", "disappointing", "fail", "broken"}


class TextInput(BaseModel):
    text: str


class SentimentResult(BaseModel):
    neg: float
    neu: float
    pos: float
    compound: float
    label: str


def _analyze(text: str) -> dict[str, float]:
    words = set(re.findall(r'\w+', text.lower()))
    pos_count = len(words & _POSITIVE)
    neg_count = len(words & _NEGATIVE)
    total = pos_count + neg_count or 1
    pos_score = round(pos_count / total, 4)
    neg_score = round(neg_count / total, 4)
    neu_score = round(1.0 - pos_score - neg_score, 4)
    compound = round((pos_score - neg_score), 4)
    return {"pos": pos_score, "neg": neg_score, "neu": neu_score, "compound": compound}


@app.get("/health")
async def health():
    return {"status": "ok", "model": model_name}


@app.post("/analyze", response_model=SentimentResult)
async def analyze(input: TextInput):
    scores = _analyze(input.text)
    compound = scores["compound"]
    if compound >= 0.05:
        label = "positive"
    elif compound <= -0.05:
        label = "negative"
    else:
        label = "neutral"
    return SentimentResult(label=label, **scores)


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
