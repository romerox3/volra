# {{.Name}}

Full-stack chat UI with WebSocket, created with [Volra](https://github.com/romerox3/volra).

## Setup

```bash
cp .env.example .env
# Edit .env with your OpenAI API key (optional — works in echo mode without it)
```

## Deploy

```bash
volra deploy
```

Open http://localhost:8000 for the chat UI.
Open http://localhost:3001 for monitoring.

## Features

- Real-time WebSocket communication
- Dark theme UI with auto-reconnect
- Responsive design (mobile-friendly)
- Graceful degradation without API key (echo mode)
