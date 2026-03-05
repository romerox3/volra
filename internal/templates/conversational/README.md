# {{.Name}}

Conversational agent with session memory, created with [Volra](https://github.com/romerox3/volra).

## Setup

```bash
cp .env.example .env
# Edit .env with your values
```

## Deploy

```bash
volra deploy
```

Volra starts your agent, PostgreSQL, and Redis together. Open http://localhost:3001 for monitoring.

## Test

```bash
curl http://localhost:8000/health

# Start a conversation
curl -X POST http://localhost:8000/chat -H "Content-Type: application/json" -d '{"message": "Hello!"}'

# Continue (replace YOUR_SESSION_ID)
curl -X POST http://localhost:8000/chat -H "Content-Type: application/json" -d '{"session_id": "YOUR_SESSION_ID", "message": "Help me"}'
```
