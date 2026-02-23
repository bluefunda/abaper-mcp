package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// APIClient wraps HTTP calls to the abaper-ts REST backend.
type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAPIClient creates a new API client for the given base URL.
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// apiResponse is the standard response envelope from abaper-ts.
type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error,omitempty"`
}

// post sends a JSON POST request and returns the data field from the response envelope.
func (c *APIClient) post(path string, body interface{}) (json.RawMessage, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+path, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response from %s: %w (status %d)", path, err, resp.StatusCode)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API error from %s: %s", path, apiResp.Error)
	}

	return apiResp.Data, nil
}

// --- Response types ---

// ObjectData is the response from /api/v1/objects/get
type ObjectData struct {
	ObjectName string `json:"object_name"`
	ObjectType string `json:"object_type"`
	Source     string `json:"source"`
	Etag       string `json:"etag"`
}

// SearchResult is the response from /api/v1/objects/search
type SearchResult struct {
	Objects []SearchObject `json:"Objects"`
}

// SearchObject is a single object in search results
type SearchObject struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Package     string `json:"package"`
}

// PackageData is a single package from /api/v1/objects/list
type PackageData struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ActivateData is the response from /api/v1/activate
type ActivateData struct {
	ObjectName string            `json:"object_name"`
	ObjectType string            `json:"object_type"`
	Success    bool              `json:"success"`
	Messages   []ActivateMessage `json:"messages"`
}

// ActivateMessage is a single activation message
type ActivateMessage struct {
	Severity string `json:"severity"`
	Text     string `json:"text"`
	Line     int    `json:"line"`
}

// UnitTestData is the response from /api/v1/unit-tests
type UnitTestData struct {
	ObjectName  string          `json:"object_name"`
	TotalTests  int             `json:"total_tests"`
	Passed      int             `json:"passed"`
	Failed      int             `json:"failed"`
	AllPassed   bool            `json:"all_passed"`
	TestClasses []TestClassData `json:"test_classes"`
}

// TestClassData is a single test class in unit test results
type TestClassData struct {
	Name    string           `json:"name"`
	Methods []TestMethodData `json:"methods"`
}

// TestMethodData is a single test method result
type TestMethodData struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// SyntaxCheckData is the response from /api/v1/syntax-check
type SyntaxCheckData struct {
	Messages []SyntaxMessage `json:"messages"`
}

