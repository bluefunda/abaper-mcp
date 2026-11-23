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
	"fmt"
	"strings"
)

// Config holds the service configuration
type Config struct {
	BackendURL    string // URL of the abaper ADT backend (e.g., http://abaper:8080). Formerly abaper-ts.
	S4TemporalURL string // URL of the s4-temporal API (e.g., http://localhost:8081)

	// S4AllowedScripts is the allowlist of batch-script names that may be
	// forwarded to the script-executing s4-temporal backend. When empty, the
	// built-in defaults apply.
	S4AllowedScripts []string

	// CAIBFFInternalURL is cai-bff's base URL, used to look up a connected
	// user's own SAP credentials per session (see sapcredentials.go and
	// bluefunda/abaper-mcp#79). Required in SSE mode: there is no shared SAP
	// identity to fall back to, so the server refuses to start without it.
	CAIBFFInternalURL string
	// InternalServiceSecret authenticates to cai-bff's internal credential
	// endpoint (X-Internal-Secret header) — must match cai-bff's own
	// INTERNAL_SERVICE_SECRET. Required in SSE mode.
	InternalServiceSecret string

	// RequirePrincipal rejects SSE/HTTP requests that carry no forwarded
	// principal (X-BF-User-Id + X-BF-Realm) with a 401. On by default
	// (ABAPER_REQUIRE_PRINCIPAL=false turns it off). Turning it off never
	// enables a shared SAP identity: a session without a user then gets a
	// tool error instead of a 401 — see principal.go and serverForSession.
	RequirePrincipal bool
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

// Transport modes (ABAPER_MODE).
const (
	modeStdio = "stdio"
	modeSSE   = "sse"
)

// ValidateForMode checks the configuration against the transport mode.
//
// SSE mode serves many users over the network, so every session must resolve
// its own SAP credentials from cai-bff: it needs CAI_BFF_INTERNAL_URL and
// INTERNAL_SERVICE_SECRET, and there is no shared SAP identity to fall back
// to.
//
// stdio mode is a single local user. It refuses an unknown mode (so a typo
// such as "http" cannot silently start the shared-identity stdio path) and
// server-only settings (so a server deployment that forgot ABAPER_MODE=sse
// fails loudly instead of running without per-user credentials).
func (c *Config) ValidateForMode(mode string) error {
	switch mode {
	case modeSSE:
		var missing []string
		if c.CAIBFFInternalURL == "" {
			missing = append(missing, "CAI_BFF_INTERNAL_URL")
		}
		if c.InternalServiceSecret == "" {
			missing = append(missing, "INTERNAL_SERVICE_SECRET")
		}
		if len(missing) > 0 {
			return fmt.Errorf("SSE mode requires per-user SAP credential lookup; set %s (there is no shared SAP identity in server mode)", strings.Join(missing, " and "))
		}
		return nil
	case modeStdio:
		if c.CAIBFFInternalURL != "" || c.InternalServiceSecret != "" {
			return fmt.Errorf("CAI_BFF_INTERNAL_URL/INTERNAL_SERVICE_SECRET are server-mode settings; set ABAPER_MODE=sse to run as a server (stdio mode is for a single local user only)")
		}
		return nil
	default:
		return fmt.Errorf("unknown ABAPER_MODE %q: must be %q or %q", mode, modeStdio, modeSSE)
	}
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
