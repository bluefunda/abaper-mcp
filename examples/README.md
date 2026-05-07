# Examples

Configuration examples for deploying `abaper-mcp` with various AI assistants and runtimes.

## claude-desktop

Adds `abaper-mcp` to Claude Desktop as a local MCP server (stdio mode).

Copy the config snippet into your Claude Desktop config file and update `ABAPER_TS_URL`:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`  
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "abaper": {
      "command": "/usr/local/bin/abaper-mcp",
      "env": {
        "ABAPER_TS_URL": "https://your-abaper-ts-host"
      }
    }
  }
}
```

See [`claude-desktop/claude_desktop_config.json`](claude-desktop/claude_desktop_config.json).

## cursor

Adds `abaper-mcp` to [Cursor](https://cursor.sh) (also works for Windsurf).

Create or edit `.cursor/mcp.json` in your project root (or the global `~/.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "abaper": {
      "command": "/usr/local/bin/abaper-mcp",
      "env": {
        "ABAPER_TS_URL": "https://your-abaper-ts-host"
      }
    }
  }
}
```

See [`cursor/mcp.json`](cursor/mcp.json).

## docker

Runs `abaper-mcp` in SSE mode alongside `abaper-ts` using Docker Compose.

```bash
cd examples/docker
# Set ABAPER_TS_URL in docker-compose.yml to point at your abaper-ts instance
docker compose up -d
```

The MCP endpoint is available at `http://localhost:8015/`. Configure your orchestrator or MCP client to connect there.

See [`docker/docker-compose.yml`](docker/docker-compose.yml).
