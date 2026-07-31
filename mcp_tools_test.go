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
	"os"
	"strings"
	"testing"

	"github.com/bluefunda/abaper-mcp/internal/logger"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestMain initializes the package-level logger (normally done in main())
// before any test runs — handlers call logger.WithTool/logger.L unconditionally,
// which panics on a nil logger otherwise. "fatal" level keeps test output
// quiet since handlers only log at info/warn/error.
func TestMain(m *testing.M) {
	_ = logger.Init(logger.Config{Level: "fatal", Format: "json", ServerName: "test", Version: "test"})
	os.Exit(m.Run())
}

// newTestMCPSession stands up abaper-mcp's real tool registrations
// (registerTools) behind an in-memory MCP transport, with the Handlers'
// APIClient pointed at an httptest.Server that mocks the abaper REST
// envelope. This exercises the full stack a real MCP client (Claude
// Desktop, Claude Code, or abaper's own embedded agent) would use: JSON-RPC
// -> schema validation -> handler -> APIClient -> HTTP -> mocked backend,
// with no subprocess and no real SAP system.
func newTestMCPSession(t *testing.T, backend http.HandlerFunc) *mcp.ClientSession {
	t.Helper()
	ts := httptest.NewServer(backend)
	t.Cleanup(ts.Close)

	config := &Config{BackendURL: ts.URL}
	handlers := NewHandlers(config)

	server := mcp.NewServer(&mcp.Implementation{Name: "abaper-mcp-test", Version: "test"}, nil)
	registerTools(server, handlers)

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

// callTool invokes a tool by name and decodes its StructuredContent into T.
func callTool[T any](t *testing.T, session *mcp.ClientSession, name string, args map[string]any) (T, *mcp.CallToolResult) {
	t.Helper()
	var out T
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	if res.StructuredContent != nil {
		data, err := json.Marshal(res.StructuredContent)
		if err != nil {
			t.Fatalf("marshal StructuredContent: %v", err)
		}
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("unmarshal StructuredContent into %T: %v (raw: %s)", out, err, data)
		}
	}
	return out, res
}

// envelopeHandler returns an http.HandlerFunc that always answers the given
// {success,data} envelope, regardless of path — good enough for tests that
// only make one backend call per tool invocation.
func envelopeHandler(t *testing.T, success bool, data any, apiErr string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": success,
			"data":    data,
			"error":   apiErr,
		})
	}
}

// pathEnvelope is one path's scripted {success,data,error} response for
// pathHandler.
type pathEnvelope struct {
	Success bool
	Data    any
	Error   string
}

// pathHandler dispatches by request path, for tools whose handler makes more
// than one backend call per invocation (e.g. create-object's existence
// check before create/update, or create-and-activate's create-then-activate
// sequence). A path with no scripted envelope falls back to a
// success/empty-data response.
func pathHandler(t *testing.T, byPath map[string]pathEnvelope) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		env, ok := byPath[r.URL.Path]
		if !ok {
			env = pathEnvelope{Success: true, Data: map[string]any{}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": env.Success,
			"data":    env.Data,
			"error":   env.Error,
		})
	}
}

func TestGetObjectTool(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{
			"object_name": "ZFOO",
			"object_type": "PROG",
			"source":      "REPORT zfoo.",
		}, ""))

		out, res := callTool[GetObjectOutput](t, session, "get-object", map[string]any{
			"object_type": "program",
			"object_name": "ZFOO",
		})
		if res.IsError {
			t.Fatalf("unexpected tool error: %+v", res)
		}
		if out.SourceCode != "REPORT zfoo." {
			t.Errorf("unexpected source: %+v", out)
		}
	})

	t.Run("function requires function_group", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{}, ""))
		_, res := callTool[GetObjectOutput](t, session, "get-object", map[string]any{
			"object_type": "function",
			"object_name": "ZFM_FOO",
		})
		if !res.IsError {
			t.Error("expected IsError=true when function_group is missing")
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{}, ""))
		_, res := callTool[GetObjectOutput](t, session, "get-object", map[string]any{
			"object_type": "bogus",
			"object_name": "ZFOO",
		})
		if !res.IsError {
			t.Error("expected IsError=true for an unsupported object type")
		}
	})

	t.Run("backend error surfaces as tool error", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, false, nil, "object not found: PROG ZMISSING"))
		_, res := callTool[GetObjectOutput](t, session, "get-object", map[string]any{
			"object_type": "program",
			"object_name": "ZMISSING",
		})
		if !res.IsError {
			t.Error("expected IsError=true when the backend reports failure")
		}
	})
}

