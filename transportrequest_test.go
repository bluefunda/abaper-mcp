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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// sampleTransportRequest is the C-AI5 JSON abaper serves for
// GET /api/v1/transports/{trkey}.
var sampleTransportRequest = map[string]any{
	"number": "DEVK900123", "description": "Sales order pricing fix", "owner": "JDOE",
	"status": "D", "type": "K", "target": "QAS",
	"tasks": []map[string]any{
		{"number": "DEVK900124", "owner": "JDOE", "description": "Pricing class changes", "status": "D",
			"objects": []map[string]any{
				{"pgmid": "R3TR", "type": "CLAS", "name": "ZCL_SD_PRICING", "description": "SD pricing helper"},
				{"pgmid": "R3TR", "type": "PROG", "name": "ZSD_PRICING_REPORT"},
			}},
		{"number": "DEVK900125", "owner": "ASMITH", "description": "DDIC changes", "status": "R",
			"objects": []map[string]any{
				{"pgmid": "R3TR", "type": "TABL", "name": "ZSD_PRICE_LOG"},
				{"pgmid": "LIMU", "type": "TABD", "name": "ZSD_PRICE_LOG"},
			}},
	},
}

func TestGetTransportRequestTool(t *testing.T) {
	var gotMethod, gotPath string
	session := newTestMCPSession(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		envelopeHandler(t, true, sampleTransportRequest, "")(w, r)
	})

	out, res := callTool[TransportRequestData](t, session, getTransportRequestTool, map[string]any{"trkey": " devk900123 "})
	if res.IsError {
		t.Fatalf("unexpected tool error: %+v", res.Content)
	}
	if gotMethod != http.MethodGet || gotPath != "/api/v1/transports/DEVK900123" {
		t.Errorf("backend call = %s %s, want GET /api/v1/transports/DEVK900123", gotMethod, gotPath)
	}
	if out.Number != "DEVK900123" || out.Target != "QAS" || len(out.Tasks) != 2 {
		t.Fatalf("unexpected result: %+v", out)
	}
	if o := out.Tasks[1].Objects[1]; o.PgmID != "LIMU" || o.Type != "TABD" || o.Name != "ZSD_PRICE_LOG" {
		t.Errorf("unexpected object: %+v", o)
	}

	// The structured output is the C-AI5 JSON, key for key.
	raw, _ := json.Marshal(res.StructuredContent)
	for _, k := range []string{`"number"`, `"description"`, `"owner"`, `"status"`, `"type"`, `"target"`, `"tasks"`, `"pgmid"`, `"objects"`} {
		if !strings.Contains(string(raw), k) {
			t.Errorf("structured output missing %s: %s", k, raw)
		}
	}
}

func TestGetTransportRequestTool_IsAnnotatedReadOnly(t *testing.T) {
	session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{}, ""))
	tools, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		if tool.Name != getTransportRequestTool {
			continue
		}
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("want ReadOnlyHint, got %+v", tool.Annotations)
		}
		if !strings.HasPrefix(tool.Description, "Read-only.") {
			t.Errorf("description must say read-only: %q", tool.Description)
		}
		return
	}
	t.Fatalf("tool %s not registered", getTransportRequestTool)
}

func TestGetTransportRequestTool_InvalidTRKey(t *testing.T) {
	for _, key := range []string{"", "DEVK12345", "DEVX900123", "DEVK900123/../../objects/get", "1EVK900123"} {
		called := false
		session := newTestMCPSession(t, func(w http.ResponseWriter, r *http.Request) {
			called = true
			envelopeHandler(t, true, sampleTransportRequest, "")(w, r)
		})
		_, res := callTool[TransportRequestData](t, session, getTransportRequestTool, map[string]any{"trkey": key})
		if !res.IsError {
			t.Errorf("%q: expected IsError", key)
		}
		if called {
			t.Errorf("%q: backend must not be called for an invalid trkey", key)
		}
	}
}

func TestGetTransportRequestTool_NotFound(t *testing.T) {
	session := newTestMCPSession(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "transport request DEVK999999 not found"})
	})
	_, res := callTool[TransportRequestData](t, session, getTransportRequestTool, map[string]any{"trkey": "DEVK999999"})
	if !res.IsError {
		t.Fatal("expected IsError")
	}
	if text := toolText(res); !strings.Contains(text, "DEVK999999 not found") {
		t.Errorf("error text = %q", text)
	}
}

func TestGetTransportRequestTool_EmptyListsNotNull(t *testing.T) {
	session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{
		"number": "DEVK900200", "owner": "JDOE", "status": "D", "type": "K",
		"tasks": []map[string]any{{"number": "DEVK900201", "owner": "JDOE"}},
	}, ""))
	_, res := callTool[TransportRequestData](t, session, getTransportRequestTool, map[string]any{"trkey": "DEVK900200"})
	if res.IsError {
		t.Fatalf("unexpected tool error: %+v", res.Content)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	if strings.Contains(string(raw), "null") {
		t.Errorf("structured output has null lists: %s", raw)
	}
}

// TestGetCarriesSameIdentityAsPost keeps get's per-user SAP identity headers
// identical to post's, so the read-only GET can't drift from the path every
// other tool uses.
func TestGetCarriesSameIdentityAsPost(t *testing.T) {
	identity := []string{"X-Realm", "Authorization", "X-SAP-Host", "X-SAP-Client", "X-SAP-User", "X-SAP-Password"}
	seen := map[string]http.Header{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.Method] = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{}}`))
	}))
	defer srv.Close()

	client := NewAPIClientForUser(srv.URL, &SAPCredentials{
		Host: "https://sap.example.com", Client: "100", Username: "alice", Password: "pw",
	}, "individual", "the-users-jwt")
	if _, err := client.post(context.Background(), "/api/v1/system/connect", map[string]string{}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.get(context.Background(), "/api/v1/transports/DEVK900123"); err != nil {
		t.Fatal(err)
	}
	for _, h := range identity {
		p, g := seen[http.MethodPost].Get(h), seen[http.MethodGet].Get(h)
		if p == "" || p != g {
			t.Errorf("header %s: differs between post and get or is empty (post set=%v, get set=%v)", h, p != "", g != "")
		}
	}

	// No per-user credentials: no identity headers on either.
	seen = map[string]http.Header{}
	anon := NewAPIClient(srv.URL)
	_, _ = anon.get(context.Background(), "/api/v1/transports/DEVK900123")
	for _, h := range identity {
		if v := seen[http.MethodGet].Get(h); v != "" {
			t.Errorf("anonymous client sent %s", h)
		}
	}
}

func TestGetWithCredentialsErr_NeverMakesARequest(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer srv.Close()

	wantErr := errors.New("SAP is not connected")
	client := NewAPIClientWithError(srv.URL, wantErr)
	if _, err := client.GetTransportRequest(context.Background(), "DEVK900123"); !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if called {
		t.Error("get must not reach the backend when credentialsErr is set")
	}
}

func toolText(res *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}
