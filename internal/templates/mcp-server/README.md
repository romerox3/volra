# {{.Name}}

MCP-compatible tool server, created with [Volra](https://github.com/romerox3/volra).

Deploy your own MCP server that works with Claude Desktop, Claude Code, Cursor, and any MCP-compatible client.

## Deploy

```bash
volra deploy
```

Open http://localhost:3001 for monitoring.

## Test

```bash
curl http://localhost:8000/health

# MCP protocol is available at /mcp
# Connect from any MCP client using: http://localhost:8000/mcp
```

## MCP Client Configuration

Add to your MCP client config (e.g. Claude Desktop):

```json
{
  "mcpServers": {
    "{{.Name}}": {
      "url": "http://localhost:8000/mcp"
    }
  }
}
```

## Tools

- `text_length` — Count characters in text
- `word_frequency` — Analyze word frequency (top 20)
- `search_data` — Search sample data by keyword (replace with your own)
