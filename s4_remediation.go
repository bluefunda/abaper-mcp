package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed s4_remediation_patterns.json
var s4RemediationPatternsJSON []byte

// S4RemediationPatterns holds the loaded patterns
type S4RemediationPatterns struct {
	RemediationPatterns []S4Pattern `json:"remediation_patterns"`
}

// S4Pattern represents a single remediation pattern
type S4Pattern struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Category   string   `json:"category"`
	Severity   string   `json:"severity,omitempty"`
	Symptoms   []string `json:"symptoms"`
	Reason     string   `json:"reason"`
	Fix        string   `json:"fix"`
	BeforeCode string   `json:"before_code"`
	AfterCode  string   `json:"after_code"`
}

// RunMetadata contains information about the analysis run
type RunMetadata struct {
	RunID         string `json:"run_id"`
	TimestampUTC  string `json:"timestamp_utc"`
	SystemID      string `json:"system_id"`
	SystemRelease string `json:"system_release"`
	Client        string `json:"client"`
	Analyst       string `json:"analyst"`
}

// ArtifactInfo contains information about the analyzed artifact
type ArtifactInfo struct {
	ArtifactName     string `json:"artifact_name"`
	ArtifactType     string `json:"artifact_type"`
	Package          string `json:"package"`
	TransportRequest string `json:"transport_request"`
}

// S4RemediationResult is the JSON output structure
type S4RemediationResult struct {
	RunMetadata RunMetadata          `json:"run_metadata"`
	Artifact    ArtifactInfo         `json:"artifact"`
	Issues      []S4RemediationIssue `json:"issues"`
}

// S4RemediationIssue represents a single detected issue
type S4RemediationIssue struct {
	PatternID       string `json:"pattern_id"`
	PatternTitle    string `json:"pattern_title"`
	SymptomDetected string `json:"symptom_detected"`
	Severity        string `json:"severity"`
	BeforeCode      string `json:"before_code"`
	AfterCode       string `json:"after_code"`
	FixDescription  string `json:"fix_description"`
}

// AnalyzeS4RemediationInput defines input for analyze-s4-remediation tool
type AnalyzeS4RemediationInput struct {
	ObjectType    string `json:"object_type" jsonschema:"Type of ABAP object (program/class/function/interface)"`
	ObjectName    string `json:"object_name" jsonschema:"Name of the ABAP object to analyze"`
	FunctionGroup string `json:"function_group,omitempty" jsonschema:"Function group name (required for function modules)"`
}

// loadPatterns loads the embedded remediation patterns
func loadPatterns() (*S4RemediationPatterns, error) {
	var patterns S4RemediationPatterns
	if err := json.Unmarshal(s4RemediationPatternsJSON, &patterns); err != nil {
		return nil, fmt.Errorf("failed to parse remediation patterns: %w", err)
	}
	return &patterns, nil
}

// getSeverityForCategory returns severity based on category
func getSeverityForCategory(category string) string {
	switch category {
	case "DB Model Change":
		return "High"
	case "Obsolete Object":
		return "High"
	default:
		return "Medium"
	}
}

// analyzeCode checks ABAP code against S/4HANA remediation patterns
func analyzeCode(sourceCode string, patterns *S4RemediationPatterns) []S4RemediationIssue {
	var issues []S4RemediationIssue
	upperCode := strings.ToUpper(sourceCode)

	for _, pattern := range patterns.RemediationPatterns {
		for _, symptom := range pattern.Symptoms {
			if matchesSymptom(upperCode, symptom) {
				// Extract the actual matching code snippet
				matchedCode := extractMatchingCode(sourceCode, symptom)

				// Use pattern severity if set, otherwise derive from category
				severity := pattern.Severity
				if severity == "" {
					severity = getSeverityForCategory(pattern.Category)
				}

				issues = append(issues, S4RemediationIssue{
					PatternID:       pattern.ID,
					PatternTitle:    pattern.Title,
					SymptomDetected: symptom,
					Severity:        severity,
					BeforeCode:      matchedCode,
					AfterCode:       pattern.AfterCode,
					FixDescription:  fmt.Sprintf("%s - %s", pattern.Reason, pattern.Fix),
				})
			}
		}
	}

	return issues
}

