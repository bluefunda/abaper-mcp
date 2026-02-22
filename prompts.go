package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerPrompts registers all MCP prompts
func registerPrompts(server *mcp.Server, handlers *Handlers) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "analyze-abap",
		Description: "Analyze ABAP code for quality, performance, and best practices",
		Arguments: []*mcp.PromptArgument{
			{Name: "object_type", Description: "Type of ABAP object (program/class/function/interface)", Required: true},
			{Name: "object_name", Description: "Name of the ABAP object to analyze", Required: true},
		},
	}, handlers.HandleAnalyzePrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "review-abap",
		Description: "Perform a comprehensive code review of ABAP code",
		Arguments: []*mcp.PromptArgument{
			{Name: "object_type", Description: "Type of ABAP object (program/class/function/interface)", Required: true},
			{Name: "object_name", Description: "Name of the ABAP object to review", Required: true},
		},
	}, handlers.HandleReviewPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "optimize-abap",
		Description: "Suggest optimizations for ABAP code performance and efficiency",
		Arguments: []*mcp.PromptArgument{
			{Name: "object_type", Description: "Type of ABAP object (program/class/function/interface)", Required: true},
			{Name: "object_name", Description: "Name of the ABAP object to optimize", Required: true},
		},
	}, handlers.HandleOptimizePrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "document-abap",
		Description: "Generate comprehensive documentation for ABAP code",
		Arguments: []*mcp.PromptArgument{
			{Name: "object_type", Description: "Type of ABAP object (program/class/function/interface)", Required: true},
			{Name: "object_name", Description: "Name of the ABAP object to document", Required: true},
		},
	}, handlers.HandleDocumentPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "test-abap",
		Description: "Generate ABAP unit test code for a given object",
		Arguments: []*mcp.PromptArgument{
			{Name: "object_type", Description: "Type of ABAP object (program/class/function/interface)", Required: true},
			{Name: "object_name", Description: "Name of the ABAP object to test", Required: true},
		},
	}, handlers.HandleTestPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "refactor-abap",
		Description: "Suggest refactoring improvements for ABAP code",
		Arguments: []*mcp.PromptArgument{
			{Name: "object_type", Description: "Type of ABAP object (program/class/function/interface)", Required: true},
			{Name: "object_name", Description: "Name of the ABAP object to refactor", Required: true},
		},
	}, handlers.HandleRefactorPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "explain-abap",
		Description: "Explain what the ABAP code does in simple terms",
		Arguments: []*mcp.PromptArgument{
			{Name: "object_type", Description: "Type of ABAP object (program/class/function/interface)", Required: true},
			{Name: "object_name", Description: "Name of the ABAP object to explain", Required: true},
		},
	}, handlers.HandleExplainPrompt)
}

// HandleAnalyzePrompt generates a prompt for analyzing ABAP code
func (h *Handlers) HandleAnalyzePrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	objectType := req.Params.Arguments["object_type"]
	objectName := req.Params.Arguments["object_name"]

	sourceCode, err := h.getSourceCode(objectType, objectName)
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(`Please analyze the following ABAP %s code for:

1. **Code Quality**: Overall code structure, readability, and maintainability
2. **Performance**: Potential performance issues and bottlenecks
3. **Best Practices**: Adherence to ABAP best practices and standards
4. **Security**: Potential security vulnerabilities
5. **Error Handling**: Quality of error handling and exception management

**ABAP %s: %s**

%s

Please provide a detailed analysis with specific recommendations for improvement.`,
		strings.ToUpper(objectType),
		strings.ToUpper(objectType),
		objectName,
		wrapCodeBlock(sourceCode))

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{Role: "user", Content: &mcp.TextContent{Text: prompt}},
		},
	}, nil
}

// HandleReviewPrompt generates a prompt for reviewing ABAP code
func (h *Handlers) HandleReviewPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	objectType := req.Params.Arguments["object_type"]
	objectName := req.Params.Arguments["object_name"]

	sourceCode, err := h.getSourceCode(objectType, objectName)
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(`Please perform a comprehensive code review of the following ABAP %s:

**Focus Areas:**
- Design patterns and architecture
- Code organization and modularity
- Naming conventions
- Code duplication
- Complexity and maintainability
- Documentation and comments
- Testing considerations

**ABAP %s: %s**

%s

Provide constructive feedback with specific examples and suggestions for improvement.`,
		strings.ToUpper(objectType),
		strings.ToUpper(objectType),
		objectName,
		wrapCodeBlock(sourceCode))

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{Role: "user", Content: &mcp.TextContent{Text: prompt}},
		},
	}, nil
}

