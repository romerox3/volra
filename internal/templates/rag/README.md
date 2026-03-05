# {{.Name}}

RAG agent with ChromaDB vector search and Redis cache, created with [Volra](https://github.com/romerox3/volra).

## Setup

```bash
cp .env.example .env
# Edit .env with your API key (optional — works without it using raw document retrieval)
```

## Deploy

```bash
volra deploy
```

Volra starts your agent, ChromaDB, and Redis together. Open http://localhost:3001 for monitoring.

## Test

```bash
curl http://localhost:8000/health

# Query the knowledge base
curl -X POST http://localhost:8000/query \
  -H "Content-Type: application/json" \
  -d '{"question": "What is Docker?"}'

# Semantic search (not just keyword matching)
curl -X POST http://localhost:8000/query \
  -H "Content-Type: application/json" \
  -d '{"question": "How do I package my application for deployment?"}'
```

## Customization

Replace `SAMPLE_DOCUMENTS` in `main.py` with your own knowledge base. ChromaDB handles embedding automatically.
