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
	// Subscribe to MCP tool requests
	// Subject pattern: mcp.abaper.tools.request
	subject := "mcp.abaper.tools.request"
	_, err := s.natsConn.Subscribe(subject, s.handleToolRequest)
	if err != nil {
		return fmt.Errorf("failed to subscribe to tool requests: %w", err)
	}

	// Subscribe to wildcard for realm-based routing
	// Subject pattern: mcp.*.abaper.tools.request
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

	// Parse the request
	var req MCPToolRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		s.sendErrorResponse(msg, "", fmt.Errorf("failed to parse request: %w", err))
		return
	}

	// Route to appropriate tool handler
	ctx := context.Background()
	result, err := s.routeToolRequest(ctx, &req)

	// Send response
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
	case "analyze-s4-remediation":
		return s.handleAnalyzeS4Remediation(ctx, req)
	default:
		return nil, fmt.Errorf("unknown tool: %s", req.Tool)
	}
}

// handleGetObject handles get-object tool request
func (s *NATSMCPServer) handleGetObject(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	// Parse arguments
	objectType, ok := req.Arguments["object_type"].(string)
	if !ok {
		return nil, fmt.Errorf("object_type is required")
	}

	objectName, ok := req.Arguments["object_name"].(string)
	if !ok {
		return nil, fmt.Errorf("object_name is required")
	}

	functionGroup, _ := req.Arguments["function_group"].(string)

	// Create input
	input := GetObjectInput{
		ObjectType:    objectType,
		ObjectName:    objectName,
		FunctionGroup: functionGroup,
	}

	// Call handler
	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleGetObject(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}

	// Convert output to map
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
	pattern, ok := req.Arguments["pattern"].(string)
	if !ok {
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

	input := SearchObjectsInput{
		Pattern:     pattern,
		ObjectTypes: objectTypes,
	}

	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleSearchObjects(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}

	// Convert to map
	objects := make([]map[string]interface{}, len(output.Objects))
	for i, obj := range output.Objects {
		objects[i] = map[string]interface{}{
			"type":        obj.Type,
			"name":        obj.Name,
			"description": obj.Description,
			"package":     obj.Package,
		}
	}

	return map[string]interface{}{
		"objects": objects,
		"count":   output.Count,
	}, nil
}

// handleListPackages handles list-packages tool request
func (s *NATSMCPServer) handleListPackages(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	input := ListPackagesInput{}
	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleListPackages(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}

	packages := make([]map[string]interface{}, len(output.Packages))
	for i, pkg := range output.Packages {
		packages[i] = map[string]interface{}{
			"name":        pkg.Name,
			"description": pkg.Description,
		}
	}

	return map[string]interface{}{
		"packages": packages,
		"count":    output.Count,
	}, nil
}

// handleTestConnection handles test-connection tool request
func (s *NATSMCPServer) handleTestConnection(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	input := TestConnectionInput{}
	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleTestConnection(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"connected": output.Connected,
		"message":   output.Message,
	}, nil
}

// handleCreateProgram handles create-program tool request
func (s *NATSMCPServer) handleCreateProgram(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	name, ok := req.Arguments["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is required")
	}

	description, ok := req.Arguments["description"].(string)
	if !ok {
		return nil, fmt.Errorf("description is required")
	}

	pkg, ok := req.Arguments["package"].(string)
	if !ok {
		return nil, fmt.Errorf("package is required")
	}

	sourceCode, ok := req.Arguments["source_code"].(string)
	if !ok {
		return nil, fmt.Errorf("source_code is required")
	}

	input := CreateProgramInput{
		Name:        name,
		Description: description,
		Package:     pkg,
		SourceCode:  sourceCode,
	}

	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleCreateProgram(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": output.Success,
		"message": output.Message,
		"name":    output.Name,
	}, nil
}

