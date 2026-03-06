# AGENTS.md — AI Coding Agent Instructions

## Project Identity

- **Name:** abaper-mcp
- **Language:** Go 1.25.6
- **Module:** `github.com/bluefunda/abaper-mcp`
- **Type:** MCP (Model Context Protocol) server for SAP ABAP operations
- **Binary:** `abaper-mcp`

## Build & Test (deterministic commands)

```bash
# Install dependencies
make install

# Build binary
make build

# Run all tests
make test

# Format code (must pass before commit)
make fmt

# Lint (requires golangci-lint installed)
make lint

# Build + run locally
make run
```

All commands are idempotent and safe to run repeatedly.

## Architecture (read-only context)

```
AI Assistant (Claude Desktop / CAI)
  → MCP Protocol (stdio / SSE / NATS)
    → abaper-mcp (this repo)
      → abaper-ts REST API (ABAPER_TS_URL)
        → SAP System (ADT)
```

**Critical constraint:** abaper-mcp does NOT connect to SAP directly. All SAP/ADT calls are delegated to the `abaper-ts` REST backend via `APIClient`. The only required runtime config is `ABAPER_TS_URL`.

### Operational Modes

Set via `ABAPER_MODE` env var:

| Mode | Transport | Default |
|------|-----------|---------|
| `stdio` | Standard I/O (Claude Desktop) | Yes |
| `sse` | HTTP/SSE on `:8015` | |
| `nats` | NATS messaging | |
| `dual` | stdio + NATS | |

## File Map

| File | Responsibility | Safe to modify |
|------|---------------|----------------|
| `main.go` | Entry point, mode routing, server wiring | Cautiously |
| `config.go` | Config struct (`AbaperTSURL`, NATS fields), validation | Yes |
| `handlers.go` | `Handlers` struct (holds `Config` + `APIClient`) | Yes |
| `tools.go` | **All MCP tool definitions, input/output types, and handler implementations** | Yes — primary extension point |
| `apiclient.go` | HTTP client for abaper-ts REST API (`APIClient`) | Yes |
| `resources.go` | MCP resource templates (`abap://class/{name}`, etc.) | Yes |
| `prompts.go` | MCP prompt definitions and handlers | Yes |
| `nats_handler.go` | NATS transport for MCP | Cautiously |
| `nats_config.go` | NATS connection config | Cautiously |
| `s4_remediation.go` | S/4HANA compatibility analysis (pattern-based, local) | Yes |
| `s4_remediation_test.go` | Tests for S/4 remediation | Yes |
| `s4_remediation_patterns.json` | Pattern data for S/4 analysis | Yes |
| `vault.go` | HashiCorp Vault secrets loading | Cautiously |
| `internal/logger/logger.go` | Structured logging (zap) — global `logger.L` | Rarely |

### Files you must NOT modify without explicit request

- `.github/workflows/*` — CI/CD pipelines (reusable workflows from `bluefunda/release-foundry`)
- `.goreleaser.yml` — release configuration
- `go.sum` — auto-managed by Go toolchain
- `Dockerfile` — production deployment
- `LICENSE`

## Package Structure

Everything is `package main` except `internal/logger`. There are no sub-packages for tools, resources, or prompts. Do not introduce new packages without explicit request.

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
       // Use h.apiClient for abaper-ts calls
       // Return (&mcp.CallToolResult{IsError: true}, zero, err) on failure
       // Return (nil, output, nil) on success
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

## How to Add a Prompt

In `prompts.go`, register a prompt with arguments and implement the handler.

## API Client Usage

`APIClient` in `apiclient.go` wraps HTTP calls to abaper-ts. All responses follow the envelope:
```json
{"success": true, "data": {...}, "error": "..."}
```

Methods: `post(path, body)` returns `json.RawMessage` (the `data` field) or error. Add new abaper-ts endpoint methods on `APIClient`.

## Testing

- Run: `make test`
- Test files: `*_test.go` in root package
- Currently only `s4_remediation_test.go` exists
- Tests do not require a running SAP system or abaper-ts instance
- When adding features, add corresponding `*_test.go` files

## Commit Conventions

Follow conventional commits:
```
feat(tools): add syntax-check tool
fix(resources): handle missing object gracefully
chore: update dependencies
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `perf`, `security`, `infra`

## Branch Naming

Follow the org convention: `<type>/<short-description>`.

| Type | Example |
|---|---|
| `feat/` | `feat/abaper-tools` |
| `fix/` | `fix/mcp-server-issues` |
| `chore/` | `chore/remove-beta-deploy` |
| `docs/` | `docs/add-agents-md` |
| `infra/` | `infra/add-internal-nats-ca` |

## Pull Request Guidelines

- PR title must use conventional commit format: `feat: ...`, `fix: ...`, `infra: ...`, etc.
- Scoped titles are encouraged: `feat(tools): add syntax-check tool`.
- The PR template is defined at org level (`bluefunda/.github` repo). Do not add a repo-level override unless diverging from the org standard.
- Required sections: Summary, Type (checkbox), Test Plan.
- Customer Impact is required for `feature`, `performance`, and `security` PRs.
- Metrics and Marketing Notes are optional.
- PRs target `main` branch.

## CI/CD

- **CI:** `.github/workflows/ci.yml` — reusable Go CI from `bluefunda/release-foundry`
- **Release:** `.github/workflows/release.yml` — release-please + Docker deploy
- Releases are automated via release-please (conventional commits drive changelog)

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ABAPER_TS_URL` | Yes | `http://localhost:8080` | abaper-ts REST API URL |
| `ABAPER_MODE` | No | `stdio` | Operational mode |
| `LOG_LEVEL` | No | `info` | debug/info/warn/error |
| `LOG_FORMAT` | No | `json` | json/console |

See `.env.template` for full list including NATS and Vault options.

## Do NOT

- Connect to SAP directly — always go through abaper-ts via `APIClient`
- Introduce new Go packages without explicit request
- Modify CI/CD workflows without explicit request
- Add `.env` or credentials to version control
- Change the MCP SDK version without testing all transports
- Use `fmt.Println` for logging — use `logger.L` (zap)
