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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// S4Client wraps HTTP calls to the s4-temporal REST API.
type S4Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewS4Client creates a new client for the s4-temporal API.
func NewS4Client(baseURL string) *S4Client {
	return &S4Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// --- Request/Response types ---

// S4RunRequest is the request body for /scripts/run.
type S4RunRequest struct {
	Script string `json:"script"`
	Bucket string `json:"bucket,omitempty"`
	Prefix string `json:"prefix,omitempty"`
	Ext    string `json:"ext,omitempty"`
}

// S4RunResponse is the response from /scripts/run.
type S4RunResponse struct {
	WorkflowID string `json:"workflowId"`
	RunID      string `json:"runId"`
}

// S4StatusResponse is the response from /scripts/status/{id}.
type S4StatusResponse struct {
	WorkflowID string `json:"workflowId"`
	Status     string `json:"status"`
}

// S4ResultResponse is the response from /scripts/result/{id}.
type S4ResultResponse struct {
	WorkflowID string `json:"workflowId"`
	Result     string `json:"result"`
}

// --- Client methods ---

// RunScript triggers a script execution via Temporal workflow.
func (c *S4Client) RunScript(ctx context.Context, req S4RunRequest) (*S4RunResponse, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/scripts/run", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to build request to /scripts/run: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request to /scripts/run failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("s4-temporal error (status %d): %s", resp.StatusCode, string(body))
	}

	var result S4RunResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetStatus checks the status of a Temporal workflow.
func (c *S4Client) GetStatus(ctx context.Context, workflowID string) (*S4StatusResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/scripts/status/"+workflowID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request to /scripts/status: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to /scripts/status failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("s4-temporal error (status %d): %s", resp.StatusCode, string(body))
	}

	var result S4StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetResult retrieves the result of a completed Temporal workflow.
func (c *S4Client) GetResult(ctx context.Context, workflowID string) (*S4ResultResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/scripts/result/"+workflowID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request to /scripts/result: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to /scripts/result failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("s4-temporal error (status %d): %s", resp.StatusCode, string(body))
	}

	var result S4ResultResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}
