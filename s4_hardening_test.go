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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestS4Handlers returns Handlers whose s4Client points at backend.
func newTestS4Handlers(t *testing.T, backend http.HandlerFunc) *Handlers {
	t.Helper()
	ts := httptest.NewServer(backend)
	t.Cleanup(ts.Close)
	return NewHandlers(&Config{BackendURL: "http://unused", S4TemporalURL: ts.URL})
}

// --- S4Client (#61: s4client.go was at 0% coverage) ---

func TestS4ClientRunScript(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scripts/run" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"workflowId":"wf-123","runId":"run-456"}`))
	}))
	defer srv.Close()

	c := NewS4Client(srv.URL)
	got, err := c.RunScript(context.Background(), S4RunRequest{Script: "minio-batch-all.sh"})
	if err != nil {
		t.Fatalf("RunScript: %v", err)
	}
	if got.WorkflowID != "wf-123" || got.RunID != "run-456" {
		t.Errorf("unexpected response: %+v", got)
	}
}

func TestS4ClientRunScriptError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewS4Client(srv.URL)
	if _, err := c.RunScript(context.Background(), S4RunRequest{Script: "x"}); err == nil {
		t.Fatal("expected error on non-200 status, got nil")
	}
}

func TestS4ClientGetStatusAndResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/scripts/status/"):
			_, _ = w.Write([]byte(`{"workflowId":"wf-1","status":"COMPLETED"}`))
		case strings.HasPrefix(r.URL.Path, "/scripts/result/"):
			_, _ = w.Write([]byte(`{"workflowId":"wf-1","result":"all good"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewS4Client(srv.URL)
	st, err := c.GetStatus(context.Background(), "wf-1")
	if err != nil || st.Status != "COMPLETED" {
		t.Fatalf("GetStatus: %+v err=%v", st, err)
	}
	res, err := c.GetResult(context.Background(), "wf-1")
	if err != nil || res.Result != "all good" {
		t.Fatalf("GetResult: %+v err=%v", res, err)
	}
}

// --- S4 handlers + script allowlist (#60) ---

func TestHandleS4BatchAnalyzeAllowed(t *testing.T) {
	h := newTestS4Handlers(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"workflowId":"wf-ok","runId":"run-ok"}`))
	})
	_, out, err := h.HandleS4BatchAnalyze(context.Background(), nil, S4BatchAnalyzeInput{Script: "minio-batch-all.sh"})
	if err != nil {
		t.Fatalf("expected success for allowed script, got %v", err)
	}
	if out.WorkflowID != "wf-ok" {
		t.Errorf("unexpected workflow id: %q", out.WorkflowID)
	}
}

func TestHandleS4BatchAnalyzeDisallowed(t *testing.T) {
	called := false
	h := newTestS4Handlers(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, _ = w.Write([]byte(`{"workflowId":"wf"}`))
	})
	_, _, err := h.HandleS4BatchAnalyze(context.Background(), nil, S4BatchAnalyzeInput{Script: "rm -rf /; evil.sh"})
	if err == nil {
		t.Fatal("expected disallowed script to be rejected")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("expected 'not allowed' error, got %v", err)
	}
	if called {
		t.Error("disallowed script must not reach the backend")
	}
}

func TestHandleS4WorkflowStatusAndResult(t *testing.T) {
	h := newTestS4Handlers(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/scripts/status/") {
			_, _ = w.Write([]byte(`{"workflowId":"wf-1","status":"RUNNING"}`))
			return
		}
		_, _ = w.Write([]byte(`{"workflowId":"wf-1","result":"done"}`))
	})

	_, st, err := h.HandleS4WorkflowStatus(context.Background(), nil, S4WorkflowStatusInput{WorkflowID: "wf-1"})
	if err != nil || st.Status != "RUNNING" {
		t.Fatalf("status: %+v err=%v", st, err)
	}
	_, res, err := h.HandleS4WorkflowResult(context.Background(), nil, S4WorkflowResultInput{WorkflowID: "wf-1"})
	if err != nil || res.Result != "done" {
		t.Fatalf("result: %+v err=%v", res, err)
	}
}

// --- analyze-s4-remediation with inline source (covers analyzeCode +
// generateMarkdownReport + the handler happy path) ---

func TestHandleAnalyzeS4RemediationInline(t *testing.T) {
	h := NewHandlers(&Config{BackendURL: "http://unused"})
	_, out, err := h.HandleAnalyzeS4Remediation(context.Background(), nil, AnalyzeS4RemediationInput{
		ObjectType:   "program",
		SourceCode:   sampleECCCode,
		OutputFormat: "markdown",
	})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if len(out.JSON.Issues) == 0 {
		t.Error("expected at least one remediation issue for ECC sample code")
	}
	if strings.TrimSpace(out.Markdown) == "" {
		t.Error("expected a non-empty markdown report")
	}
}

// --- Config allowlist (#60) ---

func TestConfigIsScriptAllowed(t *testing.T) {
	def := &Config{}
	if !def.IsScriptAllowed("minio-batch-all.sh") {
		t.Error("default allowlist should permit minio-batch-all.sh")
	}
	if def.IsScriptAllowed("evil.sh") {
		t.Error("default allowlist should reject evil.sh")
	}

	custom := &Config{S4AllowedScripts: []string{"only-this.sh"}}
	if !custom.IsScriptAllowed("only-this.sh") {
		t.Error("custom allowlist should permit only-this.sh")
	}
	if custom.IsScriptAllowed("minio-batch-all.sh") {
		t.Error("custom allowlist should override the defaults")
	}
}

func TestConfigValidate(t *testing.T) {
	if err := (&Config{}).Validate(); err == nil {
		t.Error("expected error when BackendURL is empty")
	}
	if err := (&Config{BackendURL: "http://x"}).Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- helpers / SSE bearer auth (#60) ---

func TestSplitAndTrim(t *testing.T) {
	got := splitAndTrim("  a.sh , b.sh ,, c.sh ")
	want := []string{"a.sh", "b.sh", "c.sh"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("splitAndTrim = %v, want %v", got, want)
	}
	if splitAndTrim("   ") != nil {
		t.Error("splitAndTrim of blank should be nil")
	}
}

func TestBearerAuth(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	h := bearerAuth("s3cret", next)

	tests := []struct {
		name   string
		header string
		want   int
	}{
		{"no header", "", http.StatusUnauthorized},
		{"wrong token", "Bearer nope", http.StatusUnauthorized},
		{"missing scheme", "s3cret", http.StatusUnauthorized},
		{"valid", "Bearer s3cret", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