func TestSearchObjectsTool(t *testing.T) {
	session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{
		"Objects": []map[string]any{
			{"name": "ZFOO", "type": "PROG/P", "description": "Foo", "package": "$TMP"},
			{"name": "ZBAR", "type": "CLAS/OC", "description": "Bar", "package": "$TMP"},
		},
	}, ""))

	out, res := callTool[SearchObjectsOutput](t, session, "search-objects", map[string]any{
		"pattern": "Z*",
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %+v", res)
	}
	if out.Count != 2 || len(out.Objects) != 2 {
		t.Fatalf("unexpected result: %+v", out)
	}
	if out.Objects[0].Name != "ZFOO" || out.Objects[0].Type != "PROG/P" {
		t.Errorf("unexpected first object: %+v", out.Objects[0])
	}
}

func TestListPackagesTool(t *testing.T) {
	session := newTestMCPSession(t, envelopeHandler(t, true, []map[string]any{
		{"name": "$TMP", "description": "Local objects"},
	}, ""))

	out, res := callTool[ListPackagesOutput](t, session, "list-packages", map[string]any{})
	if res.IsError {
		t.Fatalf("unexpected tool error: %+v", res)
	}
	if out.Count != 1 || out.Packages[0].Name != "$TMP" {
		t.Errorf("unexpected result: %+v", out)
	}
}

func TestTestConnectionTool(t *testing.T) {
	t.Run("connected", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{
			"authenticated": true,
			"message":       "ok",
		}, ""))
		out, res := callTool[TestConnectionOutput](t, session, "test-connection", map[string]any{})
		if res.IsError {
			t.Fatalf("unexpected tool error: %+v", res)
		}
		if !out.Connected {
			t.Errorf("expected Connected=true, got %+v", out)
		}
	})

	t.Run("backend failure reported as a normal (non-error) result", func(t *testing.T) {
		// HandleTestConnection intentionally returns (nil, output, nil) even
		// on backend failure, wrapping the error into the message instead —
		// a connectivity check failing isn't a tool-protocol error.
		session := newTestMCPSession(t, envelopeHandler(t, false, nil, "connection refused"))
		out, res := callTool[TestConnectionOutput](t, session, "test-connection", map[string]any{})
		if res.IsError {
			t.Fatalf("expected IsError=false for a reported (not protocol-level) connection failure, got %+v", res)
		}
		if out.Connected {
			t.Error("expected Connected=false")
		}
	})
}

