package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bluefunda/abaper-mcp/internal/logger"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
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
	OutputFormat  string `json:"output_format,omitempty" jsonschema:"Output format: json (default) or markdown"`
}

// S4RemediationOutput is the combined output structure supporting both JSON and Markdown
type S4RemediationOutput struct {
	JSON     S4RemediationResult `json:"json" jsonschema:"Structured JSON result"`
	Markdown string              `json:"markdown" jsonschema:"Human-readable Markdown report"`
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

// generateMarkdownReport creates a human-readable Markdown report from the analysis result
func generateMarkdownReport(result S4RemediationResult) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# S/4HANA Remediation Analysis Report\n\n")

	// Run Metadata
	sb.WriteString("## Run Information\n\n")
	sb.WriteString(fmt.Sprintf("| Field | Value |\n"))
	sb.WriteString(fmt.Sprintf("|-------|-------|\n"))
	sb.WriteString(fmt.Sprintf("| **Run ID** | `%s` |\n", result.RunMetadata.RunID))
	sb.WriteString(fmt.Sprintf("| **Timestamp (UTC)** | %s |\n", result.RunMetadata.TimestampUTC))
	sb.WriteString(fmt.Sprintf("| **System ID** | %s |\n", result.RunMetadata.SystemID))
	sb.WriteString(fmt.Sprintf("| **Client** | %s |\n", result.RunMetadata.Client))
	sb.WriteString(fmt.Sprintf("| **Analyst** | %s |\n", result.RunMetadata.Analyst))
	sb.WriteString("\n")

	// Artifact Information
	sb.WriteString("## Analyzed Artifact\n\n")
	sb.WriteString(fmt.Sprintf("| Field | Value |\n"))
	sb.WriteString(fmt.Sprintf("|-------|-------|\n"))
	sb.WriteString(fmt.Sprintf("| **Name** | `%s` |\n", result.Artifact.ArtifactName))
	sb.WriteString(fmt.Sprintf("| **Type** | %s |\n", result.Artifact.ArtifactType))
	if result.Artifact.Package != "" {
		sb.WriteString(fmt.Sprintf("| **Package** | %s |\n", result.Artifact.Package))
	}
	if result.Artifact.TransportRequest != "" {
		sb.WriteString(fmt.Sprintf("| **Transport Request** | %s |\n", result.Artifact.TransportRequest))
	}
	sb.WriteString("\n")

	// Summary
	sb.WriteString("## Summary\n\n")
	if len(result.Issues) == 0 {
		sb.WriteString("**No S/4HANA compatibility issues detected.**\n\n")
		sb.WriteString("The analyzed code appears to be compatible with S/4HANA.\n\n")
	} else {
		// Count by severity
		highCount := 0
		mediumCount := 0
		for _, issue := range result.Issues {
			switch issue.Severity {
			case "High":
				highCount++
			case "Medium":
				mediumCount++
			}
		}
		sb.WriteString(fmt.Sprintf("**Total Issues Found: %d**\n\n", len(result.Issues)))
		sb.WriteString(fmt.Sprintf("- High Severity: %d\n", highCount))
		sb.WriteString(fmt.Sprintf("- Medium Severity: %d\n", mediumCount))
		sb.WriteString("\n")
	}

	// Issues Detail
	if len(result.Issues) > 0 {
		sb.WriteString("## Issues Detail\n\n")

		for i, issue := range result.Issues {
			severityBadge := "⚠️"
			if issue.Severity == "High" {
				severityBadge = "🔴"
			}

			sb.WriteString(fmt.Sprintf("### %d. %s %s [%s]\n\n", i+1, severityBadge, issue.PatternTitle, issue.PatternID))
			sb.WriteString(fmt.Sprintf("**Severity:** %s\n\n", issue.Severity))
			sb.WriteString(fmt.Sprintf("**Symptom Detected:** `%s`\n\n", issue.SymptomDetected))

			sb.WriteString("**Problematic Code:**\n")
			sb.WriteString("```abap\n")
			sb.WriteString(issue.BeforeCode)
			sb.WriteString("\n```\n\n")

			sb.WriteString("**Recommended Fix:**\n")
			sb.WriteString("```abap\n")
			sb.WriteString(issue.AfterCode)
			sb.WriteString("\n```\n\n")

			sb.WriteString(fmt.Sprintf("**Fix Description:** %s\n\n", issue.FixDescription))
			sb.WriteString("---\n\n")
		}
	}

	// Footer
	sb.WriteString("## Recommendations\n\n")
	if len(result.Issues) > 0 {
		sb.WriteString("1. Review each issue and apply the recommended fixes\n")
		sb.WriteString("2. Test thoroughly after making changes\n")
		sb.WriteString("3. Consider using SAP's Code Inspector (SCI) for additional validation\n")
		sb.WriteString("4. Consult SAP Note 2270689 for detailed S/4HANA simplification list\n")
	} else {
		sb.WriteString("- Continue monitoring for future S/4HANA compatibility requirements\n")
		sb.WriteString("- Consider running periodic analysis as new patterns are added\n")
	}
	sb.WriteString("\n")

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("*Generated by ABAPER MCP - %s*\n", result.RunMetadata.TimestampUTC))

	return sb.String()
}

