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
	"errors"
	"fmt"
	"net/http"
	"testing"
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

func TestAPIErrorMessage(t *testing.T) {
	err := &APIError{Path: "/api/v1/activate", StatusCode: http.StatusBadRequest, Message: "syntax error"}
	want := "API error from /api/v1/activate (status 400): syntax error"
	if got := err.Error(); got != want {
		t.Errorf("APIError.Error() = %q, want %q", got, want)
	}
}
