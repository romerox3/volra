# {{.Name}}

HuggingFace smolagents code agent, created with [Volra](https://github.com/romerox3/volra).

The agent writes Python code to solve tasks, combining tools with programmatic logic.

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

# The agent writes and executes code to solve the task
curl -X POST http://localhost:8000/solve \
  -H "Content-Type: application/json" \
  -d '{"task": "What is the population of France?"}'

# Math tasks
curl -X POST http://localhost:8000/solve \
  -H "Content-Type: application/json" \
  -d '{"task": "Calculate the fibonacci sequence up to the 10th number"}'
```

## Tools

- `calculator` — Evaluate math expressions
- `DuckDuckGoSearchTool` — Search the web (built into smolagents)
