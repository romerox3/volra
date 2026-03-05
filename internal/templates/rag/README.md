# {{.Name}}

RAG agent with Redis cache, created with [Volra](https://github.com/romerox3/volra).

## Setup

```bash
cp .env.example .env
# Edit .env with your API key
```

## Deploy

```bash
volra deploy
```

Volra starts your agent and Redis together. Open http://localhost:3001 for monitoring.

## Test

```bash
curl http://localhost:8000/health
curl -X POST http://localhost:8000/query -H "Content-Type: application/json" -d '{"question": "What is Docker?"}'
```
