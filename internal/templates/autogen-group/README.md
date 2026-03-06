# {{.Name}}

3+ agent group chat with approval flow, created with [Volra](https://github.com/romerox3/volra).

## Deploy

```bash
volra deploy
```

Open http://localhost:3001 for monitoring.

## Test

```bash
curl http://localhost:8000/health
curl -X POST http://localhost:8000/discuss -H "Content-Type: application/json" -d '{"topic": "Design a caching strategy for a REST API"}'
```
