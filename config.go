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

import "fmt"

// Config holds the service configuration
type Config struct {
	BackendURL    string // URL of the abaper ADT backend (e.g., http://abaper:8080). Formerly abaper-ts.
	S4TemporalURL string // URL of the s4-temporal API (e.g., http://localhost:8081)

	// S4AllowedScripts is the allowlist of batch-script names that may be
	// forwarded to the script-executing s4-temporal backend. When empty, the
	// built-in defaults apply.
	S4AllowedScripts []string
}

// defaultS4AllowedScripts is the built-in allowlist of batch scripts the
// s4-temporal backend is permitted to run. Override via S4_ALLOWED_SCRIPTS.
var defaultS4AllowedScripts = []string{
	"minio-batch-all.sh",
	"minio-batch-folder.sh",
	"minio-batch-xml.sh",
}

// Validate ensures all required configuration is present
func (c *Config) Validate() error {
	if c.BackendURL == "" {
		return fmt.Errorf("ABAPER_BACKEND_URL (or the deprecated ABAPER_TS_URL) environment variable is required")
	}
	return nil
}

// AllowedScripts returns the effective batch-script allowlist — the configured
// list if set, otherwise the built-in defaults.
func (c *Config) AllowedScripts() []string {
	if len(c.S4AllowedScripts) == 0 {
		return defaultS4AllowedScripts
	}
	return c.S4AllowedScripts
}

// IsScriptAllowed reports whether name is in the effective batch-script
// allowlist. This prevents arbitrary script names from being passed through to
// the script-executing s4-temporal backend.
func (c *Config) IsScriptAllowed(name string) bool {
	for _, s := range c.AllowedScripts() {
		if s == name {
			return true
		}
	}
	return false
}
