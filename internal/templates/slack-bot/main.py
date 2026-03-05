"""{{.Name}} — AI-powered Slack bot with event handling, deployed with Volra."""

import os
import threading
from contextlib import asynccontextmanager

from fastapi import FastAPI
from openai import OpenAI
from slack_bolt import App as SlackApp
from slack_bolt.adapter.socket_mode import SocketModeHandler

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

SLACK_BOT_TOKEN = os.getenv("SLACK_BOT_TOKEN", "")
SLACK_APP_TOKEN = os.getenv("SLACK_APP_TOKEN", "")
OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")

# ---------------------------------------------------------------------------
# Dependencies
# ---------------------------------------------------------------------------

openai_client = OpenAI(api_key=OPENAI_API_KEY, base_url=OPENAI_BASE_URL) if OPENAI_API_KEY else None

# ---------------------------------------------------------------------------
# LLM
# ---------------------------------------------------------------------------


def generate_response(message: str) -> str:
    """Generate a response using OpenAI, or echo if no API key."""
    if openai_client is None:
        return f"[No API key] Echo: {message}"
    resp = openai_client.chat.completions.create(
        model=OPENAI_MODEL,
        messages=[
            {"role": "system", "content": "You are a helpful Slack bot. Keep responses concise."},
            {"role": "user", "content": message},
        ],
    )
    return resp.choices[0].message.content


# ---------------------------------------------------------------------------
# Slack bot
# ---------------------------------------------------------------------------

slack_app = SlackApp(token=SLACK_BOT_TOKEN) if SLACK_BOT_TOKEN else None
slack_handler = None


if slack_app is not None:

    @slack_app.command("/ask")
    def handle_ask_command(ack, command, respond):
        """Handle /ask slash command."""
        ack()
        response = generate_response(command["text"])
        respond(response)

    @slack_app.event("app_mention")
    def handle_mention(event, say):
        """Respond when the bot is mentioned."""
        text = event.get("text", "")
        # Remove the bot mention from the message
        parts = text.split(">", 1)
        question = parts[1].strip() if len(parts) > 1 else text
        response = generate_response(question)
        say(response)


def run_slack_bot() -> None:
    """Run the Slack bot in Socket Mode."""
    global slack_handler
    if slack_app is not None and SLACK_APP_TOKEN:
        slack_handler = SocketModeHandler(slack_app, SLACK_APP_TOKEN)
        slack_handler.start()


# ---------------------------------------------------------------------------
# Application
# ---------------------------------------------------------------------------


@asynccontextmanager
async def lifespan(_app: FastAPI):
    """Start the Slack bot in a background thread."""
    if SLACK_BOT_TOKEN and SLACK_APP_TOKEN:
        thread = threading.Thread(target=run_slack_bot, daemon=True)
        thread.start()
    yield


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


@app.get("/health")
async def health():
    """Check bot configuration status."""
    bot_configured = bool(SLACK_BOT_TOKEN and SLACK_APP_TOKEN)
    bot_connected = slack_handler is not None and slack_handler.client is not None
    return {
        "status": "ok" if bot_configured else "degraded",
        "bot_configured": bot_configured,
        "bot_connected": bot_connected,
        "llm_configured": openai_client is not None,
    }


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
