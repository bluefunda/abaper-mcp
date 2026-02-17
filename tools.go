package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bluefunda/abaper-mcp/internal/logger"
	"github.com/bluefunda/abaper/types"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
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

	// Tool: analyze-s4-remediation - Analyze ABAP code for S/4HANA compatibility
	mcp.AddTool(server, &mcp.Tool{
		Name:        "analyze-s4-remediation",
		Description: "Analyze ABAP code for S/4HANA compatibility issues and provide remediation suggestions. Returns both structured JSON and human-readable Markdown report formats.",
	}, handlers.HandleAnalyzeS4Remediation)

	// Tool: activate-object - Activate an ABAP object
	mcp.AddTool(server, &mcp.Tool{
		Name:        "activate-object",
		Description: "Activate an ABAP object (program, class, interface, function group, include). Must be called before running unit tests.",
	}, handlers.HandleActivateObject)

	// Tool: run-unit-tests - Run ABAP unit tests
	mcp.AddTool(server, &mcp.Tool{
		Name:        "run-unit-tests",
		Description: "Run ABAP unit tests for an object and return pass/fail results with detailed test method outcomes.",
	}, handlers.HandleRunUnitTests)

	// Tool: update-program - Update existing ABAP program source
	mcp.AddTool(server, &mcp.Tool{
		Name:        "update-program",
		Description: "Update the source code of an existing ABAP program. Provide complete source code, not diffs.",
	}, handlers.HandleUpdateProgram)

	// Tool: update-class - Update existing ABAP class source
	mcp.AddTool(server, &mcp.Tool{
		Name:        "update-class",
		Description: "Update the source code of an existing ABAP class. Provide complete source code, not diffs.",
	}, handlers.HandleUpdateClass)
}

