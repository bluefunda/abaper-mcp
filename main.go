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
	// Check for version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("abaper-mcp version %s\n", Version)
		fmt.Printf("Build time: %s\n", BuildTime)
		fmt.Printf("Git commit: %s\n", GitCommit)
		os.Exit(0)
	}

	// Initialize structured logging
	logLevel := getEnv("LOG_LEVEL", "info")
	logFormat := getEnv("LOG_FORMAT", "json") // json or console
	logger.Init(logger.Config{
		Level:      logLevel,
		Format:     logFormat,
		ServerName: "abaper-mcp",
		Version:    Version,
	})
	defer logger.Sync()

	// Determine mode: stdio (default) or nats
	mode := getEnv("ABAPER_MODE", "stdio")
	logger.L.Info("Starting ABAPER MCP server",
		zap.String("mode", mode),
		zap.String("log_level", logLevel),
	)

	// Create MCP server
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "abaper-mcp",
			Version: Version,
		},
		nil,
	)

	// Initialize ABAP client configuration
	config := &Config{
		ADTHost:     getEnv("SAP_HOST", ""),
		ADTClient:   getEnv("SAP_CLIENT", "100"),
		ADTUsername: getEnv("SAP_USERNAME", ""),
		ADTPassword: getEnv("SAP_PASSWORD", ""),
	}

	// Try to load config from NATS KV if enabled
	if getEnv("NATS_ENABLE_KV", "false") == "true" {
		if sapConfig, err := loadConfigFromNATS(); err == nil {
			logger.L.Info("Loaded SAP configuration from NATS KV")
			config.ADTHost = sapConfig.Host
			config.ADTClient = sapConfig.Client
			config.ADTUsername = sapConfig.Username
			config.ADTPassword = sapConfig.Password
		} else {
			logger.L.Warn("Failed to load config from NATS KV, falling back to environment variables",
				zap.Error(err),
			)
		}
	}

	// Create handlers with config
	handlers := NewHandlers(config)

	// Register Tools
	registerTools(server, handlers)

	// Register Resources
	registerResources(server, handlers)

	// Register Prompts
	registerPrompts(server, handlers)

	ctx := context.Background()

	// Run in appropriate mode
	switch mode {
	case "nats":
		runNATSMode(ctx, server, handlers)
	case "dual":
		runDualMode(ctx, server, handlers)
	case "sse":
		runSSEMode(ctx, server)
	default: // stdio
		runStdioMode(ctx, server)
	}
}

// runStdioMode runs the server in stdio mode (Claude Desktop)
func runStdioMode(ctx context.Context, server *mcp.Server) {
	logger.L.Info("Running in stdio mode (Claude Desktop compatible)")
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		logger.L.Fatal("Server error", zap.Error(err))
	}
}

// runNATSMode runs the server in NATS-only mode (orchestrator)
func runNATSMode(ctx context.Context, server *mcp.Server, handlers *Handlers) {
	logger.L.Info("Running in NATS mode (orchestrator compatible)")

	// Create NATS configuration
	natsConfig := NewNATSConfig()
	natsConfig.EnableMessaging = true

	// Connect to NATS
	natsConn, err := natsConfig.Connect()
	if err != nil {
		logger.L.Fatal("Failed to connect to NATS", zap.Error(err))
	}
	defer natsConn.Close()

	logger.L.Info("Connected to NATS server")

	// Create NATS MCP server
	natsMCP := NewNATSMCPServer(natsConn, handlers, server)

	// Start listening for requests
	if err := natsMCP.Start(); err != nil {
		logger.L.Fatal("Failed to start NATS MCP server", zap.Error(err))
	}

	// Keep running
	logger.L.Info("NATS MCP server is running")
	select {}
}

// runDualMode runs the server in both stdio and NATS modes
func runDualMode(ctx context.Context, server *mcp.Server, handlers *Handlers) {
	logger.L.Info("Running in dual mode (stdio + NATS)")

	// Start NATS mode in background
	go func() {
		natsConfig := NewNATSConfig()
		natsConfig.EnableMessaging = true
		natsConfig.EnableKV = false // Disable KV for messaging-only connection

		natsConn, err := natsConfig.Connect()
		if err != nil {
			logger.L.Warn("Failed to connect to NATS for messaging, continuing with stdio mode only",
				zap.Error(err),
			)
			return
		}
		defer natsConn.Close()

		natsMCP := NewNATSMCPServer(natsConn, handlers, server)
		if err := natsMCP.Start(); err != nil {
			logger.L.Warn("Failed to start NATS MCP server", zap.Error(err))
		} else {
			logger.L.Info("NATS MCP listener started successfully")
		}

		// Keep NATS connection alive
		select {}
	}()

	// Run stdio mode in foreground
	runStdioMode(ctx, server)
}

// runSSEMode runs the server in SSE/HTTP mode (for orchestrator via HTTP)
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
		logger.L.Info("Using legacy SSE transport for Claude Code compatibility")
		handler = mcp.NewSSEHandler(func(req *http.Request) *mcp.Server {
			logger.L.Debug("SSE client connected",
				zap.String("remote_addr", req.RemoteAddr),
				zap.String("method", req.Method),
				zap.String("path", req.URL.Path),
			)
			return server
		}, nil)
	} else {
		logger.L.Info("Using Streamable HTTP transport")
		handler = mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
			logger.L.Debug("HTTP client connected",
				zap.String("remote_addr", req.RemoteAddr),
				zap.String("method", req.Method),
				zap.String("path", req.URL.Path),
			)
			return server
		}, &mcp.StreamableHTTPOptions{
			Stateless:      false,
			JSONResponse:   false,
			SessionTimeout: 30 * time.Minute,
		})
	}

	// Create HTTP multiplexer for routing
	mux := http.NewServeMux()
	mux.Handle("/", handler)

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","version":"` + Version + `","mode":"sse"}`))
	})

	addr := fmt.Sprintf("%s:%s", host, port)
	logger.L.Info("SSE/HTTP MCP server listening",
		zap.String("address", "http://"+addr),
		zap.String("health_check", "http://"+addr+"/health"),
	)

	// Start HTTP server
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.L.Fatal("HTTP server error", zap.Error(err))
	}
}

// loadConfigFromNATS loads configuration from NATS KV store
func loadConfigFromNATS() (*SAPConfig, error) {
	natsConfig := NewNATSConfig()
	natsConfig.EnableKV = true

	natsConn, err := natsConfig.Connect()
	if err != nil {
		return nil, err
	}
	defer natsConn.Close()

	return natsConn.GetSAPConfig()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
