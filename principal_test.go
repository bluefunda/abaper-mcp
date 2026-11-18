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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluefunda/abaper-mcp/internal/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

const testJWT = "eyJhbGciOiJSUzI1NiJ9.secret-payload.signature"

func principalRequest() *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(headerPrincipalUserID, "8267bd3b")
	req.Header.Set(headerPrincipalRealm, "individual")
	req.Header.Set(headerPrincipalEmail, "user@example.com")
	req.Header.Set(headerPrincipalToken, testJWT)
	req.Header.Set(headerPrincipalCorrelationID, "sess-42")
	return req
}

// Tool-call log lines must say which human, in which tenant, caused them —
// and must never carry the token or the email itself.
func TestToolLoggerAttributesThePrincipal(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	prev := logger.L
	logger.L = zap.New(core)
	t.Cleanup(func() { logger.L = prev })

	h := &Handlers{principal: principalFromRequest(principalRequest())}
	log, _ := h.newToolLogger("get-object")
	log.Info("tool called")

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected one log entry, got %d", len(entries))
	}
	fields := entries[0].ContextMap()
	for key, want := range map[string]any{
		"tool":                    "get-object",
		"user_id":                 "8267bd3b",
		"realm":                   "individual",
		"correlation_id":          "sess-42",
		"email_present":           true,
		"principal_token_present": true,
		"principal_token_len":     int64(len(testJWT)),
	} {
		if fields[key] != want {
			t.Errorf("%s = %v, want %v", key, fields[key], want)
		}
	}
	for key, v := range fields {
		if s, ok := v.(string); ok && (strings.Contains(s, "secret-payload") || strings.Contains(s, "@example.com")) {
			t.Errorf("field %s leaks the token or email: %q", key, s)
		}
	}
}

func TestToolLoggerWithoutPrincipalAddsNothing(t *testing.T) {
	if fields := (principal{}).fields(); fields != nil {
		t.Fatalf("stdio / no-principal sessions must add no fields, got %v", fields)
	}
}

func TestRequirePrincipal(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	h := requirePrincipal(next)

	cases := []struct {
		name   string
		strip  string
		status int
	}{
		{"user id and realm present", "", http.StatusOK},
		{"token is optional", headerPrincipalToken, http.StatusOK},
		{"missing user id", headerPrincipalUserID, http.StatusUnauthorized},
		{"missing realm", headerPrincipalRealm, http.StatusUnauthorized},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := principalRequest()
			if c.strip != "" {
				req.Header.Del(c.strip)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != c.status {
				t.Fatalf("status = %d, want %d", rec.Code, c.status)
			}
		})
	}
}
