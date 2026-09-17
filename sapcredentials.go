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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SAPCredentials is one connected user's own SAP system identity, fetched
// from cai-bff and attached to every abaper backend call for that user's
// session (see bluefunda/cai-bff#160's /internal/sap/credentials).
type SAPCredentials struct {
	Host     string `json:"host"`
	Client   string `json:"client"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// ErrSAPNotConnected is returned when cai-bff has no stored SAP credentials
// for the calling user — distinct from a transport/lookup failure, so
// callers can surface a clear "connect ABAPer first" message instead of a
// generic error.
var ErrSAPNotConnected = errors.New("SAP is not connected for this user")

// fetchSAPCredentials calls cai-bff's internal credential-lookup endpoint for
// userID. Returns ErrSAPNotConnected (not a generic error) when the user
// simply hasn't connected SAP yet, so the caller can tell that apart from a
// real backend/network failure.
func fetchSAPCredentials(ctx context.Context, baseURL, internalSecret, userID string) (*SAPCredentials, error) {
	url := strings.TrimRight(baseURL, "/") + "/internal/sap/credentials?userID=" + userID

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build sap credentials request: %w", err)
	}
	if internalSecret != "" {
		req.Header.Set("X-Internal-Secret", internalSecret)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sap credentials request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrSAPNotConnected
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read sap credentials response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sap credentials lookup returned %d: %s", resp.StatusCode, string(body))
	}

	var creds SAPCredentials
	if err := json.Unmarshal(body, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse sap credentials response: %w", err)
	}
	return &creds, nil
}