// HandleOptimizePrompt generates a prompt for optimizing ABAP code
func (h *Handlers) HandleOptimizePrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	objectType := req.Params.Arguments["object_type"]
	objectName := req.Params.Arguments["object_name"]

	sourceCode, err := h.getSourceCode(objectType, objectName)
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(`Please analyze the following ABAP %s and suggest optimizations for:

1. **Performance**: Database queries, loops, and data processing
2. **Memory Usage**: Efficient data structures and memory management
3. **Execution Time**: Algorithm efficiency and unnecessary operations
4. **SAP Best Practices**: Modern ABAP syntax and features
5. **Resource Utilization**: System resources and scalability

**ABAP %s: %s**

%s

Provide specific optimization suggestions with code examples where applicable.`,
		strings.ToUpper(objectType),
		strings.ToUpper(objectType),
		objectName,
		wrapCodeBlock(sourceCode))

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{Role: "user", Content: &mcp.TextContent{Text: prompt}},
		},
	}, nil
}

// HandleDocumentPrompt generates a prompt for documenting ABAP code
func (h *Handlers) HandleDocumentPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	objectType := req.Params.Arguments["object_type"]
	objectName := req.Params.Arguments["object_name"]

	sourceCode, err := h.getSourceCode(objectType, objectName)
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(`Please generate comprehensive documentation for the following ABAP %s:

**Documentation Should Include:**
- Purpose and overview
- Input parameters and their descriptions
- Output/return values
- Main functionality and logic flow
- Dependencies and related objects
- Usage examples
- Error conditions and exceptions
- Any important notes or warnings

**ABAP %s: %s**

%s

Generate clear, professional documentation suitable for technical and business users.`,
		strings.ToUpper(objectType),
		strings.ToUpper(objectType),
		objectName,
		wrapCodeBlock(sourceCode))

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{Role: "user", Content: &mcp.TextContent{Text: prompt}},
		},
	}, nil
}

// HandleTestPrompt generates a prompt for creating unit tests
func (h *Handlers) HandleTestPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	objectType := req.Params.Arguments["object_type"]
	objectName := req.Params.Arguments["object_name"]

	sourceCode, err := h.getSourceCode(objectType, objectName)
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(`Please generate ABAP unit test code for the following %s:

**Test Requirements:**
- Use ABAP Unit Test framework
- Cover main functionality and edge cases
- Include positive and negative test scenarios
- Test error handling and exceptions
- Use meaningful test method names
- Include test data setup and teardown
- Add descriptive comments

**ABAP %s: %s**

%s

Generate complete, runnable ABAP unit test code with good coverage.`,
		strings.ToUpper(objectType),
		strings.ToUpper(objectType),
		objectName,
		wrapCodeBlock(sourceCode))

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{Role: "user", Content: &mcp.TextContent{Text: prompt}},
		},
	}, nil
}

// HandleRefactorPrompt generates a prompt for refactoring suggestions
func (h *Handlers) HandleRefactorPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	objectType := req.Params.Arguments["object_type"]
	objectName := req.Params.Arguments["object_name"]

	sourceCode, err := h.getSourceCode(objectType, objectName)
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(`Please suggest refactoring improvements for the following ABAP %s:

**Refactoring Focus:**
- Extract methods to reduce complexity
- Improve naming and clarity
- Eliminate code duplication
- Simplify conditional logic
- Apply design patterns where appropriate
- Modernize ABAP syntax
- Improve separation of concerns

**ABAP %s: %s**

%s

Provide specific refactoring suggestions with before/after code examples.`,
		strings.ToUpper(objectType),
		strings.ToUpper(objectType),
		objectName,
		wrapCodeBlock(sourceCode))

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{Role: "user", Content: &mcp.TextContent{Text: prompt}},
		},
	}, nil
}

// HandleExplainPrompt generates a prompt for explaining ABAP code
func (h *Handlers) HandleExplainPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	objectType := req.Params.Arguments["object_type"]
	objectName := req.Params.Arguments["object_name"]

	sourceCode, err := h.getSourceCode(objectType, objectName)
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(`Please explain the following ABAP %s in clear, simple terms:

**Explanation Should Cover:**
- What the code does (high-level purpose)
- How it works (main logic flow)
- Key components and their roles
- Important business logic
- Any complex or non-obvious parts
- Potential use cases

**ABAP %s: %s**

%s

Explain the code in a way that both technical and non-technical stakeholders can understand.`,
		strings.ToUpper(objectType),
		strings.ToUpper(objectType),
		objectName,
		wrapCodeBlock(sourceCode))

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{Role: "user", Content: &mcp.TextContent{Text: prompt}},
		},
	}, nil
}

// getSourceCode retrieves source code for a given object via abaper-ts
func (h *Handlers) getSourceCode(objectType, objectName string) (string, error) {
	adtType := normalizeObjectType(objectType)
	result, err := h.apiClient.GetObject(adtType, objectName, "")
	if err != nil {
		return "", fmt.Errorf("failed to get %s %s: %w", objectType, objectName, err)
	}
	return result.Source, nil
}

// wrapCodeBlock wraps code in markdown code block
func wrapCodeBlock(code string) string {
	return fmt.Sprintf("```abap\n%s\n```", code)
}
