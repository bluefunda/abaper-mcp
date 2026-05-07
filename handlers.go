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

// Handlers holds all MCP request handlers
type Handlers struct {
	config    *Config
	apiClient *APIClient
	s4Client  *S4Client
}

// NewHandlers creates a new handlers instance
func NewHandlers(config *Config) *Handlers {
	h := &Handlers{
		config:    config,
		apiClient: NewAPIClient(config.AbaperTSURL),
	}
	if config.S4TemporalURL != "" {
		h.s4Client = NewS4Client(config.S4TemporalURL)
	}
	return h
}
