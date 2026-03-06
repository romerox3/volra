# {{.Name}}

Single research agent with web scraping tools created with [Volra](https://github.com/romerox3/volra).

## Deploy

```bash
volra deploy
```

Open http://localhost:3001 for monitoring.

## Test

```bash
curl http://localhost:8000/health
curl -X POST http://localhost:8000/research -H "Content-Type: application/json" -d '{"topic": "AI agents in production"}'
```
