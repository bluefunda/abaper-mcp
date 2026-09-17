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
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bluefunda/abaper-mcp/internal/logger"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// Version information (set via ldflags during build)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("abaper-mcp version %s\n", Version)
		fmt.Printf("Build time: %s\n", BuildTime)
		fmt.Printf("Git commit: %s\n", GitCommit)
		os.Exit(0)
	}

	logLevel := getEnv("LOG_LEVEL", "info")
	logFormat := getEnv("LOG_FORMAT", "json")
	_ = logger.Init(logger.Config{
		Level:      logLevel,
		Format:     logFormat,
		ServerName: "abaper-mcp",
		Version:    Version,
	})
	defer logger.Sync()

	mode := getEnv("ABAPER_MODE", "stdio")
	logger.L.Info("Starting ABAPER MCP server",
		zap.String("mode", mode),
		zap.String("log_level", logLevel),
	)

	config := &Config{
		// ABAPER_BACKEND_URL is the canonical name; ABAPER_TS_URL is kept as a
		// deprecated alias for the former abaper-ts backend.
		BackendURL:            getEnv("ABAPER_BACKEND_URL", getEnv("ABAPER_TS_URL", "http://localhost:8080")),
		S4TemporalURL:         getEnv("S4_TEMPORAL_URL", ""),
		S4AllowedScripts:      splitAndTrim(getEnv("S4_ALLOWED_SCRIPTS", "")),
		CAIBFFInternalURL:     getEnv("CAI_BFF_INTERNAL_URL", ""),
		InternalServiceSecret: getEnv("INTERNAL_SERVICE_SECRET", ""),
	}

	if err := config.Validate(); err != nil {
		logger.L.Fatal("Invalid configuration", zap.Error(err))
	}

	logger.L.Info("Configuration loaded",
		zap.String("backend_url", config.BackendURL),
		zap.String("s4_temporal_url", config.S4TemporalURL),
		zap.Bool("per_user_sap_credentials_enabled", config.CAIBFFInternalURL != ""),
	)

	// Cancel the root context on SIGINT/SIGTERM so both transports can shut
	// down gracefully instead of being killed mid-request.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch mode {
	case "sse":
		runSSEMode(ctx, config)
	default: // stdio
		// stdio is always a single local process talking to one Claude
		// Desktop/Code instance — there is no per-request principal to
		// resolve per-user SAP credentials from, so this always uses the
		// backend's shared/default SAP identity (config.BackendURL).
		server := newServer()
		handlers := NewHandlers(config)
		registerTools(server, handlers)
		registerResources(server, handlers)
		registerPrompts(server, handlers)
		runStdioMode(ctx, server)
	}
}

// newServer builds a fresh, empty *mcp.Server — called once for stdio mode,
// and once per SSE session (see resolveServerForSession) so each session's
// tools are bound to that session's own resolved SAP identity.
func newServer() *mcp.Server {
	return mcp.NewServer(
		&mcp.Implementation{
			Name:    "abaper-mcp",
			Version: Version,
		},
		nil,
	)
}

func runStdioMode(ctx context.Context, server *mcp.Server) {
	logger.L.Info("Running in stdio mode")
	// server.Run returns when ctx is cancelled (SIGINT/SIGTERM); that is a
	// clean shutdown, not a server error.
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil && !errors.Is(err, context.Canceled) {
		logger.L.Fatal("Server error", zap.Error(err))
	}
	logger.L.Info("Server shut down cleanly")
}

func runSSEMode(ctx context.Context, config *Config) {
	port := getEnv("ABAPER_HTTP_PORT", "8015")
	host := getEnv("ABAPER_HTTP_HOST", "0.0.0.0")
	useLegacySSE := getEnv("ABAPER_USE_LEGACY_SSE", "true") == "true"

	logger.L.Info("Running in SSE/HTTP mode",
		zap.String("host", host),
		zap.String("port", port),
		zap.Bool("legacy_sse", useLegacySSE),
	)

	var handler http.Handler

	if useLegacySSE {
		handler = mcp.NewSSEHandler(func(req *http.Request) *mcp.Server {
			logger.L.Debug("SSE client connected",
				zap.String("remote_addr", req.RemoteAddr),
				zap.String("path", req.URL.Path),
			)
			return serverForSession(req, config)
		}, nil)
	} else {
		handler = mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
			logger.L.Debug("HTTP client connected",
				zap.String("remote_addr", req.RemoteAddr),
				zap.String("path", req.URL.Path),
			)
			return serverForSession(req, config)
		}, &mcp.StreamableHTTPOptions{
			Stateless:      false,
			JSONResponse:   false,
			SessionTimeout: 30 * time.Minute,
		})
	}

	mux := http.NewServeMux()
	// Optional bearer-token auth for the MCP endpoint. /health is registered
	// separately below and stays public for liveness probes.
	if token := getEnv("ABAPER_AUTH_TOKEN", ""); token != "" {
		logger.L.Info("SSE endpoint authentication enabled (bearer token)")
		mux.Handle("/", bearerAuth(token, handler))
	} else {
		logger.L.Warn("SSE endpoint is UNAUTHENTICATED; front it with an authenticating gateway or set ABAPER_AUTH_TOKEN")
		mux.Handle("/", handler)
	}
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","version":"` + Version + `","mode":"sse"}`))
	})

	addr := fmt.Sprintf("%s:%s", host, port)
	logger.L.Info("SSE/HTTP MCP server listening",
		zap.String("address", "http://"+addr),
		zap.String("health_check", "http://"+addr+"/health"),
	)

	// Explicit timeouts guard against slow-client (Slowloris) resource
	// exhaustion (gosec G114). WriteTimeout is left at 0 because SSE responses
	// are long-lived streams; a non-zero value would cut active streams.
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		logger.L.Fatal("HTTP server error", zap.Error(err))
	case <-ctx.Done():
		logger.L.Info("Shutdown signal received, draining connections")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.L.Error("Graceful shutdown failed", zap.Error(err))
			return
		}
		logger.L.Info("Server shut down cleanly")
	}
}

