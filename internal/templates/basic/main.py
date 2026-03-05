"""{{.Name}} — basic agent deployed with Volra."""

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

app = FastAPI(title="{{.Name}}")


@app.get("/health")
async def health():
    return {"status": "ok"}


@app.post("/ask")
async def ask(request: Request):
    body = await request.json()
    message = body.get("message", "")
    return JSONResponse(content={"response": f"You said: {message}"})


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
