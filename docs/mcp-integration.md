# MCP Integration

Volra includes a built-in [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server, allowing AI-powered editors to deploy, monitor, and debug agents directly.

## Available Tools

| Tool | Description |
|------|-------------|
| `volra_deploy` | Deploy agent from project directory |
| `volra_status` | Check agent health and container states |
| `volra_logs` | Get recent logs from agent or services |
| `volra_doctor` | Run system prerequisite checks |

## Editor Configuration

### Claude Code

Add to `~/.claude/claude_desktop_config.json` or your project's `.mcp.json`:

```json
{
  "mcpServers": {
    "volra": {
      "command": "volra",
      "args": ["mcp"]
    }
  }
}
```

### Cursor

Add to `.cursor/mcp.json` in your project root:

```json
{
  "mcpServers": {
    "volra": {
      "command": "volra",
      "args": ["mcp"]
    }
  }
}
```

### VS Code (with MCP extension)

Add to `.vscode/settings.json`:

```json
{
  "mcp.servers": {
    "volra": {
      "command": "volra",
      "args": ["mcp"]
    }
  }
}
```

## Usage Examples

Once configured, your editor's AI assistant can use Volra tools directly:

- "Deploy this agent" → calls `volra_deploy`
- "What's the status of my agent?" → calls `volra_status`
- "Show me the last 100 lines of logs" → calls `volra_logs`
- "Run diagnostics" → calls `volra_doctor`

## How It Works

`volra mcp` starts a JSON-RPC 2.0 server over stdio. The editor sends requests on stdin and reads responses from stdout. Debug logs go to stderr.

The server exposes Volra's core operations as MCP tools with typed input schemas, so AI assistants can discover and invoke them automatically.
