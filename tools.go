package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bluefunda/abaper-mcp/internal/logger"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// registerTools registers all MCP tools
func registerTools(server *mcp.Server, handlers *Handlers) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get-object",
		Description: "Retrieve source code for an ABAP object (program, class, function, interface, table, structure, include)",
	}, handlers.HandleGetObject)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search-objects",
		Description: "Search for ABAP objects by name pattern with wildcard support",
	}, handlers.HandleSearchObjects)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list-packages",
		Description: "List all ABAP packages in the system",
	}, handlers.HandleListPackages)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "test-connection",
		Description: "Test connectivity to the SAP ADT system",
	}, handlers.HandleTestConnection)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create-program",
		Description: "Create a new ABAP program with source code",
	}, handlers.HandleCreateProgram)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create-class",
		Description: "Create a new ABAP class with source code",
	}, handlers.HandleCreateClass)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update-program",
		Description: "Update the source code of an existing ABAP program. Provide complete source code, not diffs.",
	}, handlers.HandleUpdateProgram)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update-class",
		Description: "Update the source code of an existing ABAP class. Provide complete source code, not diffs.",
	}, handlers.HandleUpdateClass)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "activate-object",
		Description: "Activate an ABAP object (program, class, interface, function group, include). Must be called before running unit tests.",
	}, handlers.HandleActivateObject)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "run-unit-tests",
		Description: "Run ABAP unit tests for an object and return pass/fail results with detailed test method outcomes.",
	}, handlers.HandleRunUnitTests)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "analyze-s4-remediation",
		Description: "Analyze ABAP code for S/4HANA compatibility issues and provide remediation suggestions. Returns both structured JSON and human-readable Markdown report formats.",
	}, handlers.HandleAnalyzeS4Remediation)

	// New tools enabled by abaper-ts

	mcp.AddTool(server, &mcp.Tool{
		Name:        "syntax-check",
		Description: "Perform a syntax check on ABAP source code and return errors/warnings with line numbers.",
	}, handlers.HandleSyntaxCheck)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "format-code",
		Description: "Format ABAP source code using the SAP pretty printer.",
	}, handlers.HandleFormatCode)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "transport-info",
		Description: "Get transport request information for an ABAP object.",
	}, handlers.HandleTransportInfo)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create-transport",
		Description: "Create a new transport request for an ABAP object.",
	}, handlers.HandleCreateTransport)
}

// --- Existing tool input/output types ---

