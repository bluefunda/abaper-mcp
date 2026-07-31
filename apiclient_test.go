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
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error is not a not-found",
			err:  nil,
			want: false,
		},
		{
			name: "http 404 status is not-found",
			err:  &APIError{Path: "/api/v1/objects/get", StatusCode: http.StatusNotFound, Message: "boom"},
			want: true,
		},
		{
			name: "message 'not found' is not-found",
			err:  &APIError{Path: "/api/v1/objects/get", StatusCode: http.StatusOK, Message: "Object ZFOO not found"},
			want: true,
		},
		{
			name: "message 'does not exist' is not-found",
			err:  &APIError{Path: "/api/v1/objects/get", StatusCode: http.StatusOK, Message: "object does not exist"},
			want: true,
		},
		{
			name: "wrapped APIError is still detected",
			err:  fmt.Errorf("get object: %w", &APIError{StatusCode: http.StatusNotFound}),
			want: true,
		},
		{
			name: "backend error that is not a missing object",
			err:  &APIError{Path: "/api/v1/objects/get", StatusCode: http.StatusInternalServerError, Message: "auth token expired"},
			want: false,
		},
		{
			name: "transport error is never treated as not-found",
			err:  errors.New("request to /api/v1/objects/get failed: connection refused"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// TestGetObjectContextCancellation verifies that cancelling the context aborts
// an in-flight backend request promptly instead of blocking until the 60s
// client timeout — i.e. the handler ctx is actually wired into the HTTP call.
func TestGetObjectContextCancellation(t *testing.T) {
	quit := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hold the request open until the test finishes so it can only return
		// via the client cancelling its context.
		select {
		case <-r.Context().Done():
		case <-quit:
		}
	}))
	defer srv.Close()
	defer close(quit)

	client := NewAPIClient(srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	done := make(chan error, 1)
	start := time.Now()
	go func() {
		_, err := client.GetObject(ctx, "PROG", "ZFOO", "")
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Errorf("request did not abort promptly on cancellation: took %s", elapsed)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("GetObject did not return after context cancellation — ctx not wired into the request")
	}
}

func TestAPIErrorMessage(t *testing.T) {
	err := &APIError{Path: "/api/v1/activate", StatusCode: http.StatusBadRequest, Message: "syntax error"}
	want := "API error from /api/v1/activate (status 400): syntax error"
	if got := err.Error(); got != want {
		t.Errorf("APIError.Error() = %q, want %q", got, want)
	}
}
