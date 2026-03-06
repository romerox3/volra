# {{.Name}}

3-agent dev team (PM, Dev, QA) with CrewAI created with [Volra](https://github.com/romerox3/volra).

## Deploy

```bash
volra deploy
```

Open http://localhost:3001 for monitoring.

## Test

```bash
curl http://localhost:8000/health
curl -X POST http://localhost:8000/kickoff -H "Content-Type: application/json" -d '{"task": "Build a REST API for user management"}'
```
