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
	"net/http"
	"testing"
)

// TestLoadPatternsCached verifies the embedded remediation patterns are parsed
// once and the same instance is reused across calls (#58).
func TestLoadPatternsCached(t *testing.T) {
	p1, err := loadPatterns()
	if err != nil {
		t.Fatalf("loadPatterns: %v", err)
	}
	p2, err := loadPatterns()
	if err != nil {
		t.Fatalf("loadPatterns (2nd call): %v", err)
	}
	if p1 != p2 {
		t.Error("loadPatterns returned different instances; expected a cached single parse")
	}
	if len(p1.RemediationPatterns) == 0 {
		t.Error("expected at least one remediation pattern to be loaded")
	}
}

// TestResourceFetchError verifies not-found errors map to a ResourceNotFoundError
// while any other error is preserved (wrapped), not masked as missing (#59).
func TestResourceFetchError(t *testing.T) {
	uri := "abap://program/ZFOO"

	// A genuine not-found must NOT wrap the underlying APIError (it becomes a
	// distinct ResourceNotFoundError).
	notFound := &APIError{Path: "/api/v1/objects/get", StatusCode: http.StatusNotFound, Message: "not found"}
	got := resourceFetchError(uri, notFound)
	if got == nil {
		t.Fatal("expected an error, got nil")
	}
	if errors.Is(got, notFound) {
		t.Error("not-found should map to ResourceNotFoundError, not wrap the transport error")
	}

	// Any other error (network/auth/backend) must be preserved so the real
	// cause is not lost.
	transport := errors.New("connection refused")
	got = resourceFetchError(uri, transport)
	if !errors.Is(got, transport) {
		t.Errorf("expected wrapped transport error to be preserved, got %v", got)
	}
}
