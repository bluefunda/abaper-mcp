package main

// Handlers holds all MCP request handlers
type Handlers struct {
	config        *Config
	clientManager *ClientManager
}

// NewHandlers creates a new handlers instance
func NewHandlers(config *Config) *Handlers {
	return &Handlers{
		config:        config,
		clientManager: NewClientManager(config),
	}
}
