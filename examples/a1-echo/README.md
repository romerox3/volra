# Echo Agent

Minimal FastAPI agent — the simplest possible Volra deployment.

## What it does

- `/health` — health check endpoint
- `/echo` — returns whatever JSON you POST

## Deploy

```bash
cd examples/a1-echo
volra deploy
```

Open http://localhost:3001 to see monitoring.

## Test

```bash
# Health check
curl http://localhost:8000/health

# Echo
curl -X POST http://localhost:8000/echo \
  -H "Content-Type: application/json" \
  -d '{"message": "hello"}'
```

## Stop

```bash
docker compose -f .volra/docker-compose.yml down
```

## Files

| File | Purpose |
|------|---------|
| `main.py` | FastAPI app with 2 endpoints |
| `requirements.txt` | Python dependencies |
| `Agentfile` | Volra deployment config |
