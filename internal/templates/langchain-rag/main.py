"""{{.Name}} — LangChain RAG with ChromaDB, deployed with Volra."""

import os
import time
from contextlib import asynccontextmanager

import chromadb
from fastapi import FastAPI
from langchain_chroma import Chroma
from langchain_core.documents import Document
from langchain_core.prompts import ChatPromptTemplate
from langchain_core.runnables import RunnablePassthrough
from langchain_openai import ChatOpenAI, OpenAIEmbeddings
from pydantic import BaseModel

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")

CHROMA_HOST = os.getenv("CHROMA_HOST", "{{.Name}}-chromadb")
CHROMA_PORT = int(os.getenv("CHROMA_PORT", "8000"))

COLLECTION_NAME = "knowledge"

SAMPLE_DOCS = [
    "Python is a high-level programming language known for readability and a rich ecosystem.",
    "Docker containers package code with all dependencies for consistent deployment.",
    "FastAPI is a modern Python framework for building APIs with automatic docs and async support.",
    "Kubernetes orchestrates containers across clusters with automated scaling and self-healing.",
    "PostgreSQL is an advanced open-source relational database with ACID compliance.",
]

RAG_PROMPT = ChatPromptTemplate.from_template(
    "Answer the question based on the context below. "
    "If the context doesn't help, say so.\n\n"
    "Context:\n{context}\n\nQuestion: {question}"
)


def connect_chroma(max_retries: int = 30) -> chromadb.HttpClient:
    for attempt in range(max_retries):
        try:
            client = chromadb.HttpClient(host=CHROMA_HOST, port=CHROMA_PORT)
            client.heartbeat()
            return client
        except Exception:
            if attempt == max_retries - 1:
                raise
            time.sleep(1)
    raise RuntimeError("Cannot connect to ChromaDB")


@asynccontextmanager
async def lifespan(_app: FastAPI):
    chroma_client = connect_chroma()
    _app.state.chroma = chroma_client
    _app.state.llm_configured = bool(OPENAI_API_KEY)

    if OPENAI_API_KEY:
        embeddings = OpenAIEmbeddings(
            openai_api_key=OPENAI_API_KEY,
            openai_api_base=OPENAI_BASE_URL,
        )
        vectorstore = Chroma(
            client=chroma_client,
            collection_name=COLLECTION_NAME,
            embedding_function=embeddings,
        )
        # Seed sample docs if collection is empty
        col = chroma_client.get_or_create_collection(COLLECTION_NAME)
        if col.count() == 0:
            docs = [Document(page_content=d) for d in SAMPLE_DOCS]
            vectorstore.add_documents(docs)
        _app.state.vectorstore = vectorstore
        _app.state.llm = ChatOpenAI(model=OPENAI_MODEL, base_url=OPENAI_BASE_URL)
    else:
        _app.state.vectorstore = None
        _app.state.llm = None
    yield


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


class QueryRequest(BaseModel):
    question: str


class IngestRequest(BaseModel):
    documents: list[str]


@app.get("/health")
async def health():
    chroma_ok = False
    try:
        app.state.chroma.heartbeat()
        chroma_ok = True
    except Exception:
        pass
    return {
        "status": "ok" if chroma_ok else "degraded",
        "chromadb_connected": chroma_ok,
        "llm_configured": app.state.llm_configured,
    }


@app.post("/query")
async def query(req: QueryRequest):
    if app.state.vectorstore is None:
        return {"answer": "[No API key] Set OPENAI_API_KEY.", "sources": 0}
    retriever = app.state.vectorstore.as_retriever(search_kwargs={"k": 3})
    docs = retriever.invoke(req.question)
    context = "\n\n".join(d.page_content for d in docs)
    chain = RAG_PROMPT | app.state.llm
    result = chain.invoke({"context": context, "question": req.question})
    return {"answer": result.content, "sources": len(docs)}


@app.post("/ingest")
async def ingest(req: IngestRequest):
    if app.state.vectorstore is None:
        return {"error": "[No API key] Set OPENAI_API_KEY.", "ingested": 0}
    docs = [Document(page_content=d) for d in req.documents]
    app.state.vectorstore.add_documents(docs)
    return {"ingested": len(docs)}


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
