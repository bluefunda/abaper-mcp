# AGENTS.md — AI Coding Agent Instructions

## Project Identity

- **Name:** abaper-mcp
- **Language:** Go 1.25
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
      → abaper-ts REST API (ABAPER_TS_URL)
        → SAP system (ADT)
```

**Critical constraint:** abaper-mcp does NOT connect to SAP directly. All SAP/ADT calls are delegated to the `abaper-ts` REST backend via `APIClient`. The only required runtime config is `ABAPER_TS_URL`.

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
| `apiclient.go` | HTTP client for abaper-ts REST API (`APIClient`) | Yes |
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
       // Use h.apiClient for abaper-ts calls
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
- Normalize object types with `normalizeObjectType()` before sending to abaper-ts
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

`APIClient` in `apiclient.go` wraps HTTP calls to abaper-ts. All responses follow the envelope:
```json
{"success": true, "data": {...}, "error": "..."}
```

Add new abaper-ts endpoint methods directly on `APIClient`.

## Testing

- Run: `make test`
- Test files: `*_test.go` in root package
- Tests do not require a running SAP system or abaper-ts instance
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
| `ABAPER_TS_URL` | Yes | `http://localhost:8080` | abaper-ts REST API URL |
| `ABAPER_MODE` | No | `stdio` | `stdio` or `sse` |
| `LOG_LEVEL` | No | `info` | `debug`/`info`/`warn`/`error` |
| `LOG_FORMAT` | No | `json` | `json`/`console` |

## Do NOT

- Connect to SAP directly — always go through abaper-ts via `APIClient`
- Introduce new Go packages without explicit request
- Modify CI/CD workflows without explicit request
- Add `.env` or credentials to version control
- Use `fmt.Println` for logging — use `logger.L` (zap)