// getClient creates a fresh ADT client for the operation
// Each request gets a new connection to avoid session timeout issues
func (h *Handlers) getClient() (types.ADTClient, error) {
	return h.clientManager.GetClient()
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
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "get-object")

	log.Info("Tool execution started",
		zap.String("object_type", input.ObjectType),
		zap.String("object_name", input.ObjectName),
		zap.String("function_group", input.FunctionGroup),
	)

	objectType := strings.ToLower(input.ObjectType)

	// Validate function group requirement early
	if (objectType == "function" || objectType == "func") && input.FunctionGroup == "" {
		log.Warn("Validation failed: function_group required")
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
		log.Warn("Validation failed: unsupported object type", zap.String("object_type", input.ObjectType))
		return &mcp.CallToolResult{IsError: true}, GetObjectOutput{}, fmt.Errorf("unsupported object type: %s", input.ObjectType)
	}

	// Get fresh client for this request
	client, err := h.getClient()
	if err != nil {
		log.Error("Failed to get ADT client", zap.Error(err))
		return &mcp.CallToolResult{IsError: true}, GetObjectOutput{}, fmt.Errorf("failed to connect: %w", err)
	}

	var source *types.ADTSourceCode
	switch objectType {
	case "program", "prog":
		source, err = client.GetProgram(input.ObjectName)
	case "class", "clas":
		source, err = client.GetClass(input.ObjectName)
	case "function", "func":
		source, err = client.GetFunction(input.ObjectName, input.FunctionGroup)
	case "interface", "intf":
		source, err = client.GetInterface(input.ObjectName)
	case "table", "tabl":
		source, err = client.GetTable(input.ObjectName)
	case "structure", "stru":
		source, err = client.GetStructure(input.ObjectName)
	case "include", "incl":
		source, err = client.GetInclude(input.ObjectName)
	}

	if err != nil {
		log.Error("Failed to get object", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return &mcp.CallToolResult{IsError: true}, GetObjectOutput{}, fmt.Errorf("failed to get object: %w", err)
	}

	output := GetObjectOutput{
		ObjectType: input.ObjectType,
		ObjectName: input.ObjectName,
		SourceCode: source.Source,
	}

	log.Info("Tool execution completed",
		zap.Int("source_len", len(source.Source)),
		zap.Duration("duration", time.Since(start)),
	)

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
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "search-objects")

	log.Info("Tool execution started",
		zap.String("pattern", input.Pattern),
		zap.Strings("object_types", input.ObjectTypes),
	)

	// Get fresh client for this request
	client, err := h.getClient()
	if err != nil {
		log.Error("Failed to get ADT client", zap.Error(err))
		return &mcp.CallToolResult{IsError: true}, SearchObjectsOutput{}, fmt.Errorf("failed to connect: %w", err)
	}

	results, err := client.SearchObjects(input.Pattern, input.ObjectTypes)
	if err != nil {
		log.Error("Search failed", zap.Error(err), zap.Duration("duration", time.Since(start)))
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

	log.Info("Tool execution completed",
		zap.Int("results_count", len(objects)),
		zap.Duration("duration", time.Since(start)),
	)

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
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "list-packages")

	log.Info("Tool execution started")

	// Get fresh client for this request
	client, err := h.getClient()
	if err != nil {
		log.Error("Failed to get ADT client", zap.Error(err))
		return &mcp.CallToolResult{IsError: true}, ListPackagesOutput{}, fmt.Errorf("failed to connect: %w", err)
	}

	packages, err := client.ListPackages("*")
	if err != nil {
		log.Error("Failed to list packages", zap.Error(err), zap.Duration("duration", time.Since(start)))
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

	log.Info("Tool execution completed",
		zap.Int("packages_count", len(pkgInfos)),
		zap.Duration("duration", time.Since(start)),
	)

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
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "test-connection")

	log.Info("Tool execution started")

	// Get fresh client for this request
	client, err := h.getClient()
	if err != nil {
		log.Error("Failed to get ADT client", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return nil, TestConnectionOutput{
			Connected: false,
			Message:   fmt.Sprintf("Connection failed: %v", err),
		}, nil
	}

	err = client.TestConnection()
	if err != nil {
		log.Warn("Connection test failed", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return nil, TestConnectionOutput{
			Connected: false,
			Message:   fmt.Sprintf("Connection failed: %v", err),
		}, nil
	}

	log.Info("Tool execution completed",
		zap.Bool("connected", true),
		zap.Duration("duration", time.Since(start)),
	)

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
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "create-program")

	log.Info("Tool execution started",
		zap.String("name", input.Name),
		zap.String("package", input.Package),
		zap.Int("source_len", len(input.SourceCode)),
	)

	// Get fresh client for this request
	client, err := h.getClient()
	if err != nil {
		log.Error("Failed to get ADT client", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return nil, CreateProgramOutput{
			Success: false,
			Message: fmt.Sprintf("Failed to connect: %v", err),
			Name:    input.Name,
		}, nil
	}

	err = client.CreateProgram(input.Name, input.Description, input.Package, input.SourceCode)
	if err != nil {
		log.Error("Failed to create program", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return nil, CreateProgramOutput{
			Success: false,
			Message: fmt.Sprintf("Failed to create program: %v", err),
			Name:    input.Name,
		}, nil
	}

	log.Info("Tool execution completed",
		zap.Bool("success", true),
		zap.Duration("duration", time.Since(start)),
	)

	return nil, CreateProgramOutput{
		Success: true,
		Message: "Program created successfully",
		Name:    input.Name,
	}, nil
}

// ActivateObjectInput defines input for activate-object tool
type ActivateObjectInput struct {
	ObjectType string `json:"object_type" jsonschema:"Type of ABAP object (program/class/interface/include/function_group)"`
	ObjectName string `json:"object_name" jsonschema:"Name of the ABAP object to activate"`
}

// ActivateObjectOutput defines output for activate-object tool
type ActivateObjectOutput struct {
	Success    bool   `json:"success" jsonschema:"Whether activation was successful"`
	Message    string `json:"message" jsonschema:"Result message with any warnings or errors"`
	ObjectName string `json:"object_name" jsonschema:"Activated object name"`
	ObjectType string `json:"object_type" jsonschema:"Activated object type"`
}

// HandleActivateObject activates an ABAP object
func (h *Handlers) HandleActivateObject(ctx context.Context, req *mcp.CallToolRequest, input ActivateObjectInput) (*mcp.CallToolResult, ActivateObjectOutput, error) {
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "activate-object")

	log.Info("Tool execution started",
		zap.String("object_type", input.ObjectType),
		zap.String("object_name", input.ObjectName),
	)

	objectType := strings.ToLower(input.ObjectType)
	validTypes := map[string]bool{
		"program": true, "prog": true,
		"class": true, "clas": true,
		"interface": true, "intf": true,
		"include": true, "incl": true,
		"function_group": true, "fugr": true,
	}
	if !validTypes[objectType] {
		log.Warn("Validation failed: unsupported object type", zap.String("object_type", input.ObjectType))
		return &mcp.CallToolResult{IsError: true}, ActivateObjectOutput{}, fmt.Errorf("unsupported object type: %s", input.ObjectType)
	}

	client, err := h.getClient()
	if err != nil {
		log.Error("Failed to get ADT client", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return nil, ActivateObjectOutput{
			Success:    false,
			Message:    fmt.Sprintf("Failed to connect: %v", err),
			ObjectName: input.ObjectName,
			ObjectType: input.ObjectType,
		}, nil
	}

	result, err := client.ActivateObject(input.ObjectType, input.ObjectName)
	if err != nil {
		log.Error("Activation failed", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return nil, ActivateObjectOutput{
			Success:    false,
			Message:    fmt.Sprintf("Activation failed: %v", err),
			ObjectName: input.ObjectName,
			ObjectType: input.ObjectType,
		}, nil
	}

	message := "Object activated successfully"
	if !result.Success {
		var msgs []string
		for _, m := range result.Messages {
			msgs = append(msgs, fmt.Sprintf("[%s] %s", m.Severity, m.Text))
		}
		message = "Activation failed: " + strings.Join(msgs, "; ")
	} else if len(result.Messages) > 0 {
		var msgs []string
		for _, m := range result.Messages {
			msgs = append(msgs, fmt.Sprintf("[%s] %s", m.Severity, m.Text))
		}
		message = "Object activated with messages: " + strings.Join(msgs, "; ")
	}

	log.Info("Tool execution completed",
		zap.Bool("success", result.Success),
		zap.Duration("duration", time.Since(start)),
	)

	return nil, ActivateObjectOutput{
		Success:    result.Success,
		Message:    message,
		ObjectName: input.ObjectName,
		ObjectType: input.ObjectType,
	}, nil
}

// RunUnitTestsInput defines input for run-unit-tests tool
type RunUnitTestsInput struct {
	ObjectType string `json:"object_type" jsonschema:"Type of ABAP object (program/class/interface)"`
	ObjectName string `json:"object_name" jsonschema:"Name of the ABAP object to test"`
}

// RunUnitTestsOutput defines output for run-unit-tests tool
type RunUnitTestsOutput struct {
	AllPassed  bool   `json:"all_passed" jsonschema:"Whether all tests passed"`
	TotalTests int    `json:"total_tests" jsonschema:"Total number of tests executed"`
	Passed     int    `json:"passed" jsonschema:"Number of passed tests"`
	Failed     int    `json:"failed" jsonschema:"Number of failed tests"`
	Details    string `json:"details" jsonschema:"Human-readable test results"`
	ObjectName string `json:"object_name" jsonschema:"Tested object name"`
}

// HandleRunUnitTests runs ABAP unit tests
func (h *Handlers) HandleRunUnitTests(ctx context.Context, req *mcp.CallToolRequest, input RunUnitTestsInput) (*mcp.CallToolResult, RunUnitTestsOutput, error) {
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "run-unit-tests")

	log.Info("Tool execution started",
		zap.String("object_type", input.ObjectType),
		zap.String("object_name", input.ObjectName),
	)

	objectType := strings.ToLower(input.ObjectType)
	validTypes := map[string]bool{
		"program": true, "prog": true,
		"class": true, "clas": true,
		"interface": true, "intf": true,
		"include": true, "incl": true,
		"function_group": true, "fugr": true,
	}
	if !validTypes[objectType] {
		log.Warn("Validation failed: unsupported object type", zap.String("object_type", input.ObjectType))
		return &mcp.CallToolResult{IsError: true}, RunUnitTestsOutput{}, fmt.Errorf("unsupported object type: %s", input.ObjectType)
	}

	client, err := h.getClient()
	if err != nil {
		log.Error("Failed to get ADT client", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return nil, RunUnitTestsOutput{
			AllPassed:  false,
			Details:    fmt.Sprintf("Failed to connect: %v", err),
			ObjectName: input.ObjectName,
		}, nil
	}

	result, err := client.RunUnitTests(input.ObjectType, input.ObjectName)
	if err != nil {
		log.Error("Unit test execution failed", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return nil, RunUnitTestsOutput{
			AllPassed:  false,
			Details:    fmt.Sprintf("Test execution failed: %v", err),
			ObjectName: input.ObjectName,
		}, nil
	}

	// Build human-readable details
	var details strings.Builder
	fmt.Fprintf(&details, "Unit Test Results for %s\n", input.ObjectName)
	fmt.Fprintf(&details, "Total: %d | Passed: %d | Failed: %d\n\n", result.TotalTests, result.Passed, result.Failed)

	for _, tc := range result.TestClasses {
		fmt.Fprintf(&details, "Test Class: %s\n", tc.Name)
		for _, tm := range tc.Methods {
			status := "PASS"
			if tm.Status != "passed" {
				status = "FAIL"
			}
			fmt.Fprintf(&details, "  [%s] %s", status, tm.Name)
			if tm.Message != "" {
				fmt.Fprintf(&details, " - %s", tm.Message)
			}
			details.WriteString("\n")
		}
		details.WriteString("\n")
	}

	log.Info("Tool execution completed",
		zap.Bool("all_passed", result.AllPassed),
		zap.Int("total", result.TotalTests),
		zap.Int("passed", result.Passed),
		zap.Int("failed", result.Failed),
		zap.Duration("duration", time.Since(start)),
	)

	return nil, RunUnitTestsOutput{
		AllPassed:  result.AllPassed,
		TotalTests: result.TotalTests,
		Passed:     result.Passed,
		Failed:     result.Failed,
		Details:    details.String(),
		ObjectName: input.ObjectName,
	}, nil
}

// UpdateProgramInput defines input for update-program tool
type UpdateProgramInput struct {
	Name       string `json:"name" jsonschema:"Program name (e.g. ZTEST_PROG)"`
	SourceCode string `json:"source_code" jsonschema:"Complete ABAP source code (not diffs)"`
}

// UpdateProgramOutput defines output for update-program tool
type UpdateProgramOutput struct {
	Success bool   `json:"success" jsonschema:"Whether update was successful"`
	Message string `json:"message" jsonschema:"Result message"`
	Name    string `json:"name" jsonschema:"Updated program name"`
}

// HandleUpdateProgram updates an existing ABAP program
func (h *Handlers) HandleUpdateProgram(ctx context.Context, req *mcp.CallToolRequest, input UpdateProgramInput) (*mcp.CallToolResult, UpdateProgramOutput, error) {
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "update-program")

	log.Info("Tool execution started",
		zap.String("name", input.Name),
		zap.Int("source_len", len(input.SourceCode)),
	)

	client, err := h.getClient()
	if err != nil {
		log.Error("Failed to get ADT client", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return nil, UpdateProgramOutput{
			Success: false,
			Message: fmt.Sprintf("Failed to connect: %v", err),
			Name:    input.Name,
		}, nil
	}

	err = client.UpdateProgram(input.Name, input.SourceCode)
	if err != nil {
		log.Error("Failed to update program", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return nil, UpdateProgramOutput{
			Success: false,
			Message: fmt.Sprintf("Failed to update program: %v", err),
			Name:    input.Name,
		}, nil
	}

	log.Info("Tool execution completed",
		zap.Bool("success", true),
		zap.Duration("duration", time.Since(start)),
	)

	return nil, UpdateProgramOutput{
		Success: true,
		Message: "Program updated successfully",
		Name:    input.Name,
	}, nil
}

// UpdateClassInput defines input for update-class tool
type UpdateClassInput struct {
	Name       string `json:"name" jsonschema:"Class name (e.g. ZCL_TEST)"`
	SourceCode string `json:"source_code" jsonschema:"Complete ABAP class source code (not diffs)"`
}

// UpdateClassOutput defines output for update-class tool
type UpdateClassOutput struct {
	Success bool   `json:"success" jsonschema:"Whether update was successful"`
	Message string `json:"message" jsonschema:"Result message"`
	Name    string `json:"name" jsonschema:"Updated class name"`
}

// HandleUpdateClass updates an existing ABAP class
func (h *Handlers) HandleUpdateClass(ctx context.Context, req *mcp.CallToolRequest, input UpdateClassInput) (*mcp.CallToolResult, UpdateClassOutput, error) {
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "update-class")

	log.Info("Tool execution started",
		zap.String("name", input.Name),
		zap.Int("source_len", len(input.SourceCode)),
	)

	client, err := h.getClient()
	if err != nil {
		log.Error("Failed to get ADT client", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return nil, UpdateClassOutput{
			Success: false,
			Message: fmt.Sprintf("Failed to connect: %v", err),
			Name:    input.Name,
		}, nil
	}

	err = client.UpdateClass(input.Name, input.SourceCode)
	if err != nil {
		log.Error("Failed to update class", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return nil, UpdateClassOutput{
			Success: false,
			Message: fmt.Sprintf("Failed to update class: %v", err),
			Name:    input.Name,
		}, nil
	}

	log.Info("Tool execution completed",
		zap.Bool("success", true),
		zap.Duration("duration", time.Since(start)),
	)

	return nil, UpdateClassOutput{
		Success: true,
		Message: "Class updated successfully",
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
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "create-class")

	log.Info("Tool execution started",
		zap.String("name", input.Name),
		zap.String("package", input.Package),
		zap.Int("source_len", len(input.SourceCode)),
	)

	// Get fresh client for this request
	client, err := h.getClient()
	if err != nil {
		log.Error("Failed to get ADT client", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return nil, CreateClassOutput{
			Success: false,
			Message: fmt.Sprintf("Failed to connect: %v", err),
			Name:    input.Name,
		}, nil
	}

	err = client.CreateClass(input.Name, input.Description, input.Package, input.SourceCode)
	if err != nil {
		log.Error("Failed to create class", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return nil, CreateClassOutput{
			Success: false,
			Message: fmt.Sprintf("Failed to create class: %v", err),
			Name:    input.Name,
		}, nil
	}

	log.Info("Tool execution completed",
		zap.Bool("success", true),
		zap.Duration("duration", time.Since(start)),
	)

	return nil, CreateClassOutput{
		Success: true,
		Message: "Class created successfully",
		Name:    input.Name,
	}, nil
}

