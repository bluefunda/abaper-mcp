// Copyright 2025 bluefunda
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// getTransportRequestTool is the read-only transport contents tool used by the
// BlueRequests risk agent (C-AI5) and available to chat.
const getTransportRequestTool = "get_transport_request"

// transportNumberRe is the SAP transport request/task number convention:
// <SID>K<6 digits>, the SID starting with a letter (e.g. DEVK900123).
var transportNumberRe = regexp.MustCompile(`^[A-Z][A-Z0-9]{2}K[0-9]{6}$`)

// normalizeTransportNumber trims and upper-cases trkey and checks its format.
func normalizeTransportNumber(trkey string) (string, error) {
	n := strings.ToUpper(strings.TrimSpace(trkey))
	if !transportNumberRe.MatchString(n) {
		return "", fmt.Errorf("invalid transport number %q: expected <SID>K<6 digits>, e.g. DEVK900123", trkey)
	}
	return n, nil
}

// TransportRequestData is the response from GET /api/v1/transports/{trkey}
// and the output of get_transport_request (C-AI5 JSON).
type TransportRequestData struct {
	Number      string                `json:"number" jsonschema:"Transport request number"`
	Description string                `json:"description" jsonschema:"Request description"`
	Owner       string                `json:"owner" jsonschema:"Request owner (SAP user)"`
	Status      string                `json:"status" jsonschema:"Status code: D modifiable, R released, L locked, O release started"`
	Type        string                `json:"type" jsonschema:"Request type: K workbench, W customizing, T transport of copies"`
	Target      string                `json:"target" jsonschema:"Target system"`
	Tasks       []TransportTaskData   `json:"tasks" jsonschema:"Tasks of the request with their objects"`
	Objects     []TransportObjectData `json:"objects,omitempty" jsonschema:"Objects recorded on the request itself (moved there when a task is released)"`
}

// TransportTaskData is one task of a transport request.
type TransportTaskData struct {
	Number      string                `json:"number" jsonschema:"Task number"`
	Owner       string                `json:"owner" jsonschema:"Task owner (SAP user)"`
	Description string                `json:"description" jsonschema:"Task description"`
	Status      string                `json:"status" jsonschema:"Status code: D modifiable, R released"`
	Objects     []TransportObjectData `json:"objects" jsonschema:"Objects recorded in the task"`
}

// TransportObjectData is one object entry of a request or task.
type TransportObjectData struct {
	PgmID       string `json:"pgmid" jsonschema:"Program ID, e.g. R3TR or LIMU"`
	Type        string `json:"type" jsonschema:"Object type, e.g. CLAS, PROG, TABL, METH"`
	Name        string `json:"name" jsonschema:"Object name"`
	Description string `json:"description,omitempty" jsonschema:"Object description, if SAP returns one"`
}

// GetTransportRequest reads a transport request with its tasks and objects
// (read-only) using this client's SAP identity.
func (c *APIClient) GetTransportRequest(ctx context.Context, trkey string) (*TransportRequestData, error) {
	number, err := normalizeTransportNumber(trkey)
	if err != nil {
		return nil, err
	}
	data, err := c.get(ctx, "/api/v1/transports/"+url.PathEscape(number))
	if err != nil {
		return nil, err
	}
	var result TransportRequestData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse transport request: %w", err)
	}
	return &result, nil
}

// get sends a GET request and returns the data field from the response
// envelope. It carries the same SAP identity headers as post and, like post,
// never reaches the backend when credentialsErr is set.
func (c *APIClient) get(ctx context.Context, path string) (json.RawMessage, error) {
	if c.credentialsErr != nil {
		return nil, c.credentialsErr
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to build request to %s: %w", path, err)
	}
	req.Header.Set("Accept", "application/json")
	c.setSAPIdentityHeaders(req)

	resp, err := c.httpClient.Do(req)
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
		return nil, &APIError{Path: path, StatusCode: resp.StatusCode, Message: apiResp.Error}
	}
	return apiResp.Data, nil
}

// setSAPIdentityHeaders attaches the per-user SAP identity exactly as post
// does (X-Realm, Authorization, X-SAP-*). TestGetCarriesSameIdentityAsPost
// keeps the two in step.
func (c *APIClient) setSAPIdentityHeaders(req *http.Request) {
	if c.sapCreds == nil {
		return
	}
	if c.realm != "" {
		req.Header.Set("X-Realm", c.realm)
	}
	if c.principalToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.principalToken)
	}
	req.Header.Set("X-SAP-Host", c.sapCreds.Host)
	req.Header.Set("X-SAP-Client", c.sapCreds.Client)
	req.Header.Set("X-SAP-User", c.sapCreds.Username)
	req.Header.Set("X-SAP-Password", c.sapCreds.Password)
}

// GetTransportRequestInput defines input for get_transport_request.
type GetTransportRequestInput struct {
	TRKey string `json:"trkey" jsonschema:"SAP transport request number, e.g. DEVK900123"`
}

func registerTransportRequestTool(server *mcp.Server, handlers *Handlers) {
	mcp.AddTool(server, &mcp.Tool{
		Name: getTransportRequestTool,
		Description: "Read-only. Get the contents of an SAP transport request: number, description, owner, status, " +
			"type, target system, its tasks (number, owner, description, status) and the objects in each task " +
			"(pgmid, type, name). Uses the signed-in user's own SAP connection. Changes nothing in SAP.",
		Annotations: &mcp.ToolAnnotations{
			Title:          "Get transport request (read-only)",
			ReadOnlyHint:   true,
			IdempotentHint: true,
		},
	}, handlers.HandleGetTransportRequest)
}

// HandleGetTransportRequest returns a transport request's contents via abaper.
func (h *Handlers) HandleGetTransportRequest(ctx context.Context, req *mcp.CallToolRequest, input GetTransportRequestInput) (*mcp.CallToolResult, TransportRequestData, error) {
	log, start := h.newToolLogger(getTransportRequestTool)
	log.Info("Tool execution started", zap.String("trkey", input.TRKey))

	trkey, err := normalizeTransportNumber(input.TRKey)
	if err != nil {
		log.Warn("Validation failed", zap.Error(err))
		return &mcp.CallToolResult{IsError: true}, TransportRequestData{}, err
	}

	result, err := h.apiClient.GetTransportRequest(ctx, trkey)
	if err != nil {
		log.Error("Get transport request failed", zap.Error(err), zap.Duration("duration", time.Since(start)))
		if IsNotFound(err) {
			return &mcp.CallToolResult{IsError: true}, TransportRequestData{}, fmt.Errorf("transport request %s not found", trkey)
		}
		return &mcp.CallToolResult{IsError: true}, TransportRequestData{}, fmt.Errorf("get transport request failed: %w", err)
	}

	// Empty lists, not null, so the output matches its schema.
	if result.Tasks == nil {
		result.Tasks = []TransportTaskData{}
	}
	for i := range result.Tasks {
		if result.Tasks[i].Objects == nil {
			result.Tasks[i].Objects = []TransportObjectData{}
		}
	}

	log.Info("Tool execution completed",
		zap.String("trkey", trkey),
		zap.Int("tasks", len(result.Tasks)),
		zap.Duration("duration", time.Since(start)),
	)
	return nil, *result, nil
}
