# Claude Desktop Setup Guide

This guide explains how to configure the ABAPER MCP server with Claude Desktop.

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

3. Have your SAP system credentials ready:
   - SAP Host (including protocol and port, e.g., `https://sapdev.company.com:8000`)
   - SAP Client (typically `100`)
   - SAP Username
   - SAP Password

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
        "SAP_HOST": "https://your-sap-host:8000",
        "SAP_CLIENT": "100",
        "SAP_USERNAME": "your-username",
        "SAP_PASSWORD": "your-password"
      }
    }
  }
}
```

**Important:**
- Replace `/absolute/path/to/abaper-mcp` with the actual absolute path to your binary
- Replace the SAP credentials with your actual values
- The `command` path must be absolute, not relative

### Example Configuration

```json
{
  "mcpServers": {
    "abaper": {
      "command": "/Users/phani/Downloads/src/abaper-mcp/abaper-mcp",
      "env": {
        "SAP_HOST": "https://sapdev.mycompany.com:8000",
        "SAP_CLIENT": "100",
        "SAP_USERNAME": "DEVELOPER01",
        "SAP_PASSWORD": "MySecurePassword123"
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

1. **Verify SAP credentials**: Test credentials using SAP GUI first
2. **Check network**: Ensure you can reach the SAP host
3. **Check ADT enabled**: Verify ADT services are enabled on your SAP system
4. **Check client**: Ensure the SAP client number is correct

### Object Not Found

1. **Check object name**: Object names are case-sensitive
2. **Check object type**: Ensure you're using the correct type (program/class/etc.)
3. **Check permissions**: Verify you have read access to the object

## Security Considerations

### Password Storage

The configuration file stores your SAP password in plain text. To secure it:

1. **File Permissions**: Restrict access to the config file
   ```bash
   chmod 600 ~/Library/Application\ Support/Claude/claude_desktop_config.json
   ```

2. **Alternative**: Use environment variables instead
   - Set environment variables in your shell profile
   - Omit the `env` section from the config file
   - The server will read from system environment variables

### Network Security

- Ensure SAP host uses HTTPS (not HTTP)
- Use VPN when accessing SAP systems remotely
- Follow your organization's security policies

## Multiple SAP Systems

You can configure multiple SAP systems:

```json
{
  "mcpServers": {
    "abaper-dev": {
      "command": "/path/to/abaper-mcp",
      "env": {
        "SAP_HOST": "https://sapdev.company.com:8000",
        "SAP_CLIENT": "100",
        "SAP_USERNAME": "devuser",
        "SAP_PASSWORD": "devpass"
      }
    },
    "abaper-qa": {
      "command": "/path/to/abaper-mcp",
      "env": {
        "SAP_HOST": "https://sapqa.company.com:8000",
        "SAP_CLIENT": "200",
        "SAP_USERNAME": "qauser",
        "SAP_PASSWORD": "qapass"
      }
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