func TestCreateObjectTool(t *testing.T) {
	t.Run("creates a new object", func(t *testing.T) {
		session := newTestMCPSession(t, pathHandler(t, map[string]pathEnvelope{
			"/api/v1/objects/get":    {Success: false, Error: "object not found: PROG ZFOO"},
			"/api/v1/objects/create": {Success: true, Data: map[string]any{}},
		}))
		out, res := callTool[CreateObjectOutput](t, session, "create-object", map[string]any{
			"object_type": "program",
			"name":        "ZFOO",
			"description": "Test program",
			"package":     "$TMP",
			"source_code": "REPORT zfoo.",
		})
		if res.IsError {
			t.Fatalf("unexpected tool error: %+v", res)
		}
		if !out.Success {
			t.Errorf("expected Success=true, got %+v", out)
		}
	})

	// This is the idempotent path that abaper's own CLI (generate/deploy)
	// lacked until recently: create-object checks existence first and
	// routes to update+activate instead of failing with "already exists".
	t.Run("updates and activates when the object already exists", func(t *testing.T) {
		// UpdateObject (apiclient.go) posts to the same /api/v1/objects/create
		// path as CreateObject — abaper distinguishes create vs. save by
		// whether the body has a "description" field, the same convention
		// found in abaper's own rest/server this session. So the correct way
		// to confirm "update, not create, was used" is to check the body
		// shape at that shared path, not which path was hit.
		var sawDescriptionField bool
		session := newTestMCPSession(t, func(w http.ResponseWriter, r *http.Request) {
			env, ok := map[string]pathEnvelope{
				"/api/v1/objects/get": {Success: true, Data: map[string]any{"object_name": "ZFOO", "source": "REPORT zfoo. \" old"}},
				"/api/v1/activate":    {Success: true, Data: map[string]any{"activated": true, "messages": []any{}}},
			}[r.URL.Path]
			if !ok {
				env = pathEnvelope{Success: true, Data: map[string]any{}}
			}
			if r.URL.Path == "/api/v1/objects/create" {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				_, sawDescriptionField = body["description"]
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"success": env.Success, "data": env.Data, "error": env.Error})
		})

		out, res := callTool[CreateObjectOutput](t, session, "create-object", map[string]any{
			"object_type": "program",
			"name":        "ZFOO",
			"description": "Test program",
			"package":     "$TMP",
			"source_code": "REPORT zfoo. \" new",
		})
		if res.IsError {
			t.Fatalf("unexpected tool error: %+v", res)
		}
		if !out.Success {
			t.Errorf("expected Success=true, got %+v", out)
		}
		if sawDescriptionField {
			t.Error("expected UpdateObject's request (no description field) to be used, not CreateObject's, when the object already exists")
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{}, ""))
		out, res := callTool[CreateObjectOutput](t, session, "create-object", map[string]any{
			"object_type": "bogus",
			"name":        "ZFOO",
			"description": "x",
			"package":     "$TMP",
			"source_code": "x",
		})
		if res.IsError {
			t.Error("unsupported type is reported via Success=false in the output, not a protocol error")
		}
		if out.Success {
			t.Errorf("expected Success=false, got %+v", out)
		}
		if out.ErrorCode != "INVALID_TYPE" {
			t.Errorf("expected ErrorCode=INVALID_TYPE, got %q", out.ErrorCode)
		}
	})

	t.Run("backend create failure", func(t *testing.T) {
		session := newTestMCPSession(t, pathHandler(t, map[string]pathEnvelope{
			"/api/v1/objects/get":    {Success: false, Error: "not found"},
			"/api/v1/objects/create": {Success: false, Error: "already exists"},
		}))
		out, _ := callTool[CreateObjectOutput](t, session, "create-object", map[string]any{
			"object_type": "program",
			"name":        "ZFOO",
			"description": "Test program",
			"package":     "$TMP",
			"source_code": "REPORT zfoo.",
		})
		if out.Success {
			t.Error("expected Success=false")
		}
		if out.ErrorCode != "SAP_ERROR" {
			t.Errorf("expected ErrorCode=SAP_ERROR, got %q", out.ErrorCode)
		}
	})
}

func TestUpdateObjectTool(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{}, ""))
		out, res := callTool[UpdateObjectOutput](t, session, "update-object", map[string]any{
			"object_type": "class",
			"name":        "ZCL_FOO",
			"source_code": "CLASS zcl_foo DEFINITION PUBLIC.\nENDCLASS.",
		})
		if res.IsError {
			t.Fatalf("unexpected tool error: %+v", res)
		}
		if !out.Success {
			t.Errorf("expected Success=true, got %+v", out)
		}
	})

	t.Run("backend failure", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, false, nil, "does not exist"))
		out, _ := callTool[UpdateObjectOutput](t, session, "update-object", map[string]any{
			"object_type": "class",
			"name":        "ZCL_MISSING",
			"source_code": "CLASS zcl_missing DEFINITION PUBLIC.\nENDCLASS.",
		})
		if out.Success {
			t.Error("expected Success=false")
		}
		if out.ErrorDetail == "" || out.ErrorCode != "SAP_ERROR" {
			t.Errorf("expected SAP_ERROR with detail, got %+v", out)
		}
	})
}

