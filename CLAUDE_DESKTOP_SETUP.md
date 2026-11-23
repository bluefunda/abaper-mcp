# Claude Desktop Setup Guide

This guide explains how to configure the ABAPER MCP server with Claude Desktop.

Claude Desktop runs abaper-mcp in **stdio mode**: a local process for you alone that never listens on the
network. abaper-mcp itself holds no SAP credentials and never reads `SAP_*` variables. It talks to an abaper
backend that you run on your own machine with **your own** SAP credentials. (The hosted server mode is
different: there every user's credentials are looked up per user, and no shared SAP login is allowed.)

## Prerequisites

1. Build the ABAPER MCP server:
   ```bash
   cd /path/to/abaper-mcp
   make build
   ```

2. Note the absolute path to the `abaper-mcp` binary:
   ```bash
   pwd
   # Output: /Users/phani/Downloads/src/abaper-mcp
   ```

3. Install the `abaper` CLI and add your own SAP system (stored in `~/.abaper/systems.json`, mode 0600):
   ```bash
   abaper system add        # host incl. protocol and port, client, your username and password
   ```

4. Start a local backend with your own SAP identity (listens on `127.0.0.1:8080` only):
   ```bash
   abaper serve --allow-static-sap
   ```

## Configuration Steps

### 1. Locate Claude Desktop Config File

The configuration file location depends on your operating system:

**macOS:**
```
~/Library/Application Support/Claude/claude_desktop_config.json
```

**Windows:**
```
%APPDATA%\Claude\claude_desktop_config.json
```

**Linux:**
```
~/.config/Claude/claude_desktop_config.json
```

### 2. Edit Configuration File

Open the configuration file in your preferred text editor. If the file doesn't exist, create it.

Add the ABAPER MCP server configuration:

```json
{
  "mcpServers": {
    "abaper": {
      "command": "/absolute/path/to/abaper-mcp",
      "env": {
        "ABAPER_BACKEND_URL": "http://localhost:8080"
      }
    }
  }
}
```

**Important:**
- Replace `/absolute/path/to/abaper-mcp` with the actual absolute path to your binary
- `ABAPER_BACKEND_URL` points at the `abaper serve --allow-static-sap` you started; your SAP credentials stay
  in `~/.abaper/systems.json`, not in this file
- The `command` path must be absolute, not relative

### Example Configuration

```json
{
  "mcpServers": {
    "abaper": {
      "command": "/Users/phani/Downloads/src/abaper-mcp/abaper-mcp",
      "env": {
        "ABAPER_BACKEND_URL": "http://localhost:8080"
      }
    }
  }
}
```

### 3. Restart Claude Desktop

After saving the configuration file:

1. Completely quit Claude Desktop
2. Restart Claude Desktop
3. The ABAPER MCP server should now be available

## Verification

Once Claude Desktop restarts, you should see the ABAPER server available. You can verify by asking Claude:

```
"Can you list the available MCP servers?"
```

Or test directly:

```
"Use the get-object tool to retrieve the source code for class CL_ABAP_COMPILER"
```

## Available Features

### Tools

- **get-object**: Retrieve ABAP object source code
  ```
  "Get the source code for program SAPMS38M"
  "Retrieve class ZCL_MY_CLASS"
  ```

- **search-objects**: Search for ABAP objects
  ```
  "Search for all objects starting with Z_SALES"
  "Find all classes containing 'customer' in their name"
  ```

- **list-packages**: List ABAP packages
  ```
  "List all packages in the system"
  ```

- **test-connection**: Test SAP connection
  ```
  "Test the connection to the SAP system"
  ```

- **create-program**: Create new ABAP program
  ```
  "Create a new program called ZHELLO_WORLD with a simple output"
  ```

- **create-class**: Create new ABAP class
  ```
  "Create a new class ZCL_CALCULATOR with basic math methods"
  ```

### Resources

Access ABAP objects using URI schemes:

```
"Show me the code in abap://class/ZCL_CUSTOMER"
"Read abap://program/ZTEST_REPORT"
"Display abap://packages"
```

Supported URI schemes:
- `abap://program/{name}`
- `abap://class/{name}`
- `abap://interface/{name}`
- `abap://table/{name}`
- `abap://structure/{name}`
- `abap://include/{name}`
- `abap://packages`

### Prompts

Use pre-configured workflows:

```
"Use the analyze-abap prompt for class ZCL_SALES_ORDER"
"Run the optimize-abap prompt on program ZSALES_REPORT"
"Generate tests using the test-abap prompt for class ZCL_VALIDATOR"
```

Available prompts:
- `analyze-abap` - Code quality and performance analysis
- `review-abap` - Comprehensive code review
- `optimize-abap` - Performance optimization suggestions
- `document-abap` - Generate documentation
- `test-abap` - Generate unit tests
- `refactor-abap` - Refactoring suggestions
- `explain-abap` - Explain code in simple terms

## Troubleshooting

### Server Not Showing Up

1. **Check file path**: Ensure the path to `abaper-mcp` is absolute and correct
2. **Check permissions**: Make sure the binary is executable:
   ```bash
   chmod +x /path/to/abaper-mcp
   ```
3. **Check JSON syntax**: Validate your `claude_desktop_config.json` is valid JSON
4. **Check logs**: Look at Claude Desktop logs for error messages

### Connection Issues

1. **Check the local backend**: `abaper serve --allow-static-sap` must be running (`curl http://localhost:8080/health`)
2. **Verify SAP credentials**: Test your own credentials using SAP GUI first
3. **Check network**: Ensure you can reach the SAP host
4. **Check ADT enabled**: Verify ADT services are enabled on your SAP system
5. **Check client**: Ensure the SAP client number is correct

### Object Not Found

1. **Check object name**: Object names are case-sensitive
2. **Check object type**: Ensure you're using the correct type (program/class/etc.)
3. **Check permissions**: Verify you have read access to the object

## Security Considerations

### Credential Storage

Your SAP password is stored by the `abaper` CLI in `~/.abaper/systems.json` (mode 0600), not in the Claude
Desktop config. Use only your own SAP credentials, and keep `abaper serve --allow-static-sap` on your own
machine: it serves anyone who can reach it with your identity, which is why it listens on `127.0.0.1` only.
Never run it, or abaper-mcp with a shared SAP login, as a server for other people.

### Network Security

- Ensure SAP host uses HTTPS (not HTTP)
- Use VPN when accessing SAP systems remotely
- Follow your organization's security policies

## Multiple SAP Systems

Run one local backend per system on different ports, each with one of your own stored systems:

```bash
abaper serve --allow-static-sap --system dev --port 8080
abaper serve --allow-static-sap --system qa  --port 8081
```

```json
{
  "mcpServers": {
    "abaper-dev": {
      "command": "/path/to/abaper-mcp",
      "env": { "ABAPER_BACKEND_URL": "http://localhost:8080" }
    },
    "abaper-qa": {
      "command": "/path/to/abaper-mcp",
      "env": { "ABAPER_BACKEND_URL": "http://localhost:8081" }
    }
  }
}
```

Then specify which server to use:

```
"Use abaper-dev to get class ZCL_TEST"
"Use abaper-qa to search for Z_QA_*"
```

## Support

For issues:
- GitHub Issues: https://github.com/bluefunda/abaper-mcp/issues
- Check logs in Claude Desktop for error messages
- Verify SAP ADT connectivity separately

## Example Workflows

### Code Analysis Workflow

```
1. "Test the SAP connection"
2. "Search for all classes starting with ZCL_SALES"
3. "Get the source code for class ZCL_SALES_ORDER"
4. "Use the analyze-abap prompt to analyze ZCL_SALES_ORDER"
5. "Use the optimize-abap prompt for performance suggestions"
```

### Development Workflow

```
1. "Create a new class ZCL_CALCULATOR in package $TMP"
2. "Show me abap://class/ZCL_CALCULATOR"
3. "Use the test-abap prompt to generate unit tests"
4. "Use the document-abap prompt to create documentation"
```

### Maintenance Workflow

```
1. "Search for all programs containing 'REPORT'"
2. "Get program ZMONTHLY_REPORT"
3. "Use the review-abap prompt to review the code"
4. "Use the refactor-abap prompt for improvement suggestions"
```
