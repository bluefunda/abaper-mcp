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
