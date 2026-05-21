// Copyright 2025 BlueFunda, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// abaper-mcp is a Model Context Protocol (MCP) server for SAP ABAP development.
//
// It bridges AI assistants (Claude Desktop, Claude Code, Cursor, Windsurf) to
// live SAP systems by exposing ABAP operations as MCP tools, resources, and
// prompts. All ADT calls are delegated to an abaper-ts REST backend; this
// binary does not connect to SAP directly.
//
// # Installation
//
//	go install github.com/bluefunda/abaper-mcp@latest
//
// # Quick start — Claude Desktop
//
// Add to your Claude Desktop config
// (~/.config/Claude/claude_desktop_config.json on macOS):
//
//	{
//	  "mcpServers": {
//	    "abaper": {
//	      "command": "/usr/local/bin/abaper-mcp",
//	      "env": { "ABAPER_TS_URL": "https://your-abaper-ts-host" }
//	    }
//	  }
//	}
//
// # Configuration
//
// All configuration is via environment variables:
//
//	ABAPER_TS_URL      URL of the abaper-ts backend (default: http://localhost:8080)
//	ABAPER_MODE        Transport: stdio (default) or sse
//	ABAPER_HTTP_PORT   HTTP port for SSE mode (default: 8015)
//	ABAPER_HTTP_HOST   HTTP host for SSE mode (default: 0.0.0.0)
//	S4_TEMPORAL_URL    URL of the s4-temporal API (optional)
//	LOG_LEVEL          Log level: debug, info, warn, error (default: info)
//	LOG_FORMAT         Log format: json (default) or console
//
// # Transport modes
//
// stdio (default) — for Claude Desktop and IDE extensions:
//
//	abaper-mcp
//
// sse — HTTP/SSE server for remote or container deployments:
//
//	ABAPER_MODE=sse abaper-mcp
//
// # MCP tools
//
// The server exposes the following tools to MCP clients:
//
//   - get-object         — retrieve source code for any ABAP object
//   - search-objects     — search objects by pattern with wildcard support
//   - list-packages      — list all ABAP packages
//   - test-connection    — verify connectivity to abaper-ts
//   - create-object      — create a new ABAP object with source
//   - update-object      — update source code of an existing object
//   - activate-object    — activate an ABAP object after editing
//   - create-transport   — create a Workbench transport request
//   - syntax-check       — run ABAP syntax check
//   - run-unit-tests     — execute ABAP Unit tests
//   - format-code        — format ABAP source code
//   - analyze-s4-remediation — analyze S/4HANA compatibility issues
//
// # MCP resources
//
// ABAP objects are accessible by URI scheme:
//
//	abap://program/{name}
//	abap://class/{name}
//	abap://function/{group}/{name}
//	abap://interface/{name}
//	abap://table/{name}
//	abap://structure/{name}
//	abap://include/{name}
//	abap://packages
//
// # MCP prompts
//
// Pre-configured AI workflows:
//
//   - analyze-abap   — code quality, performance, and security analysis
//   - review-abap    — detailed code review with best practices
//   - optimize-abap  — performance optimization suggestions
//   - document-abap  — generate comprehensive documentation
//   - test-abap      — generate ABAP Unit test code
//   - refactor-abap  — refactoring suggestions with examples
//   - explain-abap   — explain ABAP code in plain language
//
// Authored by Phani Puttabakula. Open-sourced under Apache 2.0 by BlueFunda, Inc.
package main