// headerPrincipalUserID, headerPrincipalRealm and headerPrincipalToken mirror
// cai-llm-router's internal/mcp/principal.go Header* constants — duplicated
// (not imported) since that's an internal/ package of a different module and
// these are the wire contract abaper-mcp receives, not sends. Changing any of
// these silently stops per-user SAP resolution from working; treat them as
// frozen.
const (
	headerPrincipalUserID = "X-BF-User-Id"
	headerPrincipalRealm  = "X-BF-Realm"
	// headerPrincipalToken carries the calling user's own Keycloak JWT —
	// forwarded to the real abaper backend as "Authorization: Bearer
	// <token>", required alongside X-Realm/X-SAP-* (confirmed via a live
	// test: those alone got an empty 401 from a gate in front of the real
	// backend).
	headerPrincipalToken = "X-BF-Principal-Token"
)

// serverForSession builds a fresh *mcp.Server + Handlers for one SSE/HTTP
// session, resolving that session's SAP identity from the connecting
// request's forwarded principal (see bluefunda/abaper-mcp#79). cai-llm-router
// builds a new MCP client per chat request and attaches the principal to
// every outbound request on it (see cai-llm-router's principalRoundTripper) —
// so the request that establishes this session reliably belongs to one user
// for the session's whole lifetime, and resolving once here (rather than per
// tool call) is correct, not just a convenient shortcut.
//
// Three outcomes:
//   - No X-BF-User-Id header, or per-user lookup not configured at all
//     (CAIBFFInternalURL empty): falls back to the shared/default SAP
//     identity — identical to this feature not existing, e.g. stdio-style
//     callers or any deployment that hasn't enabled it.
//   - X-BF-User-Id present and cai-bff has stored credentials: every backend
//     call in this session uses that user's own SAP system.
//   - X-BF-User-Id present but no credentials found (or the lookup itself
//     failed): every backend call in this session fails with a clear error
//     instead of silently using the shared identity — a user who explicitly
//     has their own SAP connection expected must never have their prompt
//     answered against the wrong, unrelated SAP system.
func serverForSession(req *http.Request, config *Config) *mcp.Server {
	server := newServer()

	userID := req.Header.Get(headerPrincipalUserID)
	if userID == "" || config.CAIBFFInternalURL == "" {
		handlers := NewHandlers(config)
		registerTools(server, handlers)
		registerResources(server, handlers)
		registerPrompts(server, handlers)
		return server
	}

	realm := req.Header.Get(headerPrincipalRealm)
	principalToken := req.Header.Get(headerPrincipalToken)

	creds, err := fetchSAPCredentials(req.Context(), config.CAIBFFInternalURL, config.InternalServiceSecret, userID)
	var handlers *Handlers
	switch {
	case err == nil:
		logger.L.Debug("resolved per-user SAP credentials for session",
			zap.String("userID", userID),
			zap.Bool("principal_token_present", principalToken != ""),
		)
		handlers = NewHandlersForSession(config, creds, nil, realm, principalToken)
	case errors.Is(err, ErrSAPNotConnected):
		logger.L.Info("session's user has not connected SAP", zap.String("userID", userID))
		handlers = NewHandlersForSession(config, nil, fmt.Errorf("SAP is not connected — connect it in Settings → Agents first"), realm, principalToken)
	default:
		logger.L.Error("sap credentials lookup failed", zap.String("userID", userID), zap.Error(err))
		handlers = NewHandlersForSession(config, nil, fmt.Errorf("failed to resolve your SAP connection, please try again: %w", err), realm, principalToken)
	}

	registerTools(server, handlers)
	registerResources(server, handlers)
	registerPrompts(server, handlers)
	return server
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// bearerAuth wraps next with a constant-time check that the request carries a
// matching "Authorization: Bearer <token>" header, returning 401 otherwise.
func bearerAuth(token string, next http.Handler) http.Handler {
	want := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(got, want) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// splitAndTrim splits a comma-separated list into trimmed, non-empty entries.
func splitAndTrim(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
