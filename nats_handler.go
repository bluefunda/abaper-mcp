package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nats-io/nats.go"
)

// NATSMCPServer handles MCP requests via NATS
type NATSMCPServer struct {
	natsConn *NATSConnection
	handlers *Handlers
	server   *mcp.Server
}

// MCPToolRequest represents an MCP tool request from orchestrator
type MCPToolRequest struct {
	Tool      string                 `json:"tool"`
	Arguments map[string]interface{} `json:"arguments"`
	RequestID string                 `json:"request_id,omitempty"`
}

// MCPToolResponse represents an MCP tool response
type MCPToolResponse struct {
	RequestID string                 `json:"request_id,omitempty"`
	Success   bool                   `json:"success"`
	Result    map[string]interface{} `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

// NewNATSMCPServer creates a new NATS MCP server
func NewNATSMCPServer(natsConn *NATSConnection, handlers *Handlers, server *mcp.Server) *NATSMCPServer {
	return &NATSMCPServer{
		natsConn: natsConn,
		handlers: handlers,
		server:   server,
	}
}

// Start starts the NATS MCP server by subscribing to subjects
func (s *NATSMCPServer) Start() error {
	subject := "mcp.abaper.tools.request"
	_, err := s.natsConn.Subscribe(subject, s.handleToolRequest)
	if err != nil {
		return fmt.Errorf("failed to subscribe to tool requests: %w", err)
	}

	wildcardSubject := "mcp.*.abaper.tools.request"
	_, err = s.natsConn.Subscribe(wildcardSubject, s.handleToolRequest)
	if err != nil {
		return fmt.Errorf("failed to subscribe to wildcard tool requests: %w", err)
	}

	fmt.Println("NATS MCP server started and listening for tool requests")
	return nil
}

// handleToolRequest handles incoming MCP tool requests from NATS
func (s *NATSMCPServer) handleToolRequest(msg *nats.Msg) {
	fmt.Printf("Received NATS request on subject: %s\n", msg.Subject)

	var req MCPToolRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		s.sendErrorResponse(msg, "", fmt.Errorf("failed to parse request: %w", err))
		return
	}

	ctx := context.Background()
	result, err := s.routeToolRequest(ctx, &req)

	if err != nil {
		s.sendErrorResponse(msg, req.RequestID, err)
		return
	}

	s.sendSuccessResponse(msg, req.RequestID, result)
}

// routeToolRequest routes the request to the appropriate tool handler
func (s *NATSMCPServer) routeToolRequest(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	switch req.Tool {
	case "get-object":
		return s.handleGetObject(ctx, req)
	case "search-objects":
		return s.handleSearchObjects(ctx, req)
	case "list-packages":
		return s.handleListPackages(ctx, req)
	case "test-connection":
		return s.handleTestConnection(ctx, req)
	case "create-program":
		return s.handleCreateProgram(ctx, req)
	case "create-class":
		return s.handleCreateClass(ctx, req)
	case "update-program":
		return s.handleUpdateProgram(ctx, req)
	case "update-class":
		return s.handleUpdateClass(ctx, req)
	case "activate-object":
		return s.handleActivateObject(ctx, req)
	case "run-unit-tests":
		return s.handleRunUnitTests(ctx, req)
	case "analyze-s4-remediation":
		return s.handleAnalyzeS4Remediation(ctx, req)
	case "syntax-check":
		return s.handleSyntaxCheck(ctx, req)
	case "format-code":
		return s.handleFormatCode(ctx, req)
	case "transport-info":
		return s.handleTransportInfo(ctx, req)
	case "create-transport":
		return s.handleCreateTransport(ctx, req)
	default:
		return nil, fmt.Errorf("unknown tool: %s", req.Tool)
	}
}

// handleGetObject handles get-object tool request
func (s *NATSMCPServer) handleGetObject(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	objectType, _ := req.Arguments["object_type"].(string)
	objectName, _ := req.Arguments["object_name"].(string)
	functionGroup, _ := req.Arguments["function_group"].(string)

	if objectType == "" || objectName == "" {
		return nil, fmt.Errorf("object_type and object_name are required")
	}

	input := GetObjectInput{ObjectType: objectType, ObjectName: objectName, FunctionGroup: functionGroup}
	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleGetObject(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"object_type": output.ObjectType,
		"object_name": output.ObjectName,
		"source_code": output.SourceCode,
	}
	if output.Description != "" {
		result["description"] = output.Description
	}
	return result, nil
}

// handleSearchObjects handles search-objects tool request
func (s *NATSMCPServer) handleSearchObjects(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	pattern, _ := req.Arguments["pattern"].(string)
	if pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}

	var objectTypes []string
	if types, ok := req.Arguments["object_types"].([]interface{}); ok {
		for _, t := range types {
			if str, ok := t.(string); ok {
				objectTypes = append(objectTypes, str)
			}
		}
	}

	input := SearchObjectsInput{Pattern: pattern, ObjectTypes: objectTypes}
	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleSearchObjects(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}

	objects := make([]map[string]interface{}, len(output.Objects))
	for i, obj := range output.Objects {
		objects[i] = map[string]interface{}{
			"type": obj.Type, "name": obj.Name,
			"description": obj.Description, "package": obj.Package,
		}
	}
	return map[string]interface{}{"objects": objects, "count": output.Count}, nil
}

// handleListPackages handles list-packages tool request
func (s *NATSMCPServer) handleListPackages(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleListPackages(ctx, mcpReq, ListPackagesInput{})
	if err != nil {
		return nil, err
	}

	packages := make([]map[string]interface{}, len(output.Packages))
	for i, pkg := range output.Packages {
		packages[i] = map[string]interface{}{"name": pkg.Name, "description": pkg.Description}
	}
	return map[string]interface{}{"packages": packages, "count": output.Count}, nil
}

// handleTestConnection handles test-connection tool request
func (s *NATSMCPServer) handleTestConnection(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleTestConnection(ctx, mcpReq, TestConnectionInput{})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"connected": output.Connected, "message": output.Message}, nil
}

// handleCreateProgram handles create-program tool request
func (s *NATSMCPServer) handleCreateProgram(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	name, _ := req.Arguments["name"].(string)
	description, _ := req.Arguments["description"].(string)
	pkg, _ := req.Arguments["package"].(string)
	sourceCode, _ := req.Arguments["source_code"].(string)

	if name == "" || description == "" || pkg == "" || sourceCode == "" {
		return nil, fmt.Errorf("name, description, package, and source_code are required")
	}

	input := CreateProgramInput{Name: name, Description: description, Package: pkg, SourceCode: sourceCode}
	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleCreateProgram(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": output.Success, "message": output.Message, "name": output.Name}, nil
}

// handleCreateClass handles create-class tool request
func (s *NATSMCPServer) handleCreateClass(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	name, _ := req.Arguments["name"].(string)
	description, _ := req.Arguments["description"].(string)
	pkg, _ := req.Arguments["package"].(string)
	sourceCode, _ := req.Arguments["source_code"].(string)

	if name == "" || description == "" || pkg == "" || sourceCode == "" {
		return nil, fmt.Errorf("name, description, package, and source_code are required")
	}

	input := CreateClassInput{Name: name, Description: description, Package: pkg, SourceCode: sourceCode}
	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleCreateClass(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": output.Success, "message": output.Message, "name": output.Name}, nil
}

// handleUpdateProgram handles update-program tool request
func (s *NATSMCPServer) handleUpdateProgram(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	name, _ := req.Arguments["name"].(string)
	sourceCode, _ := req.Arguments["source_code"].(string)

	if name == "" || sourceCode == "" {
		return nil, fmt.Errorf("name and source_code are required")
	}

	input := UpdateProgramInput{Name: name, SourceCode: sourceCode}
	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleUpdateProgram(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": output.Success, "message": output.Message, "name": output.Name}, nil
}

// handleUpdateClass handles update-class tool request
func (s *NATSMCPServer) handleUpdateClass(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	name, _ := req.Arguments["name"].(string)
	sourceCode, _ := req.Arguments["source_code"].(string)

	if name == "" || sourceCode == "" {
		return nil, fmt.Errorf("name and source_code are required")
	}

	input := UpdateClassInput{Name: name, SourceCode: sourceCode}
	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleUpdateClass(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": output.Success, "message": output.Message, "name": output.Name}, nil
}

// handleActivateObject handles activate-object tool request
func (s *NATSMCPServer) handleActivateObject(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	objectType, _ := req.Arguments["object_type"].(string)
	objectName, _ := req.Arguments["object_name"].(string)

	if objectType == "" || objectName == "" {
		return nil, fmt.Errorf("object_type and object_name are required")
	}

	input := ActivateObjectInput{ObjectType: objectType, ObjectName: objectName}
	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleActivateObject(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"success": output.Success, "message": output.Message,
		"object_name": output.ObjectName, "object_type": output.ObjectType,
	}, nil
}

// handleRunUnitTests handles run-unit-tests tool request
func (s *NATSMCPServer) handleRunUnitTests(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	objectType, _ := req.Arguments["object_type"].(string)
	objectName, _ := req.Arguments["object_name"].(string)

	if objectType == "" || objectName == "" {
		return nil, fmt.Errorf("object_type and object_name are required")
	}

	input := RunUnitTestsInput{ObjectType: objectType, ObjectName: objectName}
	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleRunUnitTests(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"all_passed": output.AllPassed, "total_tests": output.TotalTests,
		"passed": output.Passed, "failed": output.Failed,
		"details": output.Details, "object_name": output.ObjectName,
	}, nil
}

// handleAnalyzeS4Remediation handles analyze-s4-remediation tool request
func (s *NATSMCPServer) handleAnalyzeS4Remediation(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	objectType, _ := req.Arguments["object_type"].(string)
	objectName, _ := req.Arguments["object_name"].(string)
	functionGroup, _ := req.Arguments["function_group"].(string)
	outputFormat, _ := req.Arguments["output_format"].(string)

	if objectType == "" || objectName == "" {
		return nil, fmt.Errorf("object_type and object_name are required")
	}

	input := AnalyzeS4RemediationInput{
		ObjectType: objectType, ObjectName: objectName,
		FunctionGroup: functionGroup, OutputFormat: outputFormat,
	}
	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleAnalyzeS4Remediation(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}

	issues := make([]map[string]interface{}, len(output.JSON.Issues))
	for i, issue := range output.JSON.Issues {
		issues[i] = map[string]interface{}{
			"pattern_id": issue.PatternID, "pattern_title": issue.PatternTitle,
			"symptom_detected": issue.SymptomDetected, "severity": issue.Severity,
			"before_code": issue.BeforeCode, "after_code": issue.AfterCode,
			"fix_description": issue.FixDescription,
		}
	}

	return map[string]interface{}{
		"json": map[string]interface{}{
			"run_metadata": map[string]interface{}{
				"run_id": output.JSON.RunMetadata.RunID, "timestamp_utc": output.JSON.RunMetadata.TimestampUTC,
				"system_id": output.JSON.RunMetadata.SystemID, "client": output.JSON.RunMetadata.Client,
				"analyst": output.JSON.RunMetadata.Analyst,
			},
			"artifact": map[string]interface{}{
				"artifact_name": output.JSON.Artifact.ArtifactName, "artifact_type": output.JSON.Artifact.ArtifactType,
			},
			"issues": issues,
		},
		"markdown": output.Markdown,
	}, nil
}

// handleSyntaxCheck handles syntax-check tool request
func (s *NATSMCPServer) handleSyntaxCheck(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	objectType, _ := req.Arguments["object_type"].(string)
	objectName, _ := req.Arguments["object_name"].(string)
	sourceCode, _ := req.Arguments["source_code"].(string)

	if objectType == "" || objectName == "" || sourceCode == "" {
		return nil, fmt.Errorf("object_type, object_name, and source_code are required")
	}

	input := SyntaxCheckInput{ObjectType: objectType, ObjectName: objectName, SourceCode: sourceCode}
	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleSyntaxCheck(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"has_errors": output.HasErrors, "messages": output.Messages, "count": output.Count,
	}, nil
}

// handleFormatCode handles format-code tool request
func (s *NATSMCPServer) handleFormatCode(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	sourceCode, _ := req.Arguments["source_code"].(string)
	if sourceCode == "" {
		return nil, fmt.Errorf("source_code is required")
	}

	input := FormatCodeInput{SourceCode: sourceCode}
	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleFormatCode(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"formatted_code": output.FormattedCode}, nil
}

// handleTransportInfo handles transport-info tool request
func (s *NATSMCPServer) handleTransportInfo(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	objectType, _ := req.Arguments["object_type"].(string)
	objectName, _ := req.Arguments["object_name"].(string)
	pkg, _ := req.Arguments["package"].(string)

	if objectType == "" || objectName == "" {
		return nil, fmt.Errorf("object_type and object_name are required")
	}

	input := TransportInfoInput{ObjectType: objectType, ObjectName: objectName, Package: pkg}
	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleTransportInfo(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"object": output.Object, "package": output.Package,
		"transports": output.Transports, "count": output.Count,
	}, nil
}

// handleCreateTransport handles create-transport tool request
func (s *NATSMCPServer) handleCreateTransport(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	objectType, _ := req.Arguments["object_type"].(string)
	objectName, _ := req.Arguments["object_name"].(string)
	description, _ := req.Arguments["description"].(string)
	pkg, _ := req.Arguments["package"].(string)

	if objectType == "" || objectName == "" || description == "" {
		return nil, fmt.Errorf("object_type, object_name, and description are required")
	}

	input := CreateTransportInput{ObjectType: objectType, ObjectName: objectName, Description: description, Package: pkg}
	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleCreateTransport(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"success": output.Success, "transport_number": output.TransportNumber,
		"description": output.Description, "message": output.Message,
	}, nil
}

// sendSuccessResponse sends a success response back via NATS
func (s *NATSMCPServer) sendSuccessResponse(msg *nats.Msg, requestID string, result map[string]interface{}) {
	response := MCPToolResponse{
		RequestID: requestID,
		Success:   true,
		Result:    result,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("Error marshaling success response: %v\n", err)
		return
	}

	if msg.Reply != "" {
		if err := msg.Respond(data); err != nil {
			fmt.Printf("Error sending response: %v\n", err)
		}
	}

	responseSubject := s.getResponseSubject(msg.Subject)
	if responseSubject != "" {
		_ = s.natsConn.Publish(responseSubject, data)
	}
}

// sendErrorResponse sends an error response back via NATS
func (s *NATSMCPServer) sendErrorResponse(msg *nats.Msg, requestID string, err error) {
	response := MCPToolResponse{
		RequestID: requestID,
		Success:   false,
		Error:     err.Error(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	data, _ := json.Marshal(response)

	if msg.Reply != "" {
		_ = msg.Respond(data)
	}

	responseSubject := s.getResponseSubject(msg.Subject)
	if responseSubject != "" {
		_ = s.natsConn.Publish(responseSubject, data)
	}
}

// getResponseSubject generates a response subject from request subject
func (s *NATSMCPServer) getResponseSubject(requestSubject string) string {
	return strings.Replace(requestSubject, ".request", ".response", 1)
}
