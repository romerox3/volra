# RAG Knowledge Base

In-memory knowledge base with Redis query cache. Demonstrates Volra service dependencies.

## What it does

- `/health` — health check (reports Redis connection status)
- `/query` — keyword search against an in-memory KB, results cached in Redis

## Deploy

```bash
cd examples/a4-rag-kb
volra deploy
```

Volra automatically starts Redis alongside your agent. Open http://localhost:3001 for monitoring.

## Test

```bash
# Health check
curl http://localhost:8000/health

# Query the KB
curl -X POST http://localhost:8000/query \
  -H "Content-Type: application/json" \
  -d '{"question": "What is Docker?"}'

# Second call hits Redis cache
curl -X POST http://localhost:8000/query \
  -H "Content-Type: application/json" \
  -d '{"question": "What is Docker?"}'
```

## Stop

```bash
docker compose -f .volra/docker-compose.yml down
```

## Agentfile highlights

```yaml
services:
  redis-cache:
    image: redis:7-alpine
    port: 6379
```

Services declared in the Agentfile are automatically included in the Docker Compose stack with networking, health checks, and resource limits.

## Files

| File | Purpose |
|------|---------|
| `main.py` | FastAPI app with KB search + Redis cache |
| `requirements.txt` | Python dependencies (fastapi, redis) |
| `Agentfile` | Volra config with Redis service |
