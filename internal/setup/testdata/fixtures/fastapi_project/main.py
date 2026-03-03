import os
from fastapi import FastAPI

app = FastAPI()

API_KEY = os.environ["OPENAI_API_KEY"]
MODEL = os.getenv("MODEL_NAME", "gpt-4")

@app.get("/health")
def health():
    return {"status": "ok"}

@app.post("/run")
def run_agent():
    return {"result": "done"}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8080)
