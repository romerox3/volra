"""{{.Name}} — custom agent deployed with Volra."""

from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI(title="{{.Name}}")


# TODO: Add your configuration here
# import os
# OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")


# TODO: Add your tools / agent logic here
def process_message(message: str) -> str:
    """Process a user message. Replace with your agent logic."""
    return f"Echo: {message}"


class InvokeRequest(BaseModel):
    message: str


class InvokeResponse(BaseModel):
    response: str


@app.get("/health")
async def health():
    return {"status": "ok"}


@app.post("/invoke", response_model=InvokeResponse)
async def invoke(req: InvokeRequest):
    """Invoke the agent. Replace process_message() with your logic."""
    result = process_message(req.message)
    return InvokeResponse(response=result)


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
