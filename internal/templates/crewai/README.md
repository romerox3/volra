# {{.Name}}

CrewAI multi-agent research crew, created with [Volra](https://github.com/romerox3/volra).

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

# Run a research crew (may take 30-60 seconds)
curl -X POST http://localhost:8000/kickoff \
  -H "Content-Type: application/json" \
  -d '{"topic": "The current state of AI agent frameworks in 2026"}'
```

## Agents

- **Researcher** — Gathers information using search tools
- **Writer** — Produces a structured summary from the research
