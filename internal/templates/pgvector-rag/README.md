# {{.Name}}

Hybrid search (vector + keyword) with pgvector, created with [Volra](https://github.com/romerox3/volra).

## Deploy

```bash
volra deploy
```

Open http://localhost:3001 for monitoring.

## Test

```bash
curl http://localhost:8000/health
curl -X POST http://localhost:8000/query -H "Content-Type: application/json" -d '{"question": "What is pgvector?"}'
curl -X POST http://localhost:8000/ingest -H "Content-Type: application/json" -d '{"documents": ["New fact..."]}'
```
