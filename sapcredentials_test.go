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
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchSAPCredentials_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("userID"); got != "user-1" {
			t.Errorf("userID query param = %q, want %q", got, "user-1")
		}
		if got := r.Header.Get("X-Internal-Secret"); got != "shh" {
			t.Errorf("X-Internal-Secret header = %q, want %q", got, "shh")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"host":"https://sap.example.com","client":"100","username":"alice","password":"hunter2"}`))
	}))
	defer srv.Close()

	creds, err := fetchSAPCredentials(context.Background(), srv.URL, "shh", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.Host != "https://sap.example.com" || creds.Client != "100" || creds.Username != "alice" || creds.Password != "hunter2" {
		t.Errorf("unexpected credentials: %+v", creds)
	}
}

func TestFetchSAPCredentials_NotConnectedReturnsSentinelError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := fetchSAPCredentials(context.Background(), srv.URL, "", "never-connected")
	if !errors.Is(err, ErrSAPNotConnected) {
		t.Fatalf("expected ErrSAPNotConnected, got %v", err)
	}
}

func TestFetchSAPCredentials_ServerErrorIsNotSAPNotConnected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := fetchSAPCredentials(context.Background(), srv.URL, "", "user-1")
	if err == nil {
		t.Fatal("expected an error for a 500 response")
	}
	if errors.Is(err, ErrSAPNotConnected) {
		t.Error("a transport/server failure must not be reported as ErrSAPNotConnected — callers distinguish these")
	}
}