func TestActivateObjectTool(t *testing.T) {
	t.Run("success with no messages", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{
			"activated": true,
			"messages":  []any{},
		}, ""))
		out, res := callTool[ActivateObjectOutput](t, session, "activate-object", map[string]any{
			"object_type": "program",
			"object_name": "ZFOO",
		})
		if res.IsError {
			t.Fatalf("unexpected tool error: %+v", res)
		}
		if !out.Success {
			t.Errorf("expected Success=true, got %+v", out)
		}
	})

	// The exact live scenario from abaper's own bug this session: HTTP-level
	// success but SAP reports activated:false with a real compiler error —
	// must surface as Success=false with the real message, not a false
	// positive.
	t.Run("activation reports failure with messages", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{
			"activated": false,
			"messages": []map[string]any{
				{"severity": "E", "text": `No component exists with the name "CARRID".`},
			},
		}, ""))
		out, res := callTool[ActivateObjectOutput](t, session, "activate-object", map[string]any{
			"object_type": "program",
			"object_name": "ZFOO",
		})
		if res.IsError {
			t.Fatalf("unexpected protocol-level error: %+v", res)
		}
		if out.Success {
			t.Error("expected Success=false")
		}
		if out.ErrorCode != "ACTIVATION_FAILED" {
			t.Errorf("expected ErrorCode=ACTIVATION_FAILED, got %q", out.ErrorCode)
		}
		if len(out.Errors) != 1 || out.Errors[0] != `[E] No component exists with the name "CARRID".` {
			t.Errorf("expected the real ADT message (severity-prefixed) to survive, got %+v", out.Errors)
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{}, ""))
		_, res := callTool[ActivateObjectOutput](t, session, "activate-object", map[string]any{
			"object_type": "bogus",
			"object_name": "ZFOO",
		})
		if !res.IsError {
			t.Error("expected IsError=true for an unsupported object type")
		}
	})
}

func TestCreateAndActivateTool(t *testing.T) {
	t.Run("creates and activates a new object", func(t *testing.T) {
		session := newTestMCPSession(t, pathHandler(t, map[string]pathEnvelope{
			"/api/v1/objects/get":    {Success: false, Error: "not found"},
			"/api/v1/objects/create": {Success: true, Data: map[string]any{}},
			"/api/v1/activate":       {Success: true, Data: map[string]any{"activated": true, "messages": []any{}}},
		}))
		out, res := callTool[CreateAndActivateOutput](t, session, "create-and-activate", map[string]any{
			"object_type": "program",
			"object_name": "ZFOO",
			"source_code": "REPORT zfoo.",
		})
		if res.IsError {
			t.Fatalf("unexpected tool error: %+v", res)
		}
		if !out.Success || out.Action != "created_and_activated" {
			t.Errorf("expected Success=true, Action=created_and_activated, got %+v", out)
		}
		if len(out.Steps) != 3 {
			t.Fatalf("expected 3 steps (check_existence, create, activate), got %d: %+v", len(out.Steps), out.Steps)
		}
	})

	// This is the atomic fix abaper's own deploy command lacked: activation
	// failure is reported as a distinct, structured outcome in one call,
	// rather than a separate step whose result silently goes unchecked.
	t.Run("reports activation_failed distinctly from create_failed", func(t *testing.T) {
		session := newTestMCPSession(t, pathHandler(t, map[string]pathEnvelope{
			"/api/v1/objects/get":    {Success: false, Error: "not found"},
			"/api/v1/objects/create": {Success: true, Data: map[string]any{}},
			"/api/v1/activate": {Success: true, Data: map[string]any{
				"activated": false,
				"messages":  []map[string]any{{"severity": "E", "text": "syntax error"}},
			}},
		}))
		out, res := callTool[CreateAndActivateOutput](t, session, "create-and-activate", map[string]any{
			"object_type": "program",
			"object_name": "ZFOO",
			"source_code": "REPORT zfoo.",
		})
		if res.IsError {
			t.Fatalf("unexpected protocol-level error: %+v", res)
		}
		if out.Success {
			t.Error("expected Success=false")
		}
		if out.Action != "activation_failed" {
			t.Errorf("expected Action=activation_failed, got %q", out.Action)
		}
	})

	t.Run("create failure reported as create_failed", func(t *testing.T) {
		session := newTestMCPSession(t, pathHandler(t, map[string]pathEnvelope{
			"/api/v1/objects/get":    {Success: false, Error: "not found"},
			"/api/v1/objects/create": {Success: false, Error: "insufficient authorization"},
		}))
		out, _ := callTool[CreateAndActivateOutput](t, session, "create-and-activate", map[string]any{
			"object_type": "program",
			"object_name": "ZFOO",
			"source_code": "REPORT zfoo.",
		})
		if out.Success || out.Action != "create_failed" {
			t.Errorf("expected Success=false, Action=create_failed, got %+v", out)
		}
	})
}