// handleCreateClass handles create-class tool request
func (s *NATSMCPServer) handleCreateClass(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	name, ok := req.Arguments["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is required")
	}

	description, ok := req.Arguments["description"].(string)
	if !ok {
		return nil, fmt.Errorf("description is required")
	}

	pkg, ok := req.Arguments["package"].(string)
	if !ok {
		return nil, fmt.Errorf("package is required")
	}

	sourceCode, ok := req.Arguments["source_code"].(string)
	if !ok {
		return nil, fmt.Errorf("source_code is required")
	}

	input := CreateClassInput{
		Name:        name,
		Description: description,
		Package:     pkg,
		SourceCode:  sourceCode,
	}

	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleCreateClass(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": output.Success,
		"message": output.Message,
		"name":    output.Name,
	}, nil
}

// handleAnalyzeS4Remediation handles analyze-s4-remediation tool request
func (s *NATSMCPServer) handleAnalyzeS4Remediation(ctx context.Context, req *MCPToolRequest) (map[string]interface{}, error) {
	objectType, ok := req.Arguments["object_type"].(string)
	if !ok {
		return nil, fmt.Errorf("object_type is required")
	}

	objectName, ok := req.Arguments["object_name"].(string)
	if !ok {
		return nil, fmt.Errorf("object_name is required")
	}

	functionGroup, _ := req.Arguments["function_group"].(string)
	outputFormat, _ := req.Arguments["output_format"].(string)

	input := AnalyzeS4RemediationInput{
		ObjectType:    objectType,
		ObjectName:    objectName,
		FunctionGroup: functionGroup,
		OutputFormat:  outputFormat,
	}

	mcpReq := &mcp.CallToolRequest{}
	_, output, err := s.handlers.HandleAnalyzeS4Remediation(ctx, mcpReq, input)
	if err != nil {
		return nil, err
	}

	// Convert issues to map slice
	issues := make([]map[string]interface{}, len(output.JSON.Issues))
	for i, issue := range output.JSON.Issues {
		issues[i] = map[string]interface{}{
			"pattern_id":      issue.PatternID,
			"pattern_title":   issue.PatternTitle,
			"symptom_detected": issue.SymptomDetected,
			"severity":        issue.Severity,
			"before_code":     issue.BeforeCode,
			"after_code":      issue.AfterCode,
			"fix_description": issue.FixDescription,
		}
	}

	return map[string]interface{}{
		"json": map[string]interface{}{
			"run_metadata": map[string]interface{}{
				"run_id":         output.JSON.RunMetadata.RunID,
				"timestamp_utc":  output.JSON.RunMetadata.TimestampUTC,
				"system_id":      output.JSON.RunMetadata.SystemID,
				"system_release": output.JSON.RunMetadata.SystemRelease,
				"client":         output.JSON.RunMetadata.Client,
				"analyst":        output.JSON.RunMetadata.Analyst,
			},
			"artifact": map[string]interface{}{
				"artifact_name":     output.JSON.Artifact.ArtifactName,
				"artifact_type":     output.JSON.Artifact.ArtifactType,
				"package":           output.JSON.Artifact.Package,
				"transport_request": output.JSON.Artifact.TransportRequest,
			},
			"issues": issues,
		},
		"markdown": output.Markdown,
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

	// Reply to the request
	if msg.Reply != "" {
		if err := msg.Respond(data); err != nil {
			fmt.Printf("Error sending response: %v\n", err)
		}
	}

	// Also publish to response subject if extractable from request subject
	responseSubject := s.getResponseSubject(msg.Subject)
	if responseSubject != "" {
		s.natsConn.Publish(responseSubject, data)
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

	// Reply to the request
	if msg.Reply != "" {
		msg.Respond(data)
	}

	// Also publish to response subject
	responseSubject := s.getResponseSubject(msg.Subject)
	if responseSubject != "" {
		s.natsConn.Publish(responseSubject, data)
	}
}

// getResponseSubject generates a response subject from request subject
func (s *NATSMCPServer) getResponseSubject(requestSubject string) string {
	// Convert mcp.abaper.tools.request -> mcp.abaper.tools.response
	// Convert mcp.production.abaper.tools.request -> mcp.production.abaper.tools.response
	return strings.Replace(requestSubject, ".request", ".response", 1)
}

// extractRealm extracts realm from NATS subject
func extractRealm(subject string) string {
	parts := strings.Split(subject, ".")
	if len(parts) >= 4 && parts[0] == "mcp" {
		// Format: mcp.<realm>.abaper.tools.request
		return parts[1]
	}
	return ""
}
