# ABAPer MCP Architecture

## Overview

ABAPer MCP is a Model Context Protocol (MCP) server that exposes SAP ABAP operations as tools, resources, and prompts. It enables AI assistants (Claude, LLMs via CAI) to interact with SAP systems programmatically.

## How It Works

```
AI Assistant (Claude Desktop, CAI, etc.)
  │
  ▼
MCP Protocol (stdio / SSE / NATS)
  │
  ▼
abaper-mcp (Go)
  ├── Tools: get-object, create-class, activate-object, etc.
  ├── Resources: abap://class/ZCL_TEST, abap://program/ZTEST
  └── Prompts: analyze-abap, review-abap, test-abap, etc.
  │
  ▼
abaper-ts REST API (http://abaper-ts:8080)
  │
  ▼
SAP System (ADT)
```

## Key Design Decision

ABAPer MCP does **not** connect to SAP directly. It calls **abaper-ts** REST APIs, which handle the actual ADT communication. This separation means:

- abaper-mcp only needs `ABAPER_TS_URL` — no SAP credentials
- abaper-ts manages connection pooling, credential resolution, and ADT protocol details
- abaper-mcp focuses on MCP protocol and tool orchestration

## Operational Modes

| Mode | Transport | Use Case |
|------|-----------|----------|
| `stdio` | Standard I/O | Claude Desktop, local AI tools |
| `sse` | HTTP/SSE on port 8015 | CAI platform (cai-llm-router) |
| `nats` | NATS messaging | Orchestrator integration |
| `dual` | stdio + NATS | Both local and remote access |

Set via `ABAPER_MODE` environment variable (default: `stdio`).

## MCP Tools

### Object Operations

| Tool | Description |
|------|-------------|
| `get-object` | Retrieve source code (program, class, function, interface, table, structure, include) |
| `search-objects` | Search by name pattern with wildcards |
| `list-packages` | List all ABAP packages |
| `create-program` | Create a new ABAP program with source |
| `create-class` | Create a new ABAP class with source |
| `update-program` | Update existing program source (complete, not diffs) |
| `update-class` | Update existing class source (complete, not diffs) |
| `activate-object` | Activate an ABAP object |

### Development Tools

| Tool | Description |
|------|-------------|
| `syntax-check` | Check ABAP source for syntax errors |
| `format-code` | Format source with SAP pretty printer |
| `run-unit-tests` | Execute ABAP unit tests |
| `analyze-s4-remediation` | Analyze code for S/4HANA compatibility |

### Transport Tools

| Tool | Description |
|------|-------------|
| `transport-info` | Get transport request information |
| `create-transport` | Create a new transport request |

## MCP Resources

URI-based access to ABAP objects:

- `abap://program/{name}` — ABAP programs
- `abap://class/{name}` — Classes
- `abap://function/{name}` — Function modules
- `abap://interface/{name}` — Interfaces
- `abap://table/{name}` — Database tables
- `abap://structure/{name}` — Structures
- `abap://include/{name}` — Include programs
- `abap://packages` — Package listing

## MCP Prompts

Pre-configured analysis workflows:

| Prompt | Purpose |
|--------|---------|
| `analyze-abap` | Comprehensive quality, performance, and security analysis |
| `review-abap` | Code review with best practices |
| `optimize-abap` | Performance optimization suggestions |
| `document-abap` | Generate comprehensive documentation |
| `test-abap` | Generate ABAP unit test code |
| `refactor-abap` | Refactoring suggestions with examples |
| `explain-abap` | Explain code in simple terms |

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `ABAPER_TS_URL` | URL of abaper-ts REST API | `http://localhost:8080` |
| `ABAPER_MODE` | Operational mode (stdio/sse/nats/dual) | `stdio` |
| `ABAPER_HTTP_PORT` | SSE mode listen port | `8015` |
| `ABAPER_HTTP_HOST` | SSE mode listen host | `0.0.0.0` |
| `LOG_LEVEL` | Log level (debug/info/warn/error) | `info` |
| `LOG_FORMAT` | Log format (json/console) | `json` |
| `NATS_URL` | NATS server URL (for nats/dual mode) | — |
| `NATS_CREDS` | NATS credentials file path | — |
| `VAULT_ADDR` | HashiCorp Vault address (optional) | — |
| `VAULT_TOKEN` | Vault token (optional) | — |

## Deployment (Production)

In the ABAPer platform, abaper-mcp runs in **SSE mode** as a Docker container:

```
Container: abaper-mcp (:8015)
  ABAPER_MODE=sse
  ABAPER_TS_URL=http://abaper-ts:8080
```

The cai-llm-router connects to it via `http://abaper-mcp:8015/sse` as an MCP server.

## Deployment (Local / Claude Desktop)

For local use with Claude Desktop, run in **stdio mode**:

```json
{
  "mcpServers": {
    "abaper": {
      "command": "/path/to/abaper-mcp",
      "env": {
        "ABAPER_TS_URL": "http://localhost:8080"
      }
    }
  }
}
```

See [CLAUDE_DESKTOP_SETUP.md](../CLAUDE_DESKTOP_SETUP.md) for detailed setup instructions.