func TestRunUnitTestsTool(t *testing.T) {
	t.Run("all passed", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{
			"object_name": "ZCL_FOO",
			"total_tests": 2,
			"passed":      2,
			"failed":      0,
			"all_passed":  true,
			"test_classes": []map[string]any{
				{"name": "LTCL_TEST", "methods": []map[string]any{
					{"name": "test_ok", "status": "passed"},
					{"name": "test_ok2", "status": "passed"},
				}},
			},
		}, ""))

		out, res := callTool[RunUnitTestsOutput](t, session, "run-unit-tests", map[string]any{
			"object_type": "class",
			"object_name": "ZCL_FOO",
		})
		if res.IsError {
			t.Fatalf("unexpected tool error: %+v", res)
		}
		if !out.AllPassed || out.TotalTests != 2 || out.Passed != 2 {
			t.Errorf("unexpected result: %+v", out)
		}
		if !strings.Contains(out.Details, "LTCL_TEST") || !strings.Contains(out.Details, "PASS") {
			t.Errorf("expected human-readable details to include the test class/status, got: %s", out.Details)
		}
	})

	t.Run("some failed", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{
			"object_name": "ZCL_FOO",
			"total_tests": 2,
			"passed":      1,
			"failed":      1,
			"all_passed":  false,
			"test_classes": []map[string]any{
				{"name": "LTCL_TEST", "methods": []map[string]any{
					{"name": "test_ok", "status": "passed"},
					{"name": "test_bad", "status": "failed", "message": "assertion failed"},
				}},
			},
		}, ""))

		out, res := callTool[RunUnitTestsOutput](t, session, "run-unit-tests", map[string]any{
			"object_type": "class",
			"object_name": "ZCL_FOO",
		})
		if res.IsError {
			t.Fatalf("unexpected tool error: %+v", res)
		}
		if out.AllPassed || out.Failed != 1 {
			t.Errorf("unexpected result: %+v", out)
		}
		if !strings.Contains(out.Details, "FAIL") || !strings.Contains(out.Details, "assertion failed") {
			t.Errorf("expected failure detail in output, got: %s", out.Details)
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{}, ""))
		_, res := callTool[RunUnitTestsOutput](t, session, "run-unit-tests", map[string]any{
			"object_type": "table", // run-unit-tests explicitly excludes table/ddls/srvd/srvb, unlike activate-object
			"object_name": "ZFOO",
		})
		if !res.IsError {
			t.Error("expected IsError=true for an unsupported object type")
		}
	})

	t.Run("backend failure", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, false, nil, "object not found"))
		out, res := callTool[RunUnitTestsOutput](t, session, "run-unit-tests", map[string]any{
			"object_type": "class",
			"object_name": "ZCL_MISSING",
		})
		if res.IsError {
			t.Fatalf("backend failure is reported via AllPassed=false in output, not a protocol error: %+v", res)
		}
		if out.AllPassed {
			t.Error("expected AllPassed=false")
		}
		if out.ErrorCode != "SAP_ERROR" {
			t.Errorf("expected ErrorCode=SAP_ERROR, got %q", out.ErrorCode)
		}
	})
}

