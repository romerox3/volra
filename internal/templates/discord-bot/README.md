# {{.Name}}

AI-powered Discord bot with slash commands, created with [Volra](https://github.com/romerox3/volra).

## Setup

1. Create a Discord bot at https://discord.com/developers/applications
2. Enable the `applications.commands` scope and `Send Messages` permission
3. Copy the bot token

```bash
cp .env.example .env
# Set DISCORD_TOKEN and OPENAI_API_KEY
```

## Deploy

```bash
volra deploy
```

Open http://localhost:3001 for monitoring.

## Usage

In any Discord channel where the bot is present:

```
/ask What is the meaning of life?
```

## Features

- `/ask` slash command with AI responses
- Redis-based rate limiting (1 request per 5 seconds per user)
- Graceful degradation without OpenAI API key
