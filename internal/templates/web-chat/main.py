"""{{.Name}} — full-stack chat UI with WebSocket, deployed with Volra."""

import os
from contextlib import asynccontextmanager

from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.responses import HTMLResponse
from openai import OpenAI

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL") or None
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")
APP_TITLE = "{{.Name}}"

# ---------------------------------------------------------------------------
# LLM
# ---------------------------------------------------------------------------


def generate_response(client: OpenAI | None, message: str) -> str:
    """Generate a response using OpenAI, or echo if no API key."""
    if client is None:
        return f"Echo: {message}"
    resp = client.chat.completions.create(
        model=OPENAI_MODEL,
        messages=[
            {"role": "system", "content": "You are a helpful assistant. Keep responses concise."},
            {"role": "user", "content": message},
        ],
    )
    return resp.choices[0].message.content


# ---------------------------------------------------------------------------
# Chat UI (inline HTML — no subdirectories allowed in Volra templates)
# ---------------------------------------------------------------------------

CHAT_HTML = """<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>""" + APP_TITLE + """</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: system-ui, -apple-system, sans-serif;
    background: #0f172a; color: #e2e8f0;
    display: flex; flex-direction: column; height: 100vh;
  }
  header {
    padding: 16px 24px; background: #1e293b;
    border-bottom: 1px solid #334155;
    font-size: 18px; font-weight: 600;
  }
  header .dot {
    display: inline-block; width: 8px; height: 8px;
    border-radius: 50%; margin-right: 8px;
  }
  header .dot.connected { background: #22c55e; }
  header .dot.disconnected { background: #ef4444; }
  #messages {
    flex: 1; overflow-y: auto; padding: 24px;
    display: flex; flex-direction: column; gap: 12px;
  }
  .msg {
    max-width: 70%; padding: 12px 16px;
    border-radius: 12px; line-height: 1.5;
    white-space: pre-wrap; word-wrap: break-word;
  }
  .msg.user {
    align-self: flex-end; background: #3b82f6; color: white;
    border-bottom-right-radius: 4px;
  }
  .msg.assistant {
    align-self: flex-start; background: #1e293b;
    border: 1px solid #334155; border-bottom-left-radius: 4px;
  }
  .msg.system {
    align-self: center; background: transparent;
    color: #64748b; font-size: 13px; font-style: italic;
  }
  #input-area {
    padding: 16px 24px; background: #1e293b;
    border-top: 1px solid #334155; display: flex; gap: 12px;
  }
  #input {
    flex: 1; padding: 12px 16px; border-radius: 8px;
    border: 1px solid #334155; background: #0f172a;
    color: #e2e8f0; font-size: 15px; outline: none;
  }
  #input:focus { border-color: #3b82f6; }
  #send {
    padding: 12px 24px; border-radius: 8px; border: none;
    background: #3b82f6; color: white; font-size: 15px;
    cursor: pointer; font-weight: 500;
  }
  #send:hover { background: #2563eb; }
  #send:disabled { background: #475569; cursor: not-allowed; }
</style>
</head>
<body>
<header>
  <span class="dot disconnected" id="status-dot"></span>
  """ + APP_TITLE + """
</header>
<div id="messages"></div>
<div id="input-area">
  <input id="input" placeholder="Type a message..." autocomplete="off" />
  <button id="send">Send</button>
</div>
<script>
  const messages = document.getElementById('messages');
  const input = document.getElementById('input');
  const sendBtn = document.getElementById('send');
  const dot = document.getElementById('status-dot');
  let ws;

  function addMsg(text, cls) {
    const div = document.createElement('div');
    div.className = 'msg ' + cls;
    div.textContent = text;
    messages.appendChild(div);
    messages.scrollTop = messages.scrollHeight;
  }

  function connect() {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(proto + '//' + location.host + '/ws');
    ws.onopen = () => {
      dot.className = 'dot connected';
      addMsg('Connected', 'system');
      sendBtn.disabled = false;
    };
    ws.onmessage = (e) => addMsg(e.data, 'assistant');
    ws.onclose = () => {
      dot.className = 'dot disconnected';
      sendBtn.disabled = true;
      addMsg('Disconnected. Reconnecting...', 'system');
      setTimeout(connect, 2000);
    };
  }

  function send() {
    const text = input.value.trim();
    if (!text || !ws || ws.readyState !== WebSocket.OPEN) return;
    addMsg(text, 'user');
    ws.send(text);
    input.value = '';
  }

  sendBtn.addEventListener('click', send);
  input.addEventListener('keydown', (e) => { if (e.key === 'Enter') send(); });
  connect();
  input.focus();
</script>
</body>
</html>"""

# ---------------------------------------------------------------------------
# Application
# ---------------------------------------------------------------------------


@asynccontextmanager
async def lifespan(_app: FastAPI):
    """Initialize OpenAI client on startup."""
    _app.state.openai = OpenAI(api_key=OPENAI_API_KEY, base_url=OPENAI_BASE_URL) if OPENAI_API_KEY else None
    yield


app = FastAPI(title=APP_TITLE, lifespan=lifespan)


@app.get("/health")
async def health():
    """Check if the chat service is ready."""
    llm_configured = app.state.openai is not None
    return {
        "status": "ok" if llm_configured else "degraded",
        "llm_configured": llm_configured,
    }


@app.get("/", response_class=HTMLResponse)
async def index():
    """Serve the chat UI."""
    return CHAT_HTML


@app.websocket("/ws")
async def websocket_endpoint(websocket: WebSocket):
    """Handle WebSocket chat connections."""
    await websocket.accept()
    try:
        while True:
            message = await websocket.receive_text()
            response = generate_response(app.state.openai, message)
            await websocket.send_text(response)
    except WebSocketDisconnect:
        pass


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
