import os
from fastapi import FastAPI

app = FastAPI()
SECRET = os.getenv("SECRET_KEY")

@app.get("/healthz")
def healthz():
    return {"ok": True}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, port=9000)
