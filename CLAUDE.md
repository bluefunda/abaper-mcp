# CLAUDE.md — abaper-mcp

## What is this?

Go MCP (Model Context Protocol) server for SAP ABAP operations. Delegates all SAP/ADT calls to `abaper-ts` REST backend via `APIClient`.

Module: `github.com/bluefunda/abaper-mcp` | Go 1.25

## Build & Verify

```bash
make build    # Build binary
make test     # Run tests
make fmt      # Format code
make lint     # Lint
```

## Modes

Set via `ABAPER_MODE`: `stdio` (default, Claude Desktop), `sse` (HTTP on :8015), `nats`, `dual` (stdio+NATS).

## Architecture

Everything is `package main` except `internal/logger`. Do not introduce new packages.

- `tools.go` — all MCP tool definitions and handlers (primary extension point)
- `apiclient.go` — HTTP client for abaper-ts REST API
- `resources.go` — MCP resource templates
- `prompts.go` — MCP prompt definitions
- `config.go` — config struct, validation
- `handlers.go` — `Handlers` struct (Config + APIClient)
- `s4_remediation.go` — S/4HANA compatibility analysis

## Key Rules

- Does NOT connect to SAP directly — always through abaper-ts via `APIClient`
- Every handler gets `requestID` via `uuid.New().String()[:8]` and scoped logger
- Use `logger.L` (zap) for logging — never `fmt.Println`
- Normalize object types with `normalizeObjectType()` before API calls

## Conventions

- Commits: conventional format with optional scope (`feat(tools):`, `fix:`)
- Branches: `<type>/<short-description>`
- PRs: conventional commit title, target `main`, squash-merged
