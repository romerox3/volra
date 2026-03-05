# {{.Name}}

LangGraph ReAct agent with tool-calling loop, created with [Volra](https://github.com/romerox3/volra).

## Setup

```bash
cp .env.example .env
# Edit .env with your OpenAI API key
```

## Deploy

```bash
volra deploy
```

Open http://localhost:3001 for monitoring (Level 2 observability enabled).

## Test

```bash
curl http://localhost:8000/health

# Ask a question (the agent will use tools as needed)
curl -X POST http://localhost:8000/invoke \
  -H "Content-Type: application/json" \
  -d '{"message": "What is 42 * 17?"}'

# Use thread memory (same thread_id = continued conversation)
curl -X POST http://localhost:8000/invoke \
  -H "Content-Type: application/json" \
  -d '{"message": "What was the result?", "thread_id": "my-session"}'
```

## Tools

- `calculator` — Evaluate math expressions
- `current_datetime` — Get the current date and time
- `web_search` — Stub; replace with Tavily, Serper, or similar
