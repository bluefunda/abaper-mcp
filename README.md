# ABAPER MCP Server

[![Go Reference](https://pkg.go.dev/badge/github.com/bluefunda/abaper-mcp.svg)](https://pkg.go.dev/github.com/bluefunda/abaper-mcp)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A Model Context Protocol (MCP) server for SAP ABAP development, built with the official Go SDK. This server enables AI assistants like Claude to interact with SAP ABAP systems through the ABAP Development Tools (ADT) REST API.

## Overview

ABAPER MCP brings the power of AI-assisted development to SAP ABAP by providing:

- **MCP Tools**: Execute ABAP operations (get, search, create objects)
- **MCP Resources**: Access ABAP objects via URI schemes (`abap://class/ZCL_TEST`)
- **MCP Prompts**: Pre-configured workflows for code analysis, review, optimization, and more

This server leverages the [abaper](https://github.com/bluefunda/abaper) library to communicate with SAP systems via ADT and implements the [Model Context Protocol](https://modelcontextprotocol.io) for seamless integration with AI assistants.

## Features

### Tools

Execute ABAP operations programmatically:

- **get-object**: Retrieve source code for any ABAP object (program, class, function, interface, table, structure)
- **search-objects**: Search for ABAP objects by pattern with wildcard support
- **list-packages**: List all ABAP packages in the system
- **test-connection**: Test connectivity to the SAP ADT system
- **create-program**: Create a new ABAP program with source code
- **create-class**: Create a new ABAP class with source code

### Resources

Access ABAP objects using URI schemes:

- `abap://program/{name}` - ABAP programs
- `abap://class/{name}` - ABAP classes
- `abap://function/{name}` - Function modules
- `abap://interface/{name}` - Interfaces
- `abap://table/{name}` - Database tables
- `abap://structure/{name}` - Data structures
- `abap://include/{name}` - Include programs
- `abap://packages` - List all packages

### Prompts

Pre-configured workflows for ABAP development:

- **analyze-abap**: Comprehensive code quality, performance, and security analysis
- **review-abap**: Detailed code review with best practices
- **optimize-abap**: Performance optimization suggestions
- **document-abap**: Generate comprehensive documentation
- **test-abap**: Generate ABAP unit test code
- **refactor-abap**: Refactoring suggestions with examples
- **explain-abap**: Explain code in simple terms

## Installation

### Prerequisites

- Go 1.23 or higher
- Access to an SAP system with ADT enabled
- SAP credentials with appropriate permissions

### go install

```bash
go install github.com/bluefunda/abaper-mcp@latest
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/bluefunda/abaper-mcp.git
cd abaper-mcp

# Install dependencies
make install

# Build the server
make build
```

This creates an `abaper-mcp` binary in the current directory.

## Operational Modes

ABAPER MCP supports four deployment modes:

1. **stdio** (default) - For Claude Desktop integration via standard input/output
2. **sse** - For orchestrator integration via HTTP/SSE (Server-Sent Events)
3. **nats** - For orchestrator integration via NATS messaging (legacy)
4. **dual** - Both stdio and NATS simultaneously (legacy)

### Mode Selection

Set the `ABAPER_MODE` environment variable to choose your mode:

- **stdio**: Direct integration with Claude Desktop or Claude Code CLI
- **sse**: HTTP server for orchestrator integration (recommended for production)
- **nats**: NATS pub/sub messaging (legacy, for custom orchestrator implementations)
- **dual**: Both stdio and NATS (legacy)

See [NATS_INTEGRATION.md](NATS_INTEGRATION.md) for detailed NATS configuration.

## Configuration

### Environment Variables

Create a `.env` file or set the following environment variables:

```bash
# SAP system connection
SAP_HOST=https://your-sap-host:8000
SAP_CLIENT=100
SAP_USERNAME=your-username
SAP_PASSWORD=your-password

# Operational mode (default: stdio)
ABAPER_MODE=stdio              # stdio, sse, nats, or dual

# SSE/HTTP mode configuration (only needed for ABAPER_MODE=sse)
ABAPER_HTTP_PORT=8015          # Default: 8015
ABAPER_HTTP_HOST=0.0.0.0       # Default: 0.0.0.0

# NATS integration (only needed for ABAPER_MODE=nats or dual)
NATS_URL=tls://connect.ngs.global:4222
NATS_CREDS=/path/to/user.creds
NATS_ENABLE_KV=false
NATS_ENABLE_MESSAGING=false
```

Copy `.env.template` to `.env` and fill in your SAP system details:

```bash
cp .env.template .env
```

## Usage

### With Claude Desktop

Add the server to your Claude Desktop configuration:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "abaper": {
      "command": "/absolute/path/to/abaper-mcp",
      "env": {
        "SAP_HOST": "https://your-sap-host:8000",
        "SAP_CLIENT": "100",
        "SAP_USERNAME": "your-username",
        "SAP_PASSWORD": "your-password"
      }
    }
  }
}
```

After configuring, restart Claude Desktop. The ABAPER MCP server will be available for use.

### With Orchestrator (SSE/HTTP Mode)

For integration with orchestrators like cai-mcp-client, run the server in SSE mode:

```bash
# Set environment variables
export ABAPER_MODE=sse
export ABAPER_HTTP_PORT=8015
export SAP_HOST=https://your-sap-host:8000
export SAP_CLIENT=100
export SAP_USERNAME=your-username
export SAP_PASSWORD=your-password

# Run the server
./abaper-mcp
```

The server will start an HTTP server on port 8015:
- **MCP endpoint**: `http://localhost:8015/`
- **Health check**: `http://localhost:8015/health`

#### Docker Deployment

```bash
docker run -d \
  -e ABAPER_MODE=sse \
  -e ABAPER_HTTP_PORT=8015 \
  -e SAP_HOST=https://your-sap-host:8000 \
  -e SAP_CLIENT=100 \
  -e SAP_USERNAME=your-username \
  -e SAP_PASSWORD=your-password \
  -p 8015:8015 \
  --name abaper-mcp \
  --network your-network \
  bluefunda/abaper-mcp:latest
```

#### Orchestrator Configuration

Configure your orchestrator (e.g., cai-mcp-client) to connect to the SSE endpoint:

```python
# Example orchestrator configuration
mcp_servers = {
    "abaper": {
        "mcp_server_url": "http://abaper-mcp:8015/"
    }
}
```

### Standalone Testing

Run the server directly for testing:

```bash
# stdio mode (default)
./abaper-mcp

# SSE/HTTP mode
ABAPER_MODE=sse ABAPER_HTTP_PORT=8015 ./abaper-mcp
```

The server communicates via stdio (default) or HTTP/SSE depending on the mode.

## Examples

### Using Tools

Once connected to Claude Desktop, you can ask Claude to use the ABAPER tools:

```
"Use the get-object tool to retrieve the source code for class ZCL_CUSTOMER"

"Search for all ABAP objects starting with Z_TEST"

"Create a new ABAP program called ZHELLO_WORLD with a simple Hello World output"
```

### Using Resources

Reference ABAP objects directly in your conversations:

```
"Show me the code in abap://class/ZCL_SALES_ORDER"

"Compare abap://program/ZSALES_REPORT with abap://program/ZPURCHASE_REPORT"
```

### Using Prompts

Trigger pre-configured workflows:

```
"Use the analyze-abap prompt for class ZCL_PAYMENT_PROCESSOR"

"Run the optimize-abap prompt on function module Z_CALCULATE_PRICE"

"Generate tests using the test-abap prompt for class ZCL_VALIDATOR"
```

## Architecture

### Component Overview

```
┌─────────────────────────────────────────────┐
│           Claude Desktop / AI Client         │
└─────────────────┬───────────────────────────┘
                  │ MCP Protocol (stdio)
                  │
┌─────────────────▼───────────────────────────┐
│           ABAPER MCP Server (Go)             │
│  ┌─────────────────────────────────────┐    │
│  │  Tools | Resources | Prompts         │    │
│  └─────────────────┬───────────────────┘    │
│                    │                         │
│  ┌─────────────────▼───────────────────┐    │
│  │     ABAPER Library Integration       │    │
│  │  (github.com/bluefunda/abaper)       │    │
│  └─────────────────┬───────────────────┘    │
└────────────────────┼───────────────────────-┘
                     │ ADT REST API
                     │
┌────────────────────▼───────────────────────┐
│          SAP ABAP System (ADT)              │
└─────────────────────────────────────────────┘
```

### Key Components

- **main.go**: Server initialization and MCP setup
- **config.go**: Configuration management and ADT client caching
- **tools.go**: MCP tool implementations for ABAP operations
- **resources.go**: MCP resource handlers for URI-based access
- **prompts.go**: Pre-configured workflow prompts
- **handlers.go**: Request handler coordination

### Design Patterns

- **Client Caching**: ADT client connections are cached for performance (30-min TTL)
- **Lazy Initialization**: Client created only when first needed
- **Thread-Safe**: Concurrent access handled with sync.RWMutex
- **Error Handling**: Comprehensive error handling with context propagation

## Development

### Project Structure

```
abaper-mcp/
├── main.go              # Entry point and server setup
├── config.go            # Configuration and client management
├── handlers.go          # Handler coordination
├── tools.go             # MCP tools implementation
├── resources.go         # MCP resources implementation
├── prompts.go           # MCP prompts implementation
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── Makefile             # Build automation
├── .env.template        # Environment template
├── .gitignore           # Git ignore rules
└── README.md            # This file
```

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Format code
make fmt
```

### Adding New Features

#### Adding a Tool

Edit `tools.go`:

```go
// 1. Define input/output types
type MyToolInput struct {
    Param string `json:"param" jsonschema:"required,description=Parameter description"`
}

type MyToolOutput struct {
    Result string `json:"result" jsonschema:"description=Result description"`
}

// 2. Implement handler
func (h *Handlers) HandleMyTool(ctx context.Context, req *mcp.CallToolRequest,
    input MyToolInput) (*mcp.CallToolResult, MyToolOutput, error) {
    // Implementation
    return nil, MyToolOutput{Result: "success"}, nil
}

// 3. Register in registerTools()
mcp.AddTool(server, &mcp.Tool{
    Name: "my-tool",
    Description: "Tool description",
}, handlers.HandleMyTool)
```

#### Adding a Resource

Edit `resources.go`:

```go
// 1. Register resource template
server.AddResourceTemplate(&mcp.ResourceTemplate{
    URITemplate: "abap://mytype/{name}",
    Name: "ABAP MyType",
    Description: "Description",
    MIMEType: "text/x-abap",
}, handlers.HandleMyResource)

// 2. Implement handler
func (h *Handlers) HandleMyResource(ctx context.Context,
    req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
    // Implementation
}
```

#### Adding a Prompt

Edit `prompts.go`:

```go
// 1. Register prompt
server.AddPrompt(&mcp.Prompt{
    Name: "my-prompt",
    Description: "Prompt description",
    Arguments: []mcp.PromptArgument{
        {Name: "arg", Description: "Argument description", Required: true},
    },
}, handlers.HandleMyPrompt)

// 2. Implement handler
func (h *Handlers) HandleMyPrompt(ctx context.Context,
    req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
    // Build and return prompt
}
```

## Troubleshooting

### Connection Issues

**Problem**: "Failed to get ADT client" error

**Solution**: Verify your environment variables:
- Check `SAP_HOST` includes protocol and port (e.g., `https://host:8000`)
- Verify credentials are correct
- Ensure ADT services are enabled on your SAP system
- Test with: `make run` and check logs

### Authentication Failures

**Problem**: "Authentication failed" or 401 errors

**Solution**:
- Verify username and password are correct
- Check if your account has ADT access permissions
- Ensure the SAP client number is correct
- Try connecting via SAP GUI first to verify credentials

### Object Not Found

**Problem**: "Object not found" when retrieving code

**Solution**:
- Verify object name is correct (exact case)
- Ensure object type matches (program/class/function)
- Check if object exists in the specified client
- Verify you have read permissions for the object

### MCP Connection Issues

**Problem**: Claude Desktop doesn't show ABAPER server

**Solution**:
- Verify the path in `claude_desktop_config.json` is absolute
- Check the binary has execute permissions: `chmod +x abaper-mcp`
- Restart Claude Desktop completely
- Check Claude Desktop logs for error messages

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Documentation

- [NATS Integration Guide](NATS_INTEGRATION.md) - NATS messaging and orchestrator setup
- [Deployment Guide](DEPLOYMENT.md) - Docker, GitHub Actions, and GoReleaser
- [Release Summary](RELEASE_SUMMARY.md) - Release infrastructure overview

## Related Projects

- [abaper](https://github.com/bluefunda/abaper) - Core ABAP ADT client library
- [abaperx](https://github.com/bluefunda/abaperx) - ABAP Enterprise Edition with AI features
- [cai-mcp-client](https://github.com/bluefunda/cai-mcp-client) - MCP orchestrator for multi-server architectures
- [cai-xml-odata](https://github.com/bluefunda/cai-xml-odata) - XML/OData MCP server with NATS support
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) - Official Model Context Protocol SDK

## Acknowledgments

- Built with the official [Model Context Protocol Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- Powered by [abaper](https://github.com/bluefunda/abaper) for SAP ADT integration
- Inspired by [abaperx](https://github.com/bluefunda/abaperx) enterprise features

## Support

For issues and questions:

- GitHub Issues: [https://github.com/bluefunda/abaper-mcp/issues](https://github.com/bluefunda/abaper-mcp/issues)
- abaper Library: [https://github.com/bluefunda/abaper](https://github.com/bluefunda/abaper)

## Version

Current version: 1.0.0
