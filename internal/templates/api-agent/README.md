# {{.Name}}

Function-calling agent built without any framework, created with [Volra](https://github.com/romerox3/volra).

This template demonstrates the raw mechanics behind every agent framework's tool-calling loop using only the OpenAI SDK.

## Setup

```bash
cp .env.example .env
# Edit .env with your OpenAI API key
```

## Deploy

```bash
volra deploy
```

Open http://localhost:3001 for monitoring.

## Test

```bash
curl http://localhost:8000/health

# The agent uses tools automatically
curl -X POST http://localhost:8000/ask \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the square root of 144?"}'

# Multi-tool usage
curl -X POST http://localhost:8000/ask \
  -H "Content-Type: application/json" \
  -d '{"message": "Fetch the title of https://example.com"}'
```

## Tools

- `calculator` — Evaluate math expressions (supports `math` module)
- `weather` — Stub; replace with a real weather API
- `web_fetch` — Fetch URL content (first 500 chars)
