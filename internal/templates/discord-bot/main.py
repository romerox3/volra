"""{{.Name}} — AI-powered Discord bot with slash commands, deployed with Volra."""

import asyncio
import os
import threading
from contextlib import asynccontextmanager

import discord
import redis
from discord import app_commands
from fastapi import FastAPI
from openai import OpenAI

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

DISCORD_TOKEN = os.getenv("DISCORD_TOKEN", "")
OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")

REDIS_HOST = os.getenv("REDIS_HOST", "{{.Name}}-cache")
REDIS_PORT = int(os.getenv("REDIS_PORT", "6379"))

RATE_LIMIT_SECONDS = 5

# ---------------------------------------------------------------------------
# Dependencies
# ---------------------------------------------------------------------------

openai_client = OpenAI(api_key=OPENAI_API_KEY, base_url=OPENAI_BASE_URL) if OPENAI_API_KEY else None
redis_client = redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)

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
            {"role": "system", "content": "You are a helpful Discord bot. Keep responses concise."},
            {"role": "user", "content": message},
        ],
    )
    return resp.choices[0].message.content


# ---------------------------------------------------------------------------
# Rate limiting
# ---------------------------------------------------------------------------


def is_rate_limited(user_id: str) -> bool:
    """Check if a user is rate limited."""
    key = f"ratelimit:{user_id}"
    try:
        if redis_client.exists(key):
            return True
        redis_client.setex(key, RATE_LIMIT_SECONDS, "1")
    except redis.RedisError:
        pass
    return False


# ---------------------------------------------------------------------------
# Discord bot
# ---------------------------------------------------------------------------

intents = discord.Intents.default()
bot = discord.Client(intents=intents)
tree = app_commands.CommandTree(bot)


@tree.command(name="ask", description="Ask the AI a question")
async def ask_command(interaction: discord.Interaction, question: str) -> None:
    """Handle /ask slash command."""
    if is_rate_limited(str(interaction.user.id)):
        await interaction.response.send_message(
            "Please wait a few seconds before asking again.",
            ephemeral=True,
        )
        return

    await interaction.response.defer()
    response = await asyncio.to_thread(generate_response, question)

    if len(response) > 2000:
        response = response[:1997] + "..."
    await interaction.followup.send(response)


@bot.event
async def on_ready() -> None:
    """Sync slash commands when bot connects."""
    await tree.sync()


def run_bot() -> None:
    """Run the Discord bot in a blocking loop."""
    bot.run(DISCORD_TOKEN)


# ---------------------------------------------------------------------------
# Application
# ---------------------------------------------------------------------------


@asynccontextmanager
async def lifespan(_app: FastAPI):
    """Start the Discord bot in a background thread."""
    if DISCORD_TOKEN:
        thread = threading.Thread(target=run_bot, daemon=True)
        thread.start()
    yield


app = FastAPI(title="{{.Name}}", lifespan=lifespan)


@app.get("/health")
async def health():
    """Check bot and dependency connectivity."""
    bot_connected = bot.is_ready() if DISCORD_TOKEN else False
    redis_ok = False
    try:
        redis_ok = redis_client.ping()
    except Exception:
        pass
    is_healthy = bot_connected and redis_ok
    return {
        "status": "ok" if is_healthy else "degraded",
        "bot_connected": bot_connected,
        "redis_connected": redis_ok,
        "llm_configured": openai_client is not None,
    }


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