// SyntaxMessage is a single syntax check message
type SyntaxMessage struct {
	Severity string `json:"severity"`
	Text     string `json:"text"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	EndLine  int    `json:"end_line"`
	EndCol   int    `json:"end_col"`
	Code     string `json:"code"`
}

// FormatData is the response from /api/v1/format
type FormatData struct {
	Source string `json:"source"`
}

// TransportInfoData is the response from /api/v1/transports/info
type TransportInfoData struct {
	Object     string          `json:"object"`
	Package    string          `json:"package"`
	Transports []TransportData `json:"transports"`
}

// TransportData is a single transport in transport info
type TransportData struct {
	Number      string `json:"number"`
	Description string `json:"description"`
	Owner       string `json:"owner"`
	Status      string `json:"status"`
	Type        string `json:"type"`
	Target      string `json:"target"`
	Date        string `json:"date"`
}

// CreateTransportData is the response from /api/v1/transports/create
type CreateTransportData struct {
	TransportNumber string `json:"transport_number"`
	Description     string `json:"description"`
	Package         string `json:"package"`
}

// ConnectData is the response from /api/v1/system/connect
type ConnectData struct {
	Authenticated bool   `json:"authenticated"`
	Message       string `json:"message"`
}

// --- API methods ---

// GetObject retrieves an ABAP object's source code.
func (c *APIClient) GetObject(objectType, objectName, functionGroup string) (*ObjectData, error) {
	body := map[string]string{
		"object_type": objectType,
		"object_name": objectName,
	}
	if functionGroup != "" {
		body["function_group"] = functionGroup
	}

	data, err := c.post("/api/v1/objects/get", body)
	if err != nil {
		return nil, err
	}

	var result ObjectData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse object data: %w", err)
	}
	return &result, nil
}

// SearchObjects searches for ABAP objects by pattern.
func (c *APIClient) SearchObjects(pattern string, objectTypes []string) (*SearchResult, error) {
	body := map[string]interface{}{
		"object_name": pattern,
	}
	if len(objectTypes) > 0 {
		body["object_type"] = objectTypes[0]
	}

	data, err := c.post("/api/v1/objects/search", body)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search result: %w", err)
	}
	return &result, nil
}

// ListPackages lists ABAP packages.
func (c *APIClient) ListPackages() ([]PackageData, error) {
	body := map[string]string{
		"object_type": "packages",
	}

	data, err := c.post("/api/v1/objects/list", body)
	if err != nil {
		return nil, err
	}

	var result []PackageData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse packages: %w", err)
	}
	return result, nil
}

// CreateObject creates a new ABAP object with source code.
func (c *APIClient) CreateObject(objectType, objectName, description, source, pkg string) error {
	body := map[string]string{
		"object_type": objectType,
		"object_name": objectName,
		"description": description,
		"source":      source,
		"package":     pkg,
	}

	_, err := c.post("/api/v1/objects/create", body)
	return err
}

// UpdateObject updates source code of an existing ABAP object (save mode).
func (c *APIClient) UpdateObject(objectType, objectName, source string) error {
	body := map[string]string{
		"object_type": objectType,
		"object_name": objectName,
		"source":      source,
	}

	_, err := c.post("/api/v1/objects/create", body)
	return err
}

// Activate activates an ABAP object.
func (c *APIClient) Activate(objectType, objectName string) (*ActivateData, error) {
	body := map[string]string{
		"object_type": objectType,
		"object_name": objectName,
	}

	data, err := c.post("/api/v1/activate", body)
	if err != nil {
		return nil, err
	}

	var result ActivateData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse activate result: %w", err)
	}
	return &result, nil
}

// RunUnitTests runs ABAP unit tests on an object.
func (c *APIClient) RunUnitTests(objectType, objectName string) (*UnitTestData, error) {
	body := map[string]string{
		"object_type": objectType,
		"object_name": objectName,
	}

	data, err := c.post("/api/v1/unit-tests", body)
	if err != nil {
		return nil, err
	}

	var result UnitTestData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse unit test result: %w", err)
	}
	return &result, nil
}

// TestConnection tests connectivity to the SAP system via abaper-ts.
func (c *APIClient) TestConnection() (*ConnectData, error) {
	data, err := c.post("/api/v1/system/connect", map[string]string{})
	if err != nil {
		return nil, err
	}

	var result ConnectData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse connect result: %w", err)
	}
	return &result, nil
}

// SyntaxCheck performs a syntax check on ABAP source code.
func (c *APIClient) SyntaxCheck(objectType, objectName, source string) (*SyntaxCheckData, error) {
	body := map[string]string{
		"object_type": objectType,
		"object_name": objectName,
		"source":      source,
	}

	data, err := c.post("/api/v1/syntax-check", body)
	if err != nil {
		return nil, err
	}

	var result SyntaxCheckData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse syntax check result: %w", err)
	}
	return &result, nil
}

// FormatSource formats ABAP source code via the pretty printer.
func (c *APIClient) FormatSource(source string) (string, error) {
	body := map[string]string{
		"source": source,
	}

	data, err := c.post("/api/v1/format", body)
	if err != nil {
		return "", err
	}

	var result FormatData
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("failed to parse format result: %w", err)
	}
	return result.Source, nil
}

// TransportInfo retrieves transport info for an ABAP object.
func (c *APIClient) TransportInfo(objectType, objectName, pkg string) (*TransportInfoData, error) {
	body := map[string]interface{}{
		"object_type": objectType,
		"object_name": objectName,
	}
	if pkg != "" {
		body["package"] = pkg
	}

	data, err := c.post("/api/v1/transports/info", body)
	if err != nil {
		return nil, err
	}

	var result TransportInfoData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse transport info: %w", err)
	}
	return &result, nil
}

// CreateTransport creates a new transport request.
func (c *APIClient) CreateTransport(objectType, objectName, description, pkg string) (*CreateTransportData, error) {
	body := map[string]string{
		"object_type": objectType,
		"object_name": objectName,
		"description": description,
		"package":     pkg,
	}

	data, err := c.post("/api/v1/transports/create", body)
	if err != nil {
		return nil, err
	}

	var result CreateTransportData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create transport result: %w", err)
	}
	return &result, nil
}

// normalizeObjectType maps user-friendly type names to ADT type codes.
func normalizeObjectType(input string) string {
	switch strings.ToLower(input) {
	case "program", "prog":
		return "PROG"
	case "class", "clas":
		return "CLAS"
	case "function", "func":
		return "FUNC"
	case "interface", "intf":
		return "INTF"
	case "table", "tabl":
		return "TABL"
	case "structure", "stru":
		return "STRU"
	case "include", "incl":
		return "INCL"
	case "function_group", "fugr":
		return "FUGR"
	default:
		return strings.ToUpper(input)
	}
}
