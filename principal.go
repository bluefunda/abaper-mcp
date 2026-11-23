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
	"net/http"

	"go.uber.org/zap"
)

// headerPrincipal* mirror cai-llm-router's internal/mcp/principal.go Header*
// constants — duplicated (not imported) since that's an internal/ package of a
// different module and these are the wire contract abaper-mcp receives, not
// sends. Changing any of these silently stops per-user SAP resolution and
// audit attribution from working; treat them as frozen.
const (
	headerPrincipalUserID = "X-BF-User-Id"
	headerPrincipalRealm  = "X-BF-Realm"
	// headerPrincipalToken carries the calling user's own Keycloak JWT —
	// forwarded to the real abaper backend as "Authorization: Bearer
	// <token>", required alongside X-Realm/X-SAP-* (confirmed via a live
	// test: those alone got an empty 401 from a gate in front of the real
	// backend).
	headerPrincipalToken         = "X-BF-Principal-Token"
	headerPrincipalEmail         = "X-BF-User-Email"
	headerPrincipalCorrelationID = "X-BF-Correlation-Id"
)

// principal is the calling end user as forwarded by cai-llm-router, kept for
// audit attribution: which human, in which tenant, caused a tool call
// (bluefunda/abaper-mcp#77). It is identity, not authentication — transport
// auth stays the static bearer token in Authorization (bearerAuth).
//
// The token itself is deliberately not stored here: only whether it was
// present and how long it was, so it can never reach a log line.
type principal struct {
	UserID        string
	Realm         string
	CorrelationID string
	EmailPresent  bool
	TokenLen      int
}

func principalFromRequest(req *http.Request) principal {
	return principal{
		UserID:        req.Header.Get(headerPrincipalUserID),
		Realm:         req.Header.Get(headerPrincipalRealm),
		CorrelationID: req.Header.Get(headerPrincipalCorrelationID),
		EmailPresent:  req.Header.Get(headerPrincipalEmail) != "",
		TokenLen:      len(req.Header.Get(headerPrincipalToken)),
	}
}

// fields returns the principal as log fields. Empty when there is no
// principal (stdio mode, or a caller that forwards none).
func (p principal) fields() []zap.Field {
	if p == (principal{}) {
		return nil
	}
	return []zap.Field{
		zap.String("user_id", p.UserID),
		zap.String("realm", p.Realm),
		zap.String("correlation_id", p.CorrelationID),
		zap.Bool("email_present", p.EmailPresent),
		zap.Bool("principal_token_present", p.TokenLen > 0),
		zap.Int("principal_token_len", p.TokenLen),
	}
}

// requirePrincipal rejects MCP requests that carry no principal — on by
// default (ABAPER_REQUIRE_PRINCIPAL=false turns it off). Realm and user ID
// are required; the JWT is not, since cai-llm-router only forwards it when the
// caller presented one.
func requirePrincipal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(headerPrincipalUserID) == "" || r.Header.Get(headerPrincipalRealm) == "" {
			http.Error(w, "missing principal: "+headerPrincipalUserID+" and "+headerPrincipalRealm+" are required; sign in and connect SAP in Settings", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