func TestSyntaxCheckTool(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{"messages": []any{}}, ""))
		out, res := callTool[SyntaxCheckOutput](t, session, "syntax-check", map[string]any{
			"object_type": "program",
			"object_name": "ZFOO",
			"source_code": "REPORT zfoo.",
		})
		if res.IsError {
			t.Fatalf("unexpected tool error: %+v", res)
		}
		if out.HasErrors || out.Count != 0 {
			t.Errorf("unexpected result: %+v", out)
		}
	})

	t.Run("has errors", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{
			"messages": []map[string]any{
				{"severity": "error", "text": "Field \"X\" is unknown", "line": 4},
				{"severity": "warning", "text": "Unused variable", "line": 1},
			},
		}, ""))
		out, res := callTool[SyntaxCheckOutput](t, session, "syntax-check", map[string]any{
			"object_type": "program",
			"object_name": "ZFOO",
			"source_code": "REPORT zfoo.\nWRITE x.",
		})
		if res.IsError {
			t.Fatalf("unexpected tool error: %+v", res)
		}
		if !out.HasErrors || out.Count != 2 {
			t.Errorf("unexpected result: %+v", out)
		}
		if out.ErrorCode != "SYNTAX_ERROR" {
			t.Errorf("expected ErrorCode=SYNTAX_ERROR, got %q", out.ErrorCode)
		}
		if len(out.Errors) != 1 || !strings.Contains(out.Errors[0], "Field") {
			t.Errorf("expected only the error-severity message in Errors (warnings excluded), got %+v", out.Errors)
		}
	})

	t.Run("backend failure is a protocol error", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, false, nil, "object does not exist"))
		_, res := callTool[SyntaxCheckOutput](t, session, "syntax-check", map[string]any{
			"object_type": "program",
			"object_name": "ZMISSING",
			"source_code": "REPORT zfoo.",
		})
		if !res.IsError {
			t.Error("expected IsError=true when the backend call itself fails")
		}
	})
}

func TestFormatCodeTool(t *testing.T) {
	session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{
		"source": "REPORT zfoo.\n\nWRITE 'hello'.",
	}, ""))
	out, res := callTool[FormatCodeOutput](t, session, "format-code", map[string]any{
		"source_code": "report zfoo.\nwrite 'hello'.",
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %+v", res)
	}
	if out.FormattedCode != "REPORT zfoo.\n\nWRITE 'hello'." {
		t.Errorf("unexpected formatted code: %q", out.FormattedCode)
	}
}

func TestTransportInfoTool(t *testing.T) {
	session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{
		"object":  "ZFOO",
		"package": "ZPKG",
		"transports": []map[string]any{
			{"number": "Q4HK900123", "description": "Test transport", "owner": "DEVELOPER", "status": "D"},
		},
	}, ""))

	out, res := callTool[TransportInfoOutput](t, session, "transport-info", map[string]any{
		"object_type": "program",
		"object_name": "ZFOO",
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %+v", res)
	}
	if out.Count != 1 || !strings.Contains(out.Transports, "Q4HK900123") {
		t.Errorf("unexpected result: %+v", out)
	}
}

func TestTransportInfoTool_NoTransports(t *testing.T) {
	session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{
		"object":     "ZFOO",
		"package":    "ZPKG",
		"transports": []map[string]any{},
	}, ""))

	out, res := callTool[TransportInfoOutput](t, session, "transport-info", map[string]any{
		"object_type": "program",
		"object_name": "ZFOO",
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %+v", res)
	}
	if out.Count != 0 || !strings.Contains(out.Transports, "No transports found") {
		t.Errorf("unexpected result: %+v", out)
	}
}

func TestCreateTransportTool(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, true, map[string]any{
			"transport_number": "Q4HK900126",
			"description":      "My change",
			"package":          "ZPKG",
		}, ""))

		out, res := callTool[CreateTransportOutput](t, session, "create-transport", map[string]any{
			"object_type": "program",
			"object_name": "ZFOO",
			"description": "My change",
		})
		if res.IsError {
			t.Fatalf("unexpected tool error: %+v", res)
		}
		if !out.Success || out.TransportNumber != "Q4HK900126" {
			t.Errorf("unexpected result: %+v", out)
		}
	})

	t.Run("backend failure reported via Success=false, not a protocol error", func(t *testing.T) {
		session := newTestMCPSession(t, envelopeHandler(t, false, nil, "no authorization to create transports"))
		out, res := callTool[CreateTransportOutput](t, session, "create-transport", map[string]any{
			"object_type": "program",
			"object_name": "ZFOO",
			"description": "My change",
		})
		if res.IsError {
			t.Fatalf("unexpected protocol-level error: %+v", res)
		}
		if out.Success {
			t.Error("expected Success=false")
		}
	})
}
