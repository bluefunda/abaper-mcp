package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bluefunda/abaper/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerTools registers all MCP tools
func registerTools(server *mcp.Server, handlers *Handlers) {
	// Tool: get-object - Retrieve ABAP object source code
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get-object",
		Description: "Retrieve source code for an ABAP object (program, class, function, interface, etc.)",
	}, handlers.HandleGetObject)

	// Tool: search-objects - Search for ABAP objects
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search-objects",
		Description: "Search for ABAP objects by name pattern with wildcard support",
	}, handlers.HandleSearchObjects)

	// Tool: list-packages - List ABAP packages
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list-packages",
		Description: "List all ABAP packages in the system",
	}, handlers.HandleListPackages)

	// Tool: test-connection - Test ADT connection
	mcp.AddTool(server, &mcp.Tool{
		Name:        "test-connection",
		Description: "Test connectivity to the SAP ADT system",
	}, handlers.HandleTestConnection)

	// Tool: create-program - Create a new ABAP program
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create-program",
		Description: "Create a new ABAP program with source code",
	}, handlers.HandleCreateProgram)

	// Tool: create-class - Create a new ABAP class
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create-class",
		Description: "Create a new ABAP class with source code",
	}, handlers.HandleCreateClass)
}

// isSessionError checks if an error indicates a session timeout or authentication failure
// that can be resolved by reconnecting
func isSessionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "session timed out") ||
		strings.Contains(errStr, "session timeout") ||
		strings.Contains(errStr, "session expired") ||
		strings.Contains(errStr, "session may have expired") ||
		strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "authentication failed")
}

// getClientWithRetry attempts to get an ADT client with automatic retry logic
func (h *Handlers) getClientWithRetry(maxRetries int) (types.ADTClient, error) {
	var client types.ADTClient
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		client, err = h.clientManager.GetClient()
		if err == nil {
			return client, nil
		}

		if attempt < maxRetries {
			log.Printf("ADT connection attempt %d/%d failed: %v", attempt+1, maxRetries+1, err)
			h.clientManager.Reset() // Clear failed client
			// Exponential backoff: 1s, 2s, 3s...
			backoffDuration := time.Duration(attempt+1) * time.Second
			log.Printf("Retrying in %v...", backoffDuration)
			time.Sleep(backoffDuration)
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries+1, err)
}

