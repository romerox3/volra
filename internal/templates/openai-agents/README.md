# {{.Name}}

OpenAI Agents SDK with tools and handoffs, created with [Volra](https://github.com/romerox3/volra).

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

# Ask a general question
curl -X POST http://localhost:8000/run \
  -H "Content-Type: application/json" \
  -d '{"message": "What time is it?"}'

# Math question (auto-handoff to Math Specialist)
curl -X POST http://localhost:8000/run \
  -H "Content-Type: application/json" \
  -d '{"message": "Calculate the compound interest on $1000 at 5% for 3 years"}'
```

## Agents

- **Assistant** — Primary agent with datetime tool and handoff routing
- **Math Specialist** — Handles math questions with calculator tool
