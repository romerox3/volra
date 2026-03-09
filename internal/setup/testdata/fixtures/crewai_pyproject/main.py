from crewai import Crew
from fastapi import FastAPI

app = FastAPI()

@app.get("/health")
def health():
    return {"ok": True}

uvicorn.run(app, host="0.0.0.0", port=9000)
