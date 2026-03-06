# {{.Name}}

Two-agent coder + reviewer with AutoGen, created with [Volra](https://github.com/romerox3/volra).

## Deploy

```bash
volra deploy
```

Open http://localhost:3001 for monitoring.

## Test

```bash
curl http://localhost:8000/health
curl -X POST http://localhost:8000/solve -H "Content-Type: application/json" -d '{"problem": "Write a function to check if a string is a palindrome"}'
```
