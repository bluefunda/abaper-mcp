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

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// newTestMCPSessionWithPrompts is like newTestMCPSession but registers
// prompts instead of tools.
func newTestMCPSessionWithPrompts(t *testing.T, backend http.HandlerFunc) *mcp.ClientSession {
	t.Helper()
	ts := httptest.NewServer(backend)
	t.Cleanup(ts.Close)

	config := &Config{AbaperTSURL: ts.URL}
	handlers := NewHandlers(config)

	server := mcp.NewServer(&mcp.Implementation{Name: "abaper-mcp-test", Version: "test"}, nil)
	registerPrompts(server, handlers)

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

func TestPrompts(t *testing.T) {
	names := []string{
		"analyze-abap", "review-abap", "optimize-abap",
		"document-abap", "test-abap", "refactor-abap", "explain-abap",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			session := newTestMCPSessionWithPrompts(t, envelopeHandler(t, true, map[string]any{
				"object_name": "ZFOO",
				"source":      "REPORT zfoo.",
			}, ""))

			res, err := session.GetPrompt(context.Background(), &mcp.GetPromptParams{
				Name:      name,
				Arguments: map[string]string{"object_type": "program", "object_name": "ZFOO"},
			})
			if err != nil {
				t.Fatalf("GetPrompt(%s): %v", name, err)
			}
			if len(res.Messages) != 1 {
				t.Fatalf("expected 1 message, got %d", len(res.Messages))
			}
			text, ok := res.Messages[0].Content.(*mcp.TextContent)
			if !ok {
				t.Fatalf("expected TextContent, got %T", res.Messages[0].Content)
			}
			if !strings.Contains(text.Text, "REPORT zfoo.") {
				t.Errorf("expected the fetched source code to be embedded in the prompt, got: %s", text.Text)
			}
			if !strings.Contains(text.Text, "ZFOO") {
				t.Errorf("expected the object name to be embedded in the prompt, got: %s", text.Text)
			}
		})
	}
}

func TestPrompt_SourceFetchFailure(t *testing.T) {
	session := newTestMCPSessionWithPrompts(t, envelopeHandler(t, false, nil, "object not found: PROG ZMISSING"))
	_, err := session.GetPrompt(context.Background(), &mcp.GetPromptParams{
		Name:      "analyze-abap",
		Arguments: map[string]string{"object_type": "program", "object_name": "ZMISSING"},
	})
	if err == nil {
		t.Fatal("expected an error when the backend can't fetch the source code")
	}
}