// matchesSymptom checks if the code contains the symptom pattern
func matchesSymptom(upperCode, symptom string) bool {
	symptomUpper := strings.ToUpper(symptom)

	// Handle different symptom patterns
	switch {
	case strings.Contains(symptomUpper, "USED"):
		// e.g., "REUSE_ALV_LIST_DISPLAY used"
		parts := strings.Split(symptomUpper, " ")
		if len(parts) > 0 {
			return strings.Contains(upperCode, parts[0])
		}
	case strings.Contains(symptomUpper, "REFERENCED"):
		// e.g., "SLIS structures referenced"
		parts := strings.Split(symptomUpper, " ")
		if len(parts) > 0 {
			return strings.Contains(upperCode, parts[0])
		}
	case strings.Contains(symptomUpper, "SELECT ON"):
		// e.g., "SELECT on VBDATA"
		parts := strings.Split(symptomUpper, " ON ")
		if len(parts) > 1 {
			tableName := strings.TrimSpace(parts[1])
			// Match SELECT ... FROM tablename pattern
			pattern := fmt.Sprintf(`SELECT\s+.*\s+FROM\s+%s`, regexp.QuoteMeta(tableName))
			matched, _ := regexp.MatchString(pattern, upperCode)
			return matched
		}
	case strings.Contains(symptomUpper, "SELECT FROM"):
		// e.g., "SELECT from BSIS/BSAS"
		parts := strings.Split(symptomUpper, " FROM ")
		if len(parts) > 1 {
			tables := strings.Split(parts[1], "/")
			for _, table := range tables {
				tableName := strings.TrimSpace(table)
				pattern := fmt.Sprintf(`SELECT\s+.*\s+FROM\s+%s`, regexp.QuoteMeta(tableName))
				matched, _ := regexp.MatchString(pattern, upperCode)
				if matched {
					return true
				}
			}
		}
	default:
		// Direct string match
		return strings.Contains(upperCode, symptomUpper)
	}

	return false
}

// extractMatchingCode extracts the relevant code snippet that matches the symptom
func extractMatchingCode(sourceCode, symptom string) string {
	upperCode := strings.ToUpper(sourceCode)
	symptomUpper := strings.ToUpper(symptom)
	lines := strings.Split(sourceCode, "\n")

	// Find the keyword to search for
	var keyword string
	switch {
	case strings.Contains(symptomUpper, "REUSE_ALV"):
		keyword = "REUSE_ALV"
	case strings.Contains(symptomUpper, "SLIS"):
		keyword = "SLIS"
	case strings.Contains(symptomUpper, "VBDATA"):
		keyword = "VBDATA"
	case strings.Contains(symptomUpper, "BSEG"):
		keyword = "BSEG"
	case strings.Contains(symptomUpper, "MATDOC"):
		keyword = "MATDOC"
	case strings.Contains(symptomUpper, "BSIS"):
		keyword = "BSIS"
	case strings.Contains(symptomUpper, "BSAS"):
		keyword = "BSAS"
	default:
		// Extract first word from symptom
		parts := strings.Fields(symptomUpper)
		if len(parts) > 0 {
			keyword = parts[0]
		}
	}

	if keyword == "" {
		return ""
	}

	// Find lines containing the keyword
	var matchedLines []string
	for i, line := range lines {
		if strings.Contains(strings.ToUpper(line), keyword) {
			// Include context: 1 line before and after
			start := i
			if i > 0 {
				start = i - 1
			}
			end := i + 2
			if end > len(lines) {
				end = len(lines)
			}

			for j := start; j < end; j++ {
				trimmedLine := strings.TrimSpace(lines[j])
				if trimmedLine != "" && !containsString(matchedLines, trimmedLine) {
					matchedLines = append(matchedLines, trimmedLine)
				}
			}
		}
	}

	// Also check for SELECT statements with table names
	if strings.Contains(symptomUpper, "SELECT") {
		selectRegex := regexp.MustCompile(`(?i)SELECT\s+.*\s+FROM\s+\w+`)
		matches := selectRegex.FindAllString(sourceCode, -1)
		for _, match := range matches {
			if strings.Contains(strings.ToUpper(match), keyword) {
				if !containsString(matchedLines, strings.TrimSpace(match)) {
					matchedLines = append(matchedLines, strings.TrimSpace(match))
				}
			}
		}
	}

	_ = upperCode // suppress unused variable warning

	if len(matchedLines) == 0 {
		return "(code pattern detected but specific line not extracted)"
	}

	return strings.Join(matchedLines, "\n")
}

