# Contributing to ABAPER MCP

Thank you for your interest in contributing to the ABAPER MCP server! This document provides guidelines and instructions for contributing.

## Development Setup

### Prerequisites

- Go 1.21 or higher
- Git
- Access to an SAP system with ADT enabled (for testing)

### Getting Started

1. **Fork and Clone**
   ```bash
   git fork https://github.com/bluefunda/abaper-mcp
   cd abaper-mcp
   ```

2. **Install Dependencies**
   ```bash
   make install
   ```

3. **Build the Project**
   ```bash
   make build
   ```

4. **Run Tests**
   ```bash
   make test
   ```

## Project Structure

```
abaper-mcp/
├── main.go           # Entry point and MCP server setup
├── config.go         # Configuration and ADT client management
├── handlers.go       # Handler coordination
├── tools.go          # MCP tools implementation
├── resources.go      # MCP resources implementation
├── prompts.go        # MCP prompts implementation
├── go.mod            # Go module dependencies
├── Makefile          # Build automation
└── README.md         # Project documentation
```

## Code Style

- Follow standard Go conventions
- Use `gofmt` to format code: `make fmt`
- Run linter before submitting: `make lint`
- Write clear, descriptive comments
- Keep functions focused and concise

## Adding New Features

### Adding a New Tool

1. Edit `tools.go`
2. Define input and output types with JSON schema tags
3. Implement the handler function
4. Register the tool in `registerTools()`

Example:

```go
// 1. Define types
type MyToolInput struct {
    Name string `json:"name" jsonschema:"required,description=Object name"`
}

type MyToolOutput struct {
    Result string `json:"result" jsonschema:"description=Operation result"`
}

// 2. Implement handler
func (h *Handlers) HandleMyTool(ctx context.Context, req *mcp.CallToolRequest,
    input MyToolInput) (*mcp.CallToolResult, MyToolOutput, error) {

    client, err := h.clientManager.GetClient()
    if err != nil {
        return &mcp.CallToolResult{IsError: true}, MyToolOutput{}, err
    }

    // Implementation here

    return nil, MyToolOutput{Result: "success"}, nil
}

// 3. Register in registerTools()
mcp.AddTool(server, &mcp.Tool{
    Name: "my-tool",
    Description: "Description of what the tool does",
}, handlers.HandleMyTool)
```

### Adding a New Resource

1. Edit `resources.go`
2. Register the resource template
3. Implement the handler function

Example:

```go
// 1. Register resource
server.AddResourceTemplate(&mcp.ResourceTemplate{
    URITemplate: "abap://mytype/{name}",
    Name: "ABAP MyType",
    Description: "Access ABAP MyType objects",
    MIMEType: "text/x-abap",
}, handlers.HandleMyTypeResource)

// 2. Implement handler
func (h *Handlers) HandleMyTypeResource(ctx context.Context,
    req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {

    name := extractNameFromURI(req.Params.URI, "abap://mytype/")
    if name == "" {
        return nil, mcp.ResourceNotFoundError(req.Params.URI)
    }

    // Implementation here

    return &mcp.ReadResourceResult{
        Contents: []*mcp.ResourceContents{
            {
                URI: req.Params.URI,
                MIMEType: "text/x-abap",
                Text: content,
            },
        },
    }, nil
}
```

### Adding a New Prompt

1. Edit `prompts.go`
2. Register the prompt with arguments
3. Implement the handler function

Example:

```go
// 1. Register prompt
server.AddPrompt(&mcp.Prompt{
    Name: "my-prompt",
    Description: "Description of the workflow",
    Arguments: []*mcp.PromptArgument{
        {
            Name: "object_name",
            Description: "Name of the object",
            Required: true,
        },
    },
}, handlers.HandleMyPrompt)

// 2. Implement handler
func (h *Handlers) HandleMyPrompt(ctx context.Context,
    req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {

    objectName := req.Params.Arguments["object_name"]

    // Build prompt text
    prompt := fmt.Sprintf("Prompt text with %s", objectName)

    return &mcp.GetPromptResult{
        Messages: []*mcp.PromptMessage{
            {
                Role: "user",
                Content: &mcp.TextContent{Text: prompt},
            },
        },
    }, nil
}
```

## Testing

### Manual Testing

1. Build the server: `make build`
2. Configure Claude Desktop (see CLAUDE_DESKTOP_SETUP.md)
3. Test the new feature through Claude Desktop
4. Verify error handling and edge cases

### Automated Testing

Add unit tests for new features:

```go
func TestMyTool(t *testing.T) {
    // Test implementation
}
```

Run tests:
```bash
make test
```

## Commit Guidelines

### Commit Messages

Follow the conventional commits format:

```
type(scope): subject

body

footer
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**
```
feat(tools): add support for table data retrieval

Add new tool to retrieve table contents from SAP system.
Includes pagination support and field selection.

Closes #123

fix(resources): handle missing objects gracefully

Return proper ResourceNotFoundError instead of generic error
when ABAP object doesn't exist.

docs(readme): update installation instructions

Add section on macOS-specific setup requirements.
```

## Pull Request Process

1. **Create a Feature Branch**
   ```bash
   git checkout -b feature/my-new-feature
   ```

2. **Make Your Changes**
   - Write code
   - Add tests
   - Update documentation
   - Run `make fmt` and `make lint`

3. **Commit Your Changes**
   ```bash
   git add .
   git commit -m "feat(scope): description"
   ```

4. **Push to Your Fork**
   ```bash
   git push origin feature/my-new-feature
   ```

5. **Create Pull Request**
   - Go to GitHub and create a pull request
   - Fill in the pull request template
   - Link any related issues
   - Wait for review

### Pull Request Checklist

- [ ] Code follows project style guidelines
- [ ] Tests added for new features
- [ ] Documentation updated
- [ ] Commit messages follow guidelines
- [ ] No breaking changes (or clearly documented)
- [ ] All tests pass
- [ ] Code has been formatted with `gofmt`

## Code Review

- Be respectful and constructive
- Focus on code quality and maintainability
- Suggest improvements, don't demand changes
- Explain the "why" behind feedback
- Be open to discussion

## Documentation

When adding features, update:

- `README.md` - If changing user-facing functionality
- `CLAUDE_DESKTOP_SETUP.md` - If affecting setup process
- Code comments - For complex logic
- Examples - For new usage patterns

## Questions or Issues?

- Open an issue on GitHub
- Tag maintainers for urgent matters
- Join discussions in existing issues/PRs
- Check existing documentation first

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

## Recognition

Contributors will be recognized in:
- GitHub contributors page
- Release notes for significant contributions
- Project README (for major features)

Thank you for contributing to ABAPER MCP! 🎉
