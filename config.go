package main

import (
	"fmt"

	"github.com/bluefunda/abaper/lib"
	"github.com/bluefunda/abaper/types"
)

// Config holds the ABAP connection configuration
type Config struct {
	ADTHost     string
	ADTClient   string
	ADTUsername string
	ADTPassword string
	NATSUrl     string
	NATSCred    string
}

// Validate ensures all required configuration is present
func (c *Config) Validate() error {
	if c.ADTHost == "" {
		return fmt.Errorf("SAP_HOST environment variable is required")
	}
	if c.ADTUsername == "" {
		return fmt.Errorf("SAP_USERNAME environment variable is required")
	}
	if c.ADTPassword == "" {
		return fmt.Errorf("SAP_PASSWORD environment variable is required")
	}
	return nil
}

// ClientManager manages ADT client connections
// In SSE mode, we create a fresh connection for each request to avoid session timeout issues
type ClientManager struct {
	config *Config
}

// NewClientManager creates a new client manager
func NewClientManager(config *Config) *ClientManager {
	return &ClientManager{
		config: config,
	}
}

// GetClient creates a fresh ADT client for each request
// This ensures we never have stale session issues in SSE mode where requests
// can come after long idle periods
func (cm *ClientManager) GetClient() (types.ADTClient, error) {
	// Validate configuration
	if err := cm.config.Validate(); err != nil {
		return nil, err
	}

	// Always create a fresh client - no caching
	// This avoids SAP session timeout issues when requests come after idle periods
	client, err := lib.CreateADTClient(
		cm.config.ADTHost,
		cm.config.ADTClient,
		cm.config.ADTUsername,
		cm.config.ADTPassword,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ADT client: %w", err)
	}

	return client, nil
}

// Reset is a no-op since we don't cache clients anymore
// Kept for API compatibility
func (cm *ClientManager) Reset() {
	// No-op - we don't cache clients anymore
}