// containsString checks if a slice contains a string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// HandleAnalyzeS4Remediation analyzes ABAP code for S/4HANA compatibility issues
func (h *Handlers) HandleAnalyzeS4Remediation(ctx context.Context, req *mcp.CallToolRequest, input AnalyzeS4RemediationInput) (*mcp.CallToolResult, S4RemediationResult, error) {
	objectType := strings.ToLower(input.ObjectType)

	// Validate function group requirement
	if (objectType == "function" || objectType == "func") && input.FunctionGroup == "" {
		return &mcp.CallToolResult{IsError: true}, S4RemediationResult{}, fmt.Errorf("function_group is required for function modules")
	}

	// Validate object type
	validTypes := map[string]bool{
		"program": true, "prog": true,
		"class": true, "clas": true,
		"function": true, "func": true,
		"interface": true, "intf": true,
		"include": true, "incl": true,
	}
	if !validTypes[objectType] {
		return &mcp.CallToolResult{IsError: true}, S4RemediationResult{}, fmt.Errorf("unsupported object type: %s", input.ObjectType)
	}

	// Get fresh client
	client, err := h.getClient()
	if err != nil {
		return &mcp.CallToolResult{IsError: true}, S4RemediationResult{}, fmt.Errorf("failed to connect: %w", err)
	}

	// Fetch the source code
	var sourceCode string
	switch objectType {
	case "program", "prog":
		source, err := client.GetProgram(input.ObjectName)
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, S4RemediationResult{}, fmt.Errorf("failed to get program: %w", err)
		}
		sourceCode = source.Source
	case "class", "clas":
		source, err := client.GetClass(input.ObjectName)
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, S4RemediationResult{}, fmt.Errorf("failed to get class: %w", err)
		}
		sourceCode = source.Source
	case "function", "func":
		source, err := client.GetFunction(input.ObjectName, input.FunctionGroup)
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, S4RemediationResult{}, fmt.Errorf("failed to get function: %w", err)
		}
		sourceCode = source.Source
	case "interface", "intf":
		source, err := client.GetInterface(input.ObjectName)
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, S4RemediationResult{}, fmt.Errorf("failed to get interface: %w", err)
		}
		sourceCode = source.Source
	case "include", "incl":
		source, err := client.GetInclude(input.ObjectName)
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, S4RemediationResult{}, fmt.Errorf("failed to get include: %w", err)
		}
		sourceCode = source.Source
	}

	// Load patterns
	patterns, err := loadPatterns()
	if err != nil {
		return &mcp.CallToolResult{IsError: true}, S4RemediationResult{}, err
	}

	// Analyze the code
	issues := analyzeCode(sourceCode, patterns)

	// Determine artifact type display name
	artifactTypeDisplay := objectType
	switch objectType {
	case "prog":
		artifactTypeDisplay = "program"
	case "clas":
		artifactTypeDisplay = "class"
	case "func":
		artifactTypeDisplay = "function"
	case "intf":
		artifactTypeDisplay = "interface"
	case "incl":
		artifactTypeDisplay = "include"
	}

	result := S4RemediationResult{
		RunMetadata: RunMetadata{
			RunID:         uuid.New().String(),
			TimestampUTC:  time.Now().UTC().Format(time.RFC3339),
			SystemID:      h.clientManager.config.ADTHost,
			SystemRelease: "",
			Client:        h.clientManager.config.ADTClient,
			Analyst:       "Claude",
		},
		Artifact: ArtifactInfo{
			ArtifactName:     input.ObjectName,
			ArtifactType:     artifactTypeDisplay,
			Package:          "",
			TransportRequest: "",
		},
		Issues: issues,
	}

	return nil, result, nil
}
