import os
from crewai import Agent, Task, Crew
from fastapi import FastAPI

app = FastAPI()
api_key = os.environ["OPENAI_API_KEY"]

@app.get("/health")
def health():
    return {"ok": True}

uvicorn.run(app, host="0.0.0.0", port=8000)