// executeWithSessionRetry executes an operation and retries with a fresh connection if session errors occur
func (h *Handlers) executeWithSessionRetry(operation func(client types.ADTClient) error) error {
	maxRetries := 2

	for attempt := 0; attempt <= maxRetries; attempt++ {
		client, err := h.clientManager.GetClient()
		if err != nil {
			if attempt < maxRetries {
				log.Printf("Failed to get client (attempt %d/%d): %v", attempt+1, maxRetries+1, err)
				h.clientManager.Reset()
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			return fmt.Errorf("failed to get ADT client after %d attempts: %w", maxRetries+1, err)
		}

		err = operation(client)
		if err == nil {
			return nil
		}

		// Check if this is a session-related error that can be fixed by reconnecting
		if isSessionError(err) && attempt < maxRetries {
			log.Printf("Session error detected (attempt %d/%d): %v", attempt+1, maxRetries+1, err)
			log.Printf("Resetting client and retrying with fresh connection...")
			h.clientManager.Reset()
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}

		return err
	}

	return fmt.Errorf("operation failed after %d attempts", maxRetries+1)
}

// GetObjectInput defines input for get-object tool
type GetObjectInput struct {
	ObjectType    string `json:"object_type" jsonschema:"Type of ABAP object (program/class/function/interface/table/structure)"`
	ObjectName    string `json:"object_name" jsonschema:"Name of the ABAP object"`
	FunctionGroup string `json:"function_group,omitempty" jsonschema:"Function group name (required for function modules)"`
}

// GetObjectOutput defines output for get-object tool
type GetObjectOutput struct {
	ObjectType  string `json:"object_type" jsonschema:"Type of ABAP object"`
	ObjectName  string `json:"object_name" jsonschema:"Name of the ABAP object"`
	SourceCode  string `json:"source_code" jsonschema:"Source code of the object"`
	Description string `json:"description,omitempty" jsonschema:"Object description if available"`
}

// HandleGetObject retrieves ABAP object source code
func (h *Handlers) HandleGetObject(ctx context.Context, req *mcp.CallToolRequest, input GetObjectInput) (*mcp.CallToolResult, GetObjectOutput, error) {
	objectType := strings.ToLower(input.ObjectType)

	// Validate function group requirement early
	if (objectType == "function" || objectType == "func") && input.FunctionGroup == "" {
		return &mcp.CallToolResult{IsError: true}, GetObjectOutput{}, fmt.Errorf("function_group is required for function modules")
	}

	// Validate object type early
	validTypes := map[string]bool{
		"program": true, "prog": true,
		"class": true, "clas": true,
		"function": true, "func": true,
		"interface": true, "intf": true,
		"table": true, "tabl": true,
		"structure": true, "stru": true,
		"include": true, "incl": true,
	}
	if !validTypes[objectType] {
		return &mcp.CallToolResult{IsError: true}, GetObjectOutput{}, fmt.Errorf("unsupported object type: %s", input.ObjectType)
	}

	var source *types.ADTSourceCode

	err := h.executeWithSessionRetry(func(client types.ADTClient) error {
		var getErr error
		switch objectType {
		case "program", "prog":
			source, getErr = client.GetProgram(input.ObjectName)
		case "class", "clas":
			source, getErr = client.GetClass(input.ObjectName)
		case "function", "func":
			source, getErr = client.GetFunction(input.ObjectName, input.FunctionGroup)
		case "interface", "intf":
			source, getErr = client.GetInterface(input.ObjectName)
		case "table", "tabl":
			source, getErr = client.GetTable(input.ObjectName)
		case "structure", "stru":
			source, getErr = client.GetStructure(input.ObjectName)
		case "include", "incl":
			source, getErr = client.GetInclude(input.ObjectName)
		}
		return getErr
	})

	if err != nil {
		return &mcp.CallToolResult{IsError: true}, GetObjectOutput{}, fmt.Errorf("failed to get object: %w", err)
	}

	output := GetObjectOutput{
		ObjectType: input.ObjectType,
		ObjectName: input.ObjectName,
		SourceCode: source.Source,
	}

	return nil, output, nil
}

// SearchObjectsInput defines input for search-objects tool
type SearchObjectsInput struct {
	Pattern     string   `json:"pattern" jsonschema:"Search pattern with wildcard support (e.g. Z* or *TEST*)"`
	ObjectTypes []string `json:"object_types,omitempty" jsonschema:"Filter by object types (program/class/function/interface)"`
}

// SearchObjectsOutput defines output for search-objects tool
type SearchObjectsOutput struct {
	Objects []ObjectInfo `json:"objects" jsonschema:"List of found ABAP objects"`
	Count   int          `json:"count" jsonschema:"Total number of objects found"`
}

// ObjectInfo holds information about a found object
type ObjectInfo struct {
	Type        string `json:"type" jsonschema:"Object type"`
	Name        string `json:"name" jsonschema:"Object name"`
	Description string `json:"description,omitempty" jsonschema:"Object description"`
	Package     string `json:"package,omitempty" jsonschema:"Package name"`
}

// HandleSearchObjects searches for ABAP objects
func (h *Handlers) HandleSearchObjects(ctx context.Context, req *mcp.CallToolRequest, input SearchObjectsInput) (*mcp.CallToolResult, SearchObjectsOutput, error) {
	var results *types.ADTSearchResult

	err := h.executeWithSessionRetry(func(client types.ADTClient) error {
		var searchErr error
		results, searchErr = client.SearchObjects(input.Pattern, input.ObjectTypes)
		return searchErr
	})

	if err != nil {
		return &mcp.CallToolResult{IsError: true}, SearchObjectsOutput{}, fmt.Errorf("search failed: %w", err)
	}

	objects := make([]ObjectInfo, 0, len(results.Objects))
	for _, obj := range results.Objects {
		objects = append(objects, ObjectInfo{
			Type:        obj.Type,
			Name:        obj.Name,
			Description: obj.Description,
			Package:     obj.Package,
		})
	}

	output := SearchObjectsOutput{
		Objects: objects,
		Count:   len(objects),
	}

	return nil, output, nil
}

// ListPackagesInput defines input for list-packages tool
type ListPackagesInput struct {
	// No input parameters needed
}

// ListPackagesOutput defines output for list-packages tool
type ListPackagesOutput struct {
	Packages []PackageInfo `json:"packages" jsonschema:"List of ABAP packages"`
	Count    int           `json:"count" jsonschema:"Total number of packages"`
}

// PackageInfo holds information about a package
type PackageInfo struct {
	Name        string `json:"name" jsonschema:"Package name"`
	Description string `json:"description,omitempty" jsonschema:"Package description"`
}

// HandleListPackages lists ABAP packages
func (h *Handlers) HandleListPackages(ctx context.Context, req *mcp.CallToolRequest, input ListPackagesInput) (*mcp.CallToolResult, ListPackagesOutput, error) {
	var packages []types.ADTPackage

	err := h.executeWithSessionRetry(func(client types.ADTClient) error {
		var listErr error
		packages, listErr = client.ListPackages("*")
		return listErr
	})

	if err != nil {
		return &mcp.CallToolResult{IsError: true}, ListPackagesOutput{}, fmt.Errorf("failed to list packages: %w", err)
	}

	pkgInfos := make([]PackageInfo, 0, len(packages))
	for _, pkg := range packages {
		pkgInfos = append(pkgInfos, PackageInfo{
			Name:        pkg.Name,
			Description: pkg.Description,
		})
	}

	output := ListPackagesOutput{
		Packages: pkgInfos,
		Count:    len(pkgInfos),
	}

	return nil, output, nil
}

// TestConnectionInput defines input for test-connection tool
type TestConnectionInput struct {
	// No input parameters needed
}

// TestConnectionOutput defines output for test-connection tool
type TestConnectionOutput struct {
	Connected bool   `json:"connected" jsonschema:"Whether connection was successful"`
	Message   string `json:"message" jsonschema:"Connection status message"`
}

// HandleTestConnection tests ADT connection
func (h *Handlers) HandleTestConnection(ctx context.Context, req *mcp.CallToolRequest, input TestConnectionInput) (*mcp.CallToolResult, TestConnectionOutput, error) {
	err := h.executeWithSessionRetry(func(client types.ADTClient) error {
		return client.TestConnection()
	})

	if err != nil {
		return nil, TestConnectionOutput{
			Connected: false,
			Message:   fmt.Sprintf("Connection failed: %v", err),
		}, nil
	}

	return nil, TestConnectionOutput{
		Connected: true,
		Message:   "Successfully connected to SAP ADT system",
	}, nil
}

// CreateProgramInput defines input for create-program tool
type CreateProgramInput struct {
	Name        string `json:"name" jsonschema:"Program name (e.g. ZTEST_PROG)"`
	Description string `json:"description" jsonschema:"Program description"`
	Package     string `json:"package" jsonschema:"Package name (use $TMP for local objects)"`
	SourceCode  string `json:"source_code" jsonschema:"ABAP source code"`
}

// CreateProgramOutput defines output for create-program tool
type CreateProgramOutput struct {
	Success bool   `json:"success" jsonschema:"Whether creation was successful"`
	Message string `json:"message" jsonschema:"Result message"`
	Name    string `json:"name" jsonschema:"Created program name"`
}

// HandleCreateProgram creates a new ABAP program
func (h *Handlers) HandleCreateProgram(ctx context.Context, req *mcp.CallToolRequest, input CreateProgramInput) (*mcp.CallToolResult, CreateProgramOutput, error) {
	err := h.executeWithSessionRetry(func(client types.ADTClient) error {
		return client.CreateProgram(input.Name, input.Description, input.Package, input.SourceCode)
	})

	if err != nil {
		return nil, CreateProgramOutput{
			Success: false,
			Message: fmt.Sprintf("Failed to create program: %v", err),
			Name:    input.Name,
		}, nil
	}

	return nil, CreateProgramOutput{
		Success: true,
		Message: "Program created successfully",
		Name:    input.Name,
	}, nil
}

// CreateClassInput defines input for create-class tool
type CreateClassInput struct {
	Name        string `json:"name" jsonschema:"Class name (e.g. ZCL_TEST)"`
	Description string `json:"description" jsonschema:"Class description"`
	Package     string `json:"package" jsonschema:"Package name (use $TMP for local objects)"`
	SourceCode  string `json:"source_code" jsonschema:"ABAP class source code"`
}

// CreateClassOutput defines output for create-class tool
type CreateClassOutput struct {
	Success bool   `json:"success" jsonschema:"Whether creation was successful"`
	Message string `json:"message" jsonschema:"Result message"`
	Name    string `json:"name" jsonschema:"Created class name"`
}

// HandleCreateClass creates a new ABAP class
func (h *Handlers) HandleCreateClass(ctx context.Context, req *mcp.CallToolRequest, input CreateClassInput) (*mcp.CallToolResult, CreateClassOutput, error) {
	err := h.executeWithSessionRetry(func(client types.ADTClient) error {
		return client.CreateClass(input.Name, input.Description, input.Package, input.SourceCode)
	})

	if err != nil {
		return nil, CreateClassOutput{
			Success: false,
			Message: fmt.Sprintf("Failed to create class: %v", err),
			Name:    input.Name,
		}, nil
	}

	return nil, CreateClassOutput{
		Success: true,
		Message: "Class created successfully",
		Name:    input.Name,
	}, nil
}

// marshalJSON is a helper to convert output to JSON for text content
func marshalJSON(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error marshaling output: %v", err)
	}
	return string(data)
}
