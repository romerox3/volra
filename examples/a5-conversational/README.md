# Conversational Agent

Session-based chat agent with PostgreSQL for persistence and Redis for response cache. Demonstrates multi-service deployments.

## What it does

- `/health` — health check (reports Postgres + Redis status)
- `/chat` — send a message, get a response. Sessions are stored in PostgreSQL.

## Prerequisites

Create a `.env` file:

```bash
cp .env.example .env
# Edit .env with your values (defaults work for local testing)
```

## Deploy

```bash
cd examples/a5-conversational
volra deploy
```

Volra starts your agent, PostgreSQL, and Redis together. Open http://localhost:3001 for monitoring.

## Test

```bash
# Health check
curl http://localhost:8000/health

# Start a conversation
curl -X POST http://localhost:8000/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello!"}'

# Continue the conversation (use the session_id from the response)
curl -X POST http://localhost:8000/chat \
  -H "Content-Type: application/json" \
  -d '{"session_id": "YOUR_SESSION_ID", "message": "Help me with something"}'
```

## Stop

```bash
docker compose -f .volra/docker-compose.yml down
```

## Agentfile highlights

```yaml
framework: langgraph
services:
  redis:
    image: redis:7-alpine
    port: 6379
  postgres:
    image: postgres:16-alpine
    port: 5432
    volumes:
      - /var/lib/postgresql/data
    env:
      - POSTGRES_PASSWORD
      - POSTGRES_DB
```

Multiple services are orchestrated automatically. PostgreSQL data persists across redeploys via Docker volumes.

## Files

| File | Purpose |
|------|---------|
| `main.py` | FastAPI app with chat sessions + mock LLM |
| `requirements.txt` | Python dependencies (fastapi, redis, psycopg2) |
| `Agentfile` | Volra config with Redis + PostgreSQL |
