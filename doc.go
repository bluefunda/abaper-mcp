// Copyright 2025 bluefunda
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
// It exposes ABAP operations — source retrieval, editing, activation, syntax
// checking, unit testing, package browsing, and S/4HANA compatibility analysis
// — as MCP tools consumable by Claude Desktop, Claude Code, and any MCP client.
//
// abaper-mcp delegates all SAP ADT calls to an abaper-ts REST backend; it does
// not connect to SAP directly.
//
// # Installation
//
//	go install github.com/bluefunda/abaper-mcp@latest
//
// # Configuration
//
// All configuration is via environment variables:
//
//	ABAPER_TS_URL   URL of the abaper-ts backend (default: http://localhost:8080)
//	ABAPER_MODE     Transport mode: stdio (default) or sse
//	ABAPER_HTTP_PORT   HTTP port for SSE mode (default: 8015)
//	ABAPER_HTTP_HOST   HTTP host for SSE mode (default: 0.0.0.0)
//	S4_TEMPORAL_URL URL of the s4-temporal API (optional)
//	LOG_LEVEL       Log level: debug, info, warn, error (default: info)
//	LOG_FORMAT      Log format: json (default) or console
//
// # Modes
//
// stdio (default) — for use with Claude Desktop and IDE extensions:
//
//	abaper-mcp
//
// sse — HTTP/SSE server for remote or container deployments:
//
//	ABAPER_MODE=sse abaper-mcp
package main
