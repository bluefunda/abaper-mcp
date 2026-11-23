# AGENTS.md — AI Coding Agent Instructions

## Project Identity

- **Name:** abaper-mcp
- **Language:** Go 1.26
- **Module:** `github.com/bluefunda/abaper-mcp`
- **Type:** MCP (Model Context Protocol) server for SAP ABAP operations
- **Binary:** `abaper-mcp`

## Build & Test

```bash
make build    # Build binary
make test     # Run tests
make fmt      # Format code (must pass before commit)
make lint     # Lint (requires golangci-lint)
```

All four commands must pass before any change is considered complete.

## Architecture

```
AI assistant (Claude Desktop / Cursor / Windsurf)
  → MCP protocol (stdio or HTTP/SSE)
    → abaper-mcp  ← this repo
      → abaper REST API (ABAPER_BACKEND_URL)
        → SAP system (ADT)
```

**Critical constraint:** abaper-mcp does NOT connect to SAP directly. All SAP/ADT calls are delegated to the `abaper` REST backend via `APIClient`. The only required runtime config in stdio mode is `ABAPER_BACKEND_URL`; `sse` mode also requires `CAI_BFF_INTERNAL_URL` and `INTERNAL_SERVICE_SECRET`.

**SAP credentials are per user.** In `sse` mode each session's user (`X-BF-User-Id`) gets their own SAP credentials from cai-bff, sent to abaper as `X-SAP-*` headers; a session without a user or without connected SAP fails its tool calls. Never add a shared/default SAP identity for server mode. stdio mode is local and single-user and sends no SAP identity (the user runs `abaper serve --allow-static-sap` with their own credentials).

## Operational Modes

Set via `ABAPER_MODE`:

| Mode | Transport | Default |
|------|-----------|---------|
| `stdio` | Standard I/O (Claude Desktop, Claude Code) | Yes |
| `sse` | HTTP/SSE on `:8015` | |

## File Map

| File | Responsibility | Safe to modify |
|------|---------------|----------------|
| `main.go` | Entry point, mode routing, server wiring | Cautiously |
| `config.go` | Config struct (`AbaperTSURL`), validation | Yes |
| `handlers.go` | `Handlers` struct (holds `Config` + `APIClient`) | Yes |
| `tools.go` | **All MCP tool definitions, input/output types, and handler implementations** | Yes — primary extension point |
| `apiclient.go` | HTTP client for abaper REST API (`APIClient`) | Yes |
| `resources.go` | MCP resource templates (`abap://class/{name}`, etc.) | Yes |
| `prompts.go` | MCP prompt definitions and handlers | Yes |
| `s4_remediation.go` | S/4HANA compatibility analysis (pattern-based, local) | Yes |
| `s4_remediation_test.go` | Tests for S/4 remediation | Yes |
| `s4_remediation_patterns.json` | Pattern data for S/4 analysis | Yes |
| `internal/logger/logger.go` | Structured logging (zap) — global `logger.L` | Rarely |

### Files you must NOT modify without explicit request

- `.github/workflows/*` — CI/CD pipelines
- `.goreleaser.yml` — release configuration
- `go.sum` — auto-managed
- `Dockerfile` — production deployment
- `LICENSE`

## Package Structure

Everything is `package main` except `internal/logger`. Do not introduce new packages without explicit request.

## How to Add a Tool

1. **Define input/output structs** in `tools.go` with `json` and `jsonschema` tags:
   ```go
   type MyToolInput struct {
       Name string `json:"name" jsonschema:"required,description=Object name"`
   }
   type MyToolOutput struct {
       Result string `json:"result" jsonschema:"description=Result"`
   }
   ```

2. **Implement handler** on `Handlers` in `tools.go`:
   ```go
   func (h *Handlers) HandleMyTool(ctx context.Context, req *mcp.CallToolRequest,
       input MyToolInput) (*mcp.CallToolResult, MyToolOutput, error) {
       requestID := uuid.New().String()[:8]
       log := logger.WithTool(requestID, "my-tool")
       log.Info("my-tool start", zap.String("name", input.Name))
       // Use h.apiClient for abaper calls
       return nil, output, nil
   }
   ```

3. **Register** in `registerTools()` at the top of `tools.go`:
   ```go
   mcp.AddTool(server, &mcp.Tool{
       Name:        "my-tool",
       Description: "What the tool does",
   }, handlers.HandleMyTool)
   ```

### Patterns to follow

- Every handler gets a `requestID` via `uuid.New().String()[:8]` and a scoped logger via `logger.WithTool(requestID, "tool-name")`
- Log start, completion, and errors with `zap` structured fields
- Normalize object types with `normalizeObjectType()` before sending to abaper
- Return `*mcp.CallToolResult{IsError: true}` for user-facing errors

## How to Add a Resource

In `resources.go`, register a template and implement the handler:
```go
server.AddResourceTemplate(&mcp.ResourceTemplate{
    URITemplate: "abap://mytype/{name}",
    Name:        "ABAP MyType",
    Description: "...",
    MIMEType:    "text/x-abap",
}, handlers.HandleMyTypeResource)
```

## API Client Usage

`APIClient` in `apiclient.go` wraps HTTP calls to abaper. All responses follow the envelope:
```json
{"success": true, "data": {...}, "error": "..."}
```

Add new abaper endpoint methods directly on `APIClient`.

## Testing

- Run: `make test`
- Test files: `*_test.go` in root package
- Tests do not require a running SAP system or abaper instance
- Add `*_test.go` files for any new or modified code

## Code Conventions

- All `.go` files must have the Apache 2.0 license header (`// Copyright 2025 bluefunda // Licensed under...`)
- Use `logger.L` (zap) for logging — never `fmt.Println`
- Normalize object types with `normalizeObjectType()` before API calls
- Object names and types are always uppercased before API calls

## Commit Conventions

Follow conventional commits:
```
feat(tools): add syntax-check tool
fix(resources): handle missing object gracefully
chore: update dependencies
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `perf`, `security`, `infra`

## Branch Naming

`<type>/<short-description>` — e.g., `feat/add-transport-tool`, `fix/missing-object-error`

## Pull Request Guidelines

- PR title must use conventional commit format
- PRs target `main`, squash-merged
- The PR template is at org level (`bluefunda/.github`)

## CI/CD

- **CI:** `.github/workflows/ci.yml` — reusable Go CI from `bluefunda/release-foundry`
- **Release:** `.github/workflows/release.yml` — release-please + Docker deploy to `ghcr.io`

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ABAPER_BACKEND_URL` | Yes | `http://localhost:8080` | abaper REST API URL |
| `ABAPER_MODE` | No | `stdio` | `stdio` or `sse`; anything else is refused |
| `CAI_BFF_INTERNAL_URL` | `sse` | — | cai-bff URL for per-user SAP credentials; refused in stdio |
| `INTERNAL_SERVICE_SECRET` | `sse` | — | Secret for cai-bff's internal endpoint; refused in stdio |
| `ABAPER_REQUIRE_PRINCIPAL` | No | `true` | 401 for SSE requests without `X-BF-User-Id`/`X-BF-Realm` |
| `LOG_LEVEL` | No | `info` | `debug`/`info`/`warn`/`error` |
| `LOG_FORMAT` | No | `json` | `json`/`console` |

## Do NOT

- Connect to SAP directly — always go through abaper via `APIClient`
- Introduce new Go packages without explicit request
- Modify CI/CD workflows without explicit request
- Add `.env` or credentials to version control
- Fall back to a shared/default SAP identity in `sse` mode, or read `SAP_*` env
- Use `fmt.Println` for logging — use `logger.L` (zap)
