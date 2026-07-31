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
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// newTestMCPSessionWithResources is like newTestMCPSession but also
// registers resources, for tests that read abap:// URIs.
func newTestMCPSessionWithResources(t *testing.T, backend http.HandlerFunc) *mcp.ClientSession {
	t.Helper()
	ts := httptest.NewServer(backend)
	t.Cleanup(ts.Close)

	config := &Config{BackendURL: ts.URL}
	handlers := NewHandlers(config)

	server := mcp.NewServer(&mcp.Implementation{Name: "abaper-mcp-test", Version: "test"}, nil)
	registerResources(server, handlers)

	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })

	return clientSession
}

func TestResourceTemplates(t *testing.T) {
	cases := []struct {
		name     string
		uri      string
		wantType string
	}{
		{"program", "abap://program/ZFOO", "PROGRAM"},
		{"class", "abap://class/ZCL_FOO", "CLASS"},
		{"interface", "abap://interface/ZIF_FOO", "INTERFACE"},
		{"table", "abap://table/ZTAB", "TABLE"},
		{"structure", "abap://structure/ZSTRU", "STRUCTURE"},
		{"include", "abap://include/ZINCL", "INCLUDE"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			session := newTestMCPSessionWithResources(t, envelopeHandler(t, true, map[string]any{
				"object_name": "X",
				"source":      "SOURCE BODY",
			}, ""))

			res, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: tc.uri})
			if err != nil {
				t.Fatalf("ReadResource(%s): %v", tc.uri, err)
			}
			if len(res.Contents) != 1 {
				t.Fatalf("expected 1 content item, got %d", len(res.Contents))
			}
			text := res.Contents[0].Text
			if !strings.Contains(text, tc.wantType) {
				t.Errorf("expected object type %q in formatted text, got: %s", tc.wantType, text)
			}
			if !strings.Contains(text, "SOURCE BODY") {
				t.Errorf("expected source code in formatted text, got: %s", text)
			}
		})
	}
}

func TestFunctionResource(t *testing.T) {
	t.Run("valid group/name", func(t *testing.T) {
		var gotBody map[string]any
		session := newTestMCPSessionWithResources(t, func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			envelopeHandler(t, true, map[string]any{"object_name": "ZFM_FOO", "source": "FUNCTION zfm_foo."}, "")(w, r)
		})

		res, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "abap://function/ZFG/ZFM_FOO"})
		if err != nil {
			t.Fatalf("ReadResource: %v", err)
		}
		if !strings.Contains(res.Contents[0].Text, "FUNCTION zfm_foo.") {
			t.Errorf("unexpected content: %s", res.Contents[0].Text)
		}
		if gotBody["function_group"] != "ZFG" {
			t.Errorf("expected function_group=ZFG in the backend request, got %+v", gotBody)
		}
	})

	t.Run("missing name segment", func(t *testing.T) {
		session := newTestMCPSessionWithResources(t, envelopeHandler(t, true, map[string]any{}, ""))
		_, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "abap://function/ZFG"})
		if err == nil {
			t.Fatal("expected an error for a URI missing the function name segment")
		}
	})
}

func TestPackagesResource(t *testing.T) {
	session := newTestMCPSessionWithResources(t, envelopeHandler(t, true, []map[string]any{
		{"name": "$TMP", "description": "Local objects"},
		{"name": "ZPKG", "description": ""},
	}, ""))

	res, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "abap://packages"})
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	text := res.Contents[0].Text
	if !strings.Contains(text, "$TMP") || !strings.Contains(text, "Local objects") || !strings.Contains(text, "ZPKG") {
		t.Errorf("unexpected packages content: %s", text)
	}
}

func TestResourceNotFound(t *testing.T) {
	session := newTestMCPSessionWithResources(t, envelopeHandler(t, false, nil, "object not found: PROG ZMISSING"))
	_, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "abap://program/ZMISSING"})
	if err == nil {
		t.Fatal("expected an error when the backend reports the object doesn't exist")
	}
}
