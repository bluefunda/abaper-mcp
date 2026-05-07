package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
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

	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "abaper-mcp",
			Version: Version,
		},
		nil,
	)

	config := &Config{
		AbaperTSURL:   getEnv("ABAPER_TS_URL", "http://localhost:8080"),
		S4TemporalURL: getEnv("S4_TEMPORAL_URL", ""),
	}

	if err := config.Validate(); err != nil {
		logger.L.Fatal("Invalid configuration", zap.Error(err))
	}

	logger.L.Info("Configuration loaded",
		zap.String("abaper_ts_url", config.AbaperTSURL),
		zap.String("s4_temporal_url", config.S4TemporalURL),
	)

	handlers := NewHandlers(config)

	registerTools(server, handlers)
	registerResources(server, handlers)
	registerPrompts(server, handlers)

	ctx := context.Background()

	switch mode {
	case "sse":
		runSSEMode(ctx, server)
	default: // stdio
		runStdioMode(ctx, server)
	}
}

func runStdioMode(ctx context.Context, server *mcp.Server) {
	logger.L.Info("Running in stdio mode")
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		logger.L.Fatal("Server error", zap.Error(err))
	}
}

func runSSEMode(ctx context.Context, server *mcp.Server) {
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
			return server
		}, nil)
	} else {
		handler = mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
			logger.L.Debug("HTTP client connected",
				zap.String("remote_addr", req.RemoteAddr),
				zap.String("path", req.URL.Path),
			)
			return server
		}, &mcp.StreamableHTTPOptions{
			Stateless:      false,
			JSONResponse:   false,
			SessionTimeout: 30 * time.Minute,
		})
	}

	mux := http.NewServeMux()
	mux.Handle("/", handler)
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

	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.L.Fatal("HTTP server error", zap.Error(err))
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
