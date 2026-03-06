# {{.Name}}

OpenAI Assistants API with threads and code interpreter created with [Volra](https://github.com/romerox3/volra).

## Deploy

```bash
volra deploy
```

Open http://localhost:3001 for monitoring.

## Test

```bash
curl http://localhost:8000/health
curl -X POST http://localhost:8000/ask -H "Content-Type: application/json" -d '{"message": "Calculate the factorial of 10"}'
```
