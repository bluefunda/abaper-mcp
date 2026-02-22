package main

// Handlers holds all MCP request handlers
type Handlers struct {
	config    *Config
	apiClient *APIClient
}

// NewHandlers creates a new handlers instance
func NewHandlers(config *Config) *Handlers {
	return &Handlers{
		config:    config,
		apiClient: NewAPIClient(config.AbaperTSURL),
	}
}