// GetObjectInput defines input for get-object tool
type GetObjectInput struct {
	ObjectType    string `json:"object_type" jsonschema:"Type of ABAP object (program/class/function/interface/table/structure/include)"`
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

// HandleGetObject retrieves ABAP object source code via abaper-ts
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

	// Validate object type
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

	adtType := normalizeObjectType(input.ObjectType)
	result, err := h.apiClient.GetObject(adtType, input.ObjectName, input.FunctionGroup)
	if err != nil {
		log.Error("Failed to get object", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return &mcp.CallToolResult{IsError: true}, GetObjectOutput{}, fmt.Errorf("failed to get object: %w", err)
	}

	output := GetObjectOutput{
		ObjectType: input.ObjectType,
		ObjectName: input.ObjectName,
		SourceCode: result.Source,
	}

	log.Info("Tool execution completed",
		zap.Int("source_len", len(result.Source)),
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

// HandleSearchObjects searches for ABAP objects via abaper-ts
func (h *Handlers) HandleSearchObjects(ctx context.Context, req *mcp.CallToolRequest, input SearchObjectsInput) (*mcp.CallToolResult, SearchObjectsOutput, error) {
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "search-objects")

	log.Info("Tool execution started",
		zap.String("pattern", input.Pattern),
		zap.Strings("object_types", input.ObjectTypes),
	)

	results, err := h.apiClient.SearchObjects(input.Pattern, input.ObjectTypes)
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
type ListPackagesInput struct{}

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

// HandleListPackages lists ABAP packages via abaper-ts
func (h *Handlers) HandleListPackages(ctx context.Context, req *mcp.CallToolRequest, input ListPackagesInput) (*mcp.CallToolResult, ListPackagesOutput, error) {
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "list-packages")

	log.Info("Tool execution started")

	packages, err := h.apiClient.ListPackages()
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
type TestConnectionInput struct{}

// TestConnectionOutput defines output for test-connection tool
type TestConnectionOutput struct {
	Connected bool   `json:"connected" jsonschema:"Whether connection was successful"`
	Message   string `json:"message" jsonschema:"Connection status message"`
}

// HandleTestConnection tests ADT connection via abaper-ts
func (h *Handlers) HandleTestConnection(ctx context.Context, req *mcp.CallToolRequest, input TestConnectionInput) (*mcp.CallToolResult, TestConnectionOutput, error) {
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "test-connection")

	log.Info("Tool execution started")

	result, err := h.apiClient.TestConnection()
	if err != nil {
		log.Warn("Connection test failed", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return nil, TestConnectionOutput{
			Connected: false,
			Message:   fmt.Sprintf("Connection failed: %v", err),
		}, nil
	}

	log.Info("Tool execution completed",
		zap.Bool("connected", result.Authenticated),
		zap.Duration("duration", time.Since(start)),
	)

	return nil, TestConnectionOutput{
		Connected: result.Authenticated,
		Message:   result.Message,
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

// HandleCreateProgram creates a new ABAP program via abaper-ts
func (h *Handlers) HandleCreateProgram(ctx context.Context, req *mcp.CallToolRequest, input CreateProgramInput) (*mcp.CallToolResult, CreateProgramOutput, error) {
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "create-program")

	log.Info("Tool execution started",
		zap.String("name", input.Name),
		zap.String("package", input.Package),
		zap.Int("source_len", len(input.SourceCode)),
	)

	err := h.apiClient.CreateObject("PROG", input.Name, input.Description, input.SourceCode, input.Package)
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

// HandleCreateClass creates a new ABAP class via abaper-ts
func (h *Handlers) HandleCreateClass(ctx context.Context, req *mcp.CallToolRequest, input CreateClassInput) (*mcp.CallToolResult, CreateClassOutput, error) {
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "create-class")

	log.Info("Tool execution started",
		zap.String("name", input.Name),
		zap.String("package", input.Package),
		zap.Int("source_len", len(input.SourceCode)),
	)

	err := h.apiClient.CreateObject("CLAS", input.Name, input.Description, input.SourceCode, input.Package)
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

// HandleUpdateProgram updates an existing ABAP program via abaper-ts
func (h *Handlers) HandleUpdateProgram(ctx context.Context, req *mcp.CallToolRequest, input UpdateProgramInput) (*mcp.CallToolResult, UpdateProgramOutput, error) {
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "update-program")

	log.Info("Tool execution started",
		zap.String("name", input.Name),
		zap.Int("source_len", len(input.SourceCode)),
	)

	err := h.apiClient.UpdateObject("PROG", input.Name, input.SourceCode)
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

// HandleUpdateClass updates an existing ABAP class via abaper-ts
func (h *Handlers) HandleUpdateClass(ctx context.Context, req *mcp.CallToolRequest, input UpdateClassInput) (*mcp.CallToolResult, UpdateClassOutput, error) {
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "update-class")

	log.Info("Tool execution started",
		zap.String("name", input.Name),
		zap.Int("source_len", len(input.SourceCode)),
	)

	err := h.apiClient.UpdateObject("CLAS", input.Name, input.SourceCode)
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

// HandleActivateObject activates an ABAP object via abaper-ts
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

	adtType := normalizeObjectType(input.ObjectType)
	result, err := h.apiClient.Activate(adtType, input.ObjectName)
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

// HandleRunUnitTests runs ABAP unit tests via abaper-ts
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

	adtType := normalizeObjectType(input.ObjectType)
	result, err := h.apiClient.RunUnitTests(adtType, input.ObjectName)
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

// --- New tools ---

// SyntaxCheckInput defines input for syntax-check tool
type SyntaxCheckInput struct {
	ObjectType string `json:"object_type" jsonschema:"Type of ABAP object (program/class/include)"`
	ObjectName string `json:"object_name" jsonschema:"Name of the ABAP object"`
	SourceCode string `json:"source_code" jsonschema:"ABAP source code to check"`
}

// SyntaxCheckOutput defines output for syntax-check tool
type SyntaxCheckOutput struct {
	HasErrors bool   `json:"has_errors" jsonschema:"Whether syntax errors were found"`
	Messages  string `json:"messages" jsonschema:"Human-readable syntax check results"`
	Count     int    `json:"count" jsonschema:"Total number of messages"`
}

// HandleSyntaxCheck performs syntax check via abaper-ts
func (h *Handlers) HandleSyntaxCheck(ctx context.Context, req *mcp.CallToolRequest, input SyntaxCheckInput) (*mcp.CallToolResult, SyntaxCheckOutput, error) {
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "syntax-check")

	log.Info("Tool execution started",
		zap.String("object_type", input.ObjectType),
		zap.String("object_name", input.ObjectName),
	)

	adtType := normalizeObjectType(input.ObjectType)
	result, err := h.apiClient.SyntaxCheck(adtType, input.ObjectName, input.SourceCode)
	if err != nil {
		log.Error("Syntax check failed", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return &mcp.CallToolResult{IsError: true}, SyntaxCheckOutput{}, fmt.Errorf("syntax check failed: %w", err)
	}

	hasErrors := false
	var msgs strings.Builder
	for _, m := range result.Messages {
		if m.Severity == "error" {
			hasErrors = true
		}
		fmt.Fprintf(&msgs, "[%s] Line %d: %s\n", strings.ToUpper(m.Severity), m.Line, m.Text)
	}

	if len(result.Messages) == 0 {
		msgs.WriteString("No syntax errors found.\n")
	}

	log.Info("Tool execution completed",
		zap.Bool("has_errors", hasErrors),
		zap.Int("message_count", len(result.Messages)),
		zap.Duration("duration", time.Since(start)),
	)

	return nil, SyntaxCheckOutput{
		HasErrors: hasErrors,
		Messages:  msgs.String(),
		Count:     len(result.Messages),
	}, nil
}

// FormatCodeInput defines input for format-code tool
type FormatCodeInput struct {
	SourceCode string `json:"source_code" jsonschema:"ABAP source code to format"`
}

// FormatCodeOutput defines output for format-code tool
type FormatCodeOutput struct {
	FormattedCode string `json:"formatted_code" jsonschema:"Formatted ABAP source code"`
}

// HandleFormatCode formats ABAP source code via abaper-ts
func (h *Handlers) HandleFormatCode(ctx context.Context, req *mcp.CallToolRequest, input FormatCodeInput) (*mcp.CallToolResult, FormatCodeOutput, error) {
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "format-code")

	log.Info("Tool execution started",
		zap.Int("source_len", len(input.SourceCode)),
	)

	formatted, err := h.apiClient.FormatSource(input.SourceCode)
	if err != nil {
		log.Error("Format failed", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return &mcp.CallToolResult{IsError: true}, FormatCodeOutput{}, fmt.Errorf("format failed: %w", err)
	}

	log.Info("Tool execution completed",
		zap.Int("formatted_len", len(formatted)),
		zap.Duration("duration", time.Since(start)),
	)

	return nil, FormatCodeOutput{
		FormattedCode: formatted,
	}, nil
}

// TransportInfoInput defines input for transport-info tool
type TransportInfoInput struct {
	ObjectType string `json:"object_type" jsonschema:"Type of ABAP object"`
	ObjectName string `json:"object_name" jsonschema:"Name of the ABAP object"`
	Package    string `json:"package,omitempty" jsonschema:"Package name (optional)"`
}

// TransportInfoOutput defines output for transport-info tool
type TransportInfoOutput struct {
	Object     string `json:"object" jsonschema:"Object name"`
	Package    string `json:"package" jsonschema:"Package name"`
	Transports string `json:"transports" jsonschema:"Human-readable transport list"`
	Count      int    `json:"count" jsonschema:"Number of transports found"`
}

// HandleTransportInfo retrieves transport info via abaper-ts
func (h *Handlers) HandleTransportInfo(ctx context.Context, req *mcp.CallToolRequest, input TransportInfoInput) (*mcp.CallToolResult, TransportInfoOutput, error) {
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "transport-info")

	log.Info("Tool execution started",
		zap.String("object_type", input.ObjectType),
		zap.String("object_name", input.ObjectName),
	)

	adtType := normalizeObjectType(input.ObjectType)
	result, err := h.apiClient.TransportInfo(adtType, input.ObjectName, input.Package)
	if err != nil {
		log.Error("Transport info failed", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return &mcp.CallToolResult{IsError: true}, TransportInfoOutput{}, fmt.Errorf("transport info failed: %w", err)
	}

	var transports strings.Builder
	for _, t := range result.Transports {
		fmt.Fprintf(&transports, "%s - %s (Owner: %s, Status: %s)\n", t.Number, t.Description, t.Owner, t.Status)
	}
	if len(result.Transports) == 0 {
		transports.WriteString("No transports found.\n")
	}

	log.Info("Tool execution completed",
		zap.Int("transport_count", len(result.Transports)),
		zap.Duration("duration", time.Since(start)),
	)

	return nil, TransportInfoOutput{
		Object:     result.Object,
		Package:    result.Package,
		Transports: transports.String(),
		Count:      len(result.Transports),
	}, nil
}

// CreateTransportInput defines input for create-transport tool
type CreateTransportInput struct {
	ObjectType  string `json:"object_type" jsonschema:"Type of ABAP object"`
	ObjectName  string `json:"object_name" jsonschema:"Name of the ABAP object"`
	Description string `json:"description" jsonschema:"Transport description"`
	Package     string `json:"package,omitempty" jsonschema:"Package name (defaults to $TMP)"`
}

// CreateTransportOutput defines output for create-transport tool
type CreateTransportOutput struct {
	Success         bool   `json:"success" jsonschema:"Whether transport was created"`
	TransportNumber string `json:"transport_number" jsonschema:"Created transport number"`
	Description     string `json:"description" jsonschema:"Transport description"`
	Message         string `json:"message" jsonschema:"Result message"`
}

// HandleCreateTransport creates a transport request via abaper-ts
func (h *Handlers) HandleCreateTransport(ctx context.Context, req *mcp.CallToolRequest, input CreateTransportInput) (*mcp.CallToolResult, CreateTransportOutput, error) {
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "create-transport")

	log.Info("Tool execution started",
		zap.String("object_type", input.ObjectType),
		zap.String("object_name", input.ObjectName),
		zap.String("description", input.Description),
	)

	adtType := normalizeObjectType(input.ObjectType)
	pkg := input.Package
	if pkg == "" {
		pkg = "$TMP"
	}

	result, err := h.apiClient.CreateTransport(adtType, input.ObjectName, input.Description, pkg)
	if err != nil {
		log.Error("Create transport failed", zap.Error(err), zap.Duration("duration", time.Since(start)))
		return nil, CreateTransportOutput{
			Success: false,
			Message: fmt.Sprintf("Failed to create transport: %v", err),
		}, nil
	}

	log.Info("Tool execution completed",
		zap.String("transport_number", result.TransportNumber),
		zap.Duration("duration", time.Since(start)),
	)

	return nil, CreateTransportOutput{
		Success:         true,
		TransportNumber: result.TransportNumber,
		Description:     result.Description,
		Message:         fmt.Sprintf("Transport %s created successfully", result.TransportNumber),
	}, nil
}
