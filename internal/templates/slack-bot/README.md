# {{.Name}}

AI-powered Slack bot with event handling, created with [Volra](https://github.com/romerox3/volra).

## Setup

1. Create a Slack app at https://api.slack.com/apps
2. Enable Socket Mode and generate an App-Level Token (xapp-)
3. Add the `chat:write`, `commands`, and `app_mentions:read` scopes
4. Create a `/ask` slash command
5. Install the app to your workspace and copy the Bot Token (xoxb-)

```bash
cp .env.example .env
# Set SLACK_BOT_TOKEN, SLACK_APP_TOKEN, and OPENAI_API_KEY
```

## Deploy

```bash
volra deploy
```

Open http://localhost:3001 for monitoring.

## Usage

In any Slack channel where the bot is added:

```
/ask What are the latest trends in AI?
@bot-name Tell me about Kubernetes
```

## Features

- `/ask` slash command
- `@mention` responses
- Socket Mode (no public URL needed)
