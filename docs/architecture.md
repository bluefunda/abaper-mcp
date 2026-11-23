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
abaper REST API (http://abaper:8080)
  │
  ▼
SAP System (ADT)
```

## Key Design Decision

ABAPer MCP does **not** connect to SAP directly. It calls **abaper** REST APIs, which handle the actual ADT communication. This separation means:

- abaper-mcp only needs `ABAPER_BACKEND_URL` — no SAP credentials
- abaper manages connection pooling, credential resolution, and ADT protocol details
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
| `get_transport_request` | Read-only: a transport request's header, tasks and objects (`GET /api/v1/transports/{trkey}`) |

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
| `ABAPER_BACKEND_URL` | URL of abaper REST API | `http://localhost:8080` |
| `ABAPER_MODE` | Operational mode (`stdio` or `sse`; anything else is refused) | `stdio` |
| `CAI_BFF_INTERNAL_URL` | cai-bff URL for per-user SAP credentials (required in sse, refused in stdio) | — |
| `INTERNAL_SERVICE_SECRET` | Secret for cai-bff's internal endpoint (required in sse, refused in stdio) | — |
| `ABAPER_REQUIRE_PRINCIPAL` | 401 for SSE requests without `X-BF-User-Id`/`X-BF-Realm` | `true` |
| `ABAPER_HTTP_PORT` | SSE mode listen port | `8015` |
| `ABAPER_HTTP_HOST` | SSE mode listen host | `0.0.0.0` |
| `LOG_LEVEL` | Log level (debug/info/warn/error) | `info` |
| `LOG_FORMAT` | Log format (json/console) | `json` |

## Deployment (Production)

In the ABAPer platform, abaper-mcp runs in **SSE mode** as a Docker container:

```
Container: abaper-mcp (:8015)
  ABAPER_MODE=sse
  ABAPER_BACKEND_URL=http://abaper:8080
  CAI_BFF_INTERNAL_URL=http://cai-bff:8080
  INTERNAL_SERVICE_SECRET=<secret>
```

SAP credentials are per user only: each session's user (`X-BF-User-Id`) has their own SAP credentials fetched
from cai-bff and sent to abaper as `X-SAP-*` headers. There is no shared SAP login; a session without a user, or
whose user has not connected SAP, gets a tool error.

The cai-llm-router connects to it via `http://abaper-mcp:8015/sse` as an MCP server.

## Deployment (Local / Claude Desktop)

For local use with Claude Desktop, run in **stdio mode**:

```json
{
  "mcpServers": {
    "abaper": {
      "command": "/path/to/abaper-mcp",
      "env": {
        "ABAPER_BACKEND_URL": "http://localhost:8080"
      }
    }
  }
}
```

See [CLAUDE_DESKTOP_SETUP.md](../CLAUDE_DESKTOP_SETUP.md) for detailed setup instructions.
