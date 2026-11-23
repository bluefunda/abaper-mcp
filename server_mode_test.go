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
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// SAP credentials are per user only in server mode: these tests guard that no
// SSE session can reach the backend without its own user's SAP identity.

func TestValidateForMode(t *testing.T) {
	full := Config{BackendURL: "http://b", CAIBFFInternalURL: "http://bff", InternalServiceSecret: "s"}
	cases := []struct {
		name    string
		mode    string
		mutate  func(*Config)
		wantErr string
	}{
		{"sse fully configured", modeSSE, nil, ""},
		{"sse without cai-bff url", modeSSE, func(c *Config) { c.CAIBFFInternalURL = "" }, "CAI_BFF_INTERNAL_URL"},
		{"sse without internal secret", modeSSE, func(c *Config) { c.InternalServiceSecret = "" }, "INTERNAL_SERVICE_SECRET"},
		{"sse without either", modeSSE, func(c *Config) { c.CAIBFFInternalURL, c.InternalServiceSecret = "", "" }, "CAI_BFF_INTERNAL_URL and INTERNAL_SERVICE_SECRET"},
		{"stdio local", modeStdio, func(c *Config) { c.CAIBFFInternalURL, c.InternalServiceSecret = "", "" }, ""},
		{"stdio with server settings", modeStdio, nil, "ABAPER_MODE=sse"},
		{"unknown mode", "http", nil, "unknown ABAPER_MODE"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := full
			if c.mutate != nil {
				c.mutate(&cfg)
			}
			err := cfg.ValidateForMode(c.mode)
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("error = %v, want one mentioning %q", err, c.wantErr)
			}
		})
	}
}

// fakeBackend records the SAP identity headers of every request it receives.
type fakeBackend struct {
	srv     *httptest.Server
	calls   atomic.Int32
	sapUser atomic.Value
}

func newFakeBackend(t *testing.T) *fakeBackend {
	t.Helper()
	b := &fakeBackend{}
	b.sapUser.Store("")
	b.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.calls.Add(1)
		b.sapUser.Store(r.Header.Get("X-SAP-User"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{}}`))
	}))
	t.Cleanup(b.srv.Close)
	return b
}

func newFakeCAIBFF(t *testing.T, wantSecret string, creds map[string]SAPCredentials) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Secret") != wantSecret {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		c, ok := creds[r.URL.Query().Get("userID")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(c)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestHandlersForSession_NoUserIDFailsClosed(t *testing.T) {
	backend := newFakeBackend(t)
	bff := newFakeCAIBFF(t, "s", map[string]SAPCredentials{})
	cfg := &Config{BackendURL: backend.srv.URL, CAIBFFInternalURL: bff.URL, InternalServiceSecret: "s"}

	req := principalRequest()
	req.Header.Del(headerPrincipalUserID)
	h := handlersForSession(req, cfg)

	_, err := h.apiClient.post(context.Background(), "/api/v1/objects/get", map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "sign in and connect SAP in Settings") {
		t.Fatalf("post() error = %v, want the sign-in error", err)
	}
	if _, err := h.apiClient.Activate(context.Background(), "CLAS", "ZCL_X"); err == nil {
		t.Fatal("Activate() must fail without a signed-in user")
	}
	if n := backend.calls.Load(); n != 0 {
		t.Fatalf("backend was called %d times; a session without a user must never reach it", n)
	}
}

func TestHandlersForSession_UnconfiguredLookupFailsClosed(t *testing.T) {
	backend := newFakeBackend(t)
	cfg := &Config{BackendURL: backend.srv.URL}

	h := handlersForSession(principalRequest(), cfg)
	if _, err := h.apiClient.post(context.Background(), "/api/v1/objects/get", map[string]string{}); err == nil {
		t.Fatal("post() must fail when per-user lookup is not configured")
	}
	if n := backend.calls.Load(); n != 0 {
		t.Fatalf("backend was called %d times without a per-user identity", n)
	}
}

func TestHandlersForSession_ConnectedUserUsesOwnCredentials(t *testing.T) {
	backend := newFakeBackend(t)
	bff := newFakeCAIBFF(t, "s", map[string]SAPCredentials{
		"8267bd3b": {Host: "https://sap.example.com", Client: "100", Username: "ALICE", Password: "pw"},
	})
	cfg := &Config{BackendURL: backend.srv.URL, CAIBFFInternalURL: bff.URL, InternalServiceSecret: "s"}

	h := handlersForSession(principalRequest(), cfg)
	if h.principal.UserID != "8267bd3b" {
		t.Errorf("principal.UserID = %q, want the forwarded user", h.principal.UserID)
	}

	if _, err := h.apiClient.post(context.Background(), "/api/v1/objects/get", map[string]string{}); err != nil {
		t.Fatalf("post() unexpected error: %v", err)
	}
	if got := backend.sapUser.Load(); got != "ALICE" {
		t.Errorf("post() X-SAP-User = %q, want the connected user's own SAP user", got)
	}

	// Activate builds its own request; it must carry the same identity.
	backend.sapUser.Store("")
	if _, err := h.apiClient.Activate(context.Background(), "CLAS", "ZCL_X"); err != nil {
		t.Fatalf("Activate() unexpected error: %v", err)
	}
	if got := backend.sapUser.Load(); got != "ALICE" {
		t.Errorf("Activate() X-SAP-User = %q, want the connected user's own SAP user", got)
	}
}

func TestHandlersForSession_NotConnectedUserFailsClosed(t *testing.T) {
	backend := newFakeBackend(t)
	bff := newFakeCAIBFF(t, "s", map[string]SAPCredentials{})
	cfg := &Config{BackendURL: backend.srv.URL, CAIBFFInternalURL: bff.URL, InternalServiceSecret: "s"}

	h := handlersForSession(principalRequest(), cfg)
	_, err := h.apiClient.post(context.Background(), "/api/v1/objects/get", map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "SAP is not connected") {
		t.Fatalf("post() error = %v, want the not-connected error", err)
	}
	if n := backend.calls.Load(); n != 0 {
		t.Fatalf("backend was called %d times for a user without SAP credentials", n)
	}
}

func TestNewHandlersForSession_NoCredsAndNoErrorFailsClosed(t *testing.T) {
	backend := newFakeBackend(t)
	h := NewHandlersForSession(&Config{BackendURL: backend.srv.URL}, nil, nil, "", "")
	if _, err := h.apiClient.post(context.Background(), "/api/v1/objects/get", map[string]string{}); err == nil {
		t.Fatal("post() must fail with neither credentials nor an error")
	}
	if n := backend.calls.Load(); n != 0 {
		t.Fatalf("backend was called %d times without a per-user identity", n)
	}
}