// HandleAnalyzeS4Remediation analyzes ABAP code for S/4HANA compatibility issues
func (h *Handlers) HandleAnalyzeS4Remediation(ctx context.Context, req *mcp.CallToolRequest, input AnalyzeS4RemediationInput) (*mcp.CallToolResult, S4RemediationOutput, error) {
	requestID := uuid.New().String()[:8]
	start := time.Now()
	log := logger.WithTool(requestID, "analyze-s4-remediation")

	log.Info("Tool execution started",
		zap.String("object_type", input.ObjectType),
		zap.String("object_name", input.ObjectName),
		zap.String("output_format", input.OutputFormat),
	)

	objectType := strings.ToLower(input.ObjectType)

	// Validate function group requirement
	if (objectType == "function" || objectType == "func") && input.FunctionGroup == "" {
		log.Warn("Validation failed: function_group required")
		return &mcp.CallToolResult{IsError: true}, S4RemediationOutput{}, fmt.Errorf("function_group is required for function modules")
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
		log.Warn("Validation failed: unsupported object type", zap.String("object_type", input.ObjectType))
		return &mcp.CallToolResult{IsError: true}, S4RemediationOutput{}, fmt.Errorf("unsupported object type: %s", input.ObjectType)
	}

	// Get fresh client
	client, err := h.getClient()
	if err != nil {
		log.Error("Failed to get ADT client", zap.Error(err))
		return &mcp.CallToolResult{IsError: true}, S4RemediationOutput{}, fmt.Errorf("failed to connect: %w", err)
	}

	// Fetch the source code
	var sourceCode string
	switch objectType {
	case "program", "prog":
		source, err := client.GetProgram(input.ObjectName)
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, S4RemediationOutput{}, fmt.Errorf("failed to get program: %w", err)
		}
		sourceCode = source.Source
	case "class", "clas":
		source, err := client.GetClass(input.ObjectName)
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, S4RemediationOutput{}, fmt.Errorf("failed to get class: %w", err)
		}
		sourceCode = source.Source
	case "function", "func":
		source, err := client.GetFunction(input.ObjectName, input.FunctionGroup)
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, S4RemediationOutput{}, fmt.Errorf("failed to get function: %w", err)
		}
		sourceCode = source.Source
	case "interface", "intf":
		source, err := client.GetInterface(input.ObjectName)
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, S4RemediationOutput{}, fmt.Errorf("failed to get interface: %w", err)
		}
		sourceCode = source.Source
	case "include", "incl":
		source, err := client.GetInclude(input.ObjectName)
		if err != nil {
			return &mcp.CallToolResult{IsError: true}, S4RemediationOutput{}, fmt.Errorf("failed to get include: %w", err)
		}
		sourceCode = source.Source
	}

	// Load patterns
	patterns, err := loadPatterns()
	if err != nil {
		return &mcp.CallToolResult{IsError: true}, S4RemediationOutput{}, err
	}

	// Analyze the code
	issues := analyzeCode(sourceCode, patterns)

	// Ensure issues is never nil (JSON schema requires array, not null)
	if issues == nil {
		issues = []S4RemediationIssue{}
	}

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

	// Generate markdown report
	markdownReport := generateMarkdownReport(result)

	// Return combined output with both JSON and Markdown
	output := S4RemediationOutput{
		JSON:     result,
		Markdown: markdownReport,
	}

	log.Info("Tool execution completed",
		zap.Int("issues_found", len(issues)),
		zap.Duration("duration", time.Since(start)),
	)

	return nil, output, nil
}
