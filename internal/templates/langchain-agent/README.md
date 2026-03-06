# {{.Name}}

LangChain AgentExecutor with ReAct tools created with [Volra](https://github.com/romerox3/volra).

## Deploy

```bash
volra deploy
```

Open http://localhost:3001 for monitoring.

## Test

```bash
curl http://localhost:8000/health
curl -X POST http://localhost:8000/invoke -H "Content-Type: application/json" -d '{"message": "What is 42 * 17?"}'
```
