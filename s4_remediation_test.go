package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Sample ECC-style ABAP code that has S/4HANA compatibility issues
var sampleECCCode = `
REPORT ZSAMPLE_ECC_REPORT.

* Data declarations using SLIS (old ALV)
DATA: lt_fcat TYPE slis_t_fieldcat_alv,
      ls_fcat TYPE slis_fieldcat_alv,
      lt_data TYPE TABLE OF sflight,
      lv_bukrs TYPE bukrs VALUE '1000',
      lv_gjahr TYPE gjahr VALUE '2024',
      lt_bsis TYPE TABLE OF bsis.

START-OF-SELECTION.
  " Fetch data from old FI index table
  SELECT * FROM bsis
    INTO TABLE lt_bsis
    WHERE bukrs = lv_bukrs
      AND gjahr = lv_gjahr.

  " Build field catalog for ALV
  ls_fcat-fieldname = 'CARRID'.
  ls_fcat-seltext_l = 'Carrier'.
  APPEND ls_fcat TO lt_fcat.

  " Call old ALV function module
  CALL FUNCTION 'REUSE_ALV_LIST_DISPLAY'
    EXPORTING
      i_callback_program = sy-repid
    TABLES
      t_fieldcat = lt_fcat
      t_outtab   = lt_data.
`

func TestAnalyzeECCCode(t *testing.T) {
	// Load patterns
	patterns, err := loadPatterns()
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	// Analyze the sample ECC code
	issues := analyzeCode(sampleECCCode, patterns)

	// Build the result
	result := S4RemediationResult{
		RunMetadata: RunMetadata{
			RunID:         uuid.New().String(),
			TimestampUTC:  time.Now().UTC().Format(time.RFC3339),
			SystemID:      "https://a4h.bluefunda.com",
			SystemRelease: "",
			Client:        "001",
			Analyst:       "Claude",
		},
		Artifact: ArtifactInfo{
			ArtifactName:     "ZSAMPLE_ECC_REPORT",
			ArtifactType:     "program",
			Package:          "$TMP",
			TransportRequest: "",
		},
		Issues: issues,
	}

	// Output the result
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	fmt.Println("\n=== S/4HANA Remediation Analysis Output ===")
	fmt.Println(string(output))

	// Verify we found the expected issues
	if len(issues) == 0 {
		t.Error("Expected to find issues but found none")
	}

	// Check for expected patterns
	foundALV := false
	foundSLIS := false
	foundBSIS := false

	for _, issue := range issues {
		switch issue.PatternID {
		case "P001":
			if issue.SymptomDetected == "REUSE_ALV_LIST_DISPLAY used" {
				foundALV = true
			}
			if issue.SymptomDetected == "SLIS structures referenced" {
				foundSLIS = true
			}
		case "P003":
			foundBSIS = true
		}
	}

	if !foundALV {
		t.Error("Expected to find REUSE_ALV_LIST_DISPLAY issue")
	}
	if !foundSLIS {
		t.Error("Expected to find SLIS issue")
	}
	if !foundBSIS {
		t.Error("Expected to find BSIS issue")
	}
}
