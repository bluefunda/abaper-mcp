# abaper-mcp

[![Go Reference](https://pkg.go.dev/badge/github.com/bluefunda/abaper-mcp.svg)](https://pkg.go.dev/github.com/bluefunda/abaper-mcp)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/bluefunda/abaper-mcp)](https://goreportcard.com/report/github.com/bluefunda/abaper-mcp)

A [Model Context Protocol](https://modelcontextprotocol.io) (MCP) server for SAP ABAP development. Connects AI assistants (Claude, Cursor, Windsurf) to live SAP systems via the ABAP Development Tools (ADT) REST API.

## How it works

```
AI assistant (Claude Desktop / Cursor / Windsurf)
  → MCP protocol (stdio or HTTP/SSE)
    → abaper-mcp  ← this repo
      → abaper-ts REST API
        → SAP system (ADT)
```

`abaper-mcp` does not connect to SAP directly. All ADT calls are delegated to an `abaper-ts` backend instance; configure its URL via `ABAPER_TS_URL`.

## Installation

```bash
go install github.com/bluefunda/abaper-mcp@latest
```

Or build from source:

```bash
git clone https://github.com/bluefunda/abaper-mcp.git
cd abaper-mcp
make build
```

## Quick start: Claude Desktop

Add to your Claude Desktop config:

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

Restart Claude Desktop. The server starts automatically via stdio.

See [`examples/`](examples/) for Claude Desktop, Cursor, and Docker Compose configs.

## Modes

Set via `ABAPER_MODE` env var:

| Mode | Transport | Use case |
|------|-----------|----------|
| `stdio` (default) | Standard I/O | Claude Desktop, Claude Code CLI |
| `sse` | HTTP/SSE on `:8015` | Orchestrators, programmatic access |

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ABAPER_TS_URL` | Yes | `http://localhost:8080` | abaper-ts REST API base URL |
| `ABAPER_MODE` | No | `stdio` | `stdio` or `sse` |
| `LOG_LEVEL` | No | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | No | `json` | `json` or `console` |

## MCP capabilities

### Tools

| Tool | Description |
|------|-------------|
| `get-object` | Retrieve source code for any ABAP object |
| `search-objects` | Search objects by pattern with wildcard support |
| `list-packages` | List all ABAP packages |
| `test-connection` | Verify connectivity to abaper-ts |
| `create-object` | Create a new ABAP object with source |
| `update-object` | Update source code of an existing object |
| `activate-object` | Activate an ABAP object |
| `create-transport` | Create a Workbench transport request |
| `syntax-check` | Run ABAP syntax check |
| `run-unit-tests` | Execute ABAP Unit tests |
| `format-code` | Format ABAP source code |
| `analyze-s4-remediation` | Analyze S/4HANA compatibility issues |

### Resources

Access ABAP objects by URI:

```
abap://program/{name}
abap://class/{name}
abap://function/{group}/{name}
abap://interface/{name}
abap://table/{name}
abap://structure/{name}
abap://include/{name}
abap://packages
```

### Prompts

Pre-configured AI workflows:

| Prompt | Description |
|--------|-------------|
| `analyze-abap` | Code quality, performance, and security analysis |
| `review-abap` | Detailed code review with best practices |
| `optimize-abap` | Performance optimization suggestions |
| `document-abap` | Generate comprehensive documentation |
| `test-abap` | Generate ABAP Unit test code |
| `refactor-abap` | Refactoring suggestions with examples |
| `explain-abap` | Explain code in simple terms |

## Development

```bash
make build   # Build binary
make test    # Run tests
make fmt     # Format code
make lint    # Run golangci-lint
```

### Project structure

```
main.go                    # Entry point, mode routing
config.go                  # Config struct, validation
handlers.go                # Handlers struct (Config + APIClient)
tools.go                   # All MCP tool definitions and handlers
resources.go               # MCP resource templates
prompts.go                 # MCP prompt definitions
apiclient.go               # HTTP client for abaper-ts REST API
s4_remediation.go          # S/4HANA compatibility analysis
internal/logger/           # Structured logging (zap)
examples/                  # Claude Desktop, Cursor, Docker configs
```

Everything is `package main`. Do not introduce sub-packages.

## License

Apache 2.0 — see [LICENSE](LICENSE).

Authored by Amish Kushwaha, open-sourced under Apache 2.0 by BlueFunda, Inc.
