package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

	// Determine mode: stdio (default) or nats
	mode := getEnv("ABAPER_MODE", "stdio")
	fmt.Printf("Starting ABAPER MCP server in %s mode\n", mode)

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
			fmt.Println("Loaded SAP configuration from NATS KV")
			config.ADTHost = sapConfig.Host
			config.ADTClient = sapConfig.Client
			config.ADTUsername = sapConfig.Username
			config.ADTPassword = sapConfig.Password
		} else {
			fmt.Printf("Warning: Failed to load config from NATS KV: %v\n", err)
			fmt.Println("Falling back to environment variables")
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
	fmt.Println("Running in stdio mode (Claude Desktop compatible)")
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// runNATSMode runs the server in NATS-only mode (orchestrator)
func runNATSMode(ctx context.Context, server *mcp.Server, handlers *Handlers) {
	fmt.Println("Running in NATS mode (orchestrator compatible)")

	// Create NATS configuration
	natsConfig := NewNATSConfig()
	natsConfig.EnableMessaging = true

	// Connect to NATS
	natsConn, err := natsConfig.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsConn.Close()

	// Create NATS MCP server
	natsMCP := NewNATSMCPServer(natsConn, handlers, server)

	// Start listening for requests
	if err := natsMCP.Start(); err != nil {
		log.Fatalf("Failed to start NATS MCP server: %v", err)
	}

	// Keep running
	fmt.Println("NATS MCP server is running. Press Ctrl+C to exit.")
	select {}
}

// runDualMode runs the server in both stdio and NATS modes
func runDualMode(ctx context.Context, server *mcp.Server, handlers *Handlers) {
	fmt.Println("Running in dual mode (stdio + NATS)")

	// Start NATS mode in background
	go func() {
		natsConfig := NewNATSConfig()
		natsConfig.EnableMessaging = true
		natsConfig.EnableKV = false // Disable KV for messaging-only connection

		natsConn, err := natsConfig.Connect()
		if err != nil {
			fmt.Printf("Warning: Failed to connect to NATS for messaging: %v\n", err)
			fmt.Println("Continuing with stdio mode only")
			return
		}
		defer natsConn.Close()

		natsMCP := NewNATSMCPServer(natsConn, handlers, server)
		if err := natsMCP.Start(); err != nil {
			fmt.Printf("Warning: Failed to start NATS MCP server: %v\n", err)
		} else {
			fmt.Println("✅ NATS MCP listener started successfully")
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

	fmt.Printf("Running in SSE/HTTP mode (orchestrator compatible via HTTP)\n")

	// Create HTTP multiplexer for routing
	mux := http.NewServeMux()

	// Create Streamable HTTP handler for MCP protocol
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		// Return the MCP server for all requests
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless:      false, // Use sessions for connection management
		JSONResponse:   false, // Use SSE (text/event-stream) instead of JSON
		SessionTimeout: 30 * time.Minute,
	})

	// Mount MCP handler at root path
	mux.Handle("/", handler)

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","version":"` + Version + `","mode":"sse"}`))
	})

	addr := fmt.Sprintf("%s:%s", host, port)
	fmt.Printf("SSE/HTTP MCP server listening on http://%s\n", addr)
	fmt.Printf("Health check available at http://%s/health\n", addr)
	fmt.Println("Press Ctrl+C to exit.")

	// Start HTTP server
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("HTTP server error: %v", err)
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
