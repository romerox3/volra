"""Echo Agent — minimal smoke test for Volra."""

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

app = FastAPI(title="Echo Agent")


@app.get("/health")
async def health():
    return {"status": "ok"}


@app.post("/echo")
async def echo(request: Request):
    body = await request.json()
    return JSONResponse(content=body)


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
