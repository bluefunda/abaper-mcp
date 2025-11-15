package main

import (
	"fmt"
	"sync"

	"github.com/bluefunda/abaper/lib"
	"github.com/bluefunda/abaper/types"
)

// Config holds the ABAP connection configuration
type Config struct {
	ADTHost     string
	ADTClient   string
	ADTUsername string
	ADTPassword string
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

// ClientManager manages ADT client connections with caching
type ClientManager struct {
	config *Config
	client types.ADTClient
	mu     sync.RWMutex
}

// NewClientManager creates a new client manager
func NewClientManager(config *Config) *ClientManager {
	return &ClientManager{
		config: config,
	}
}

// GetClient returns a cached ADT client or creates a new one
func (cm *ClientManager) GetClient() (types.ADTClient, error) {
	cm.mu.RLock()
	if cm.client != nil {
		cm.mu.RUnlock()
		return cm.client, nil
	}
	cm.mu.RUnlock()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Double-check after acquiring write lock
	if cm.client != nil {
		return cm.client, nil
	}

	// Validate configuration
	if err := cm.config.Validate(); err != nil {
		return nil, err
	}

	// Create new client
	client, err := lib.CreateADTClient(
		cm.config.ADTHost,
		cm.config.ADTClient,
		cm.config.ADTUsername,
		cm.config.ADTPassword,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ADT client: %w", err)
	}

	cm.client = client
	return client, nil
}

// Reset clears the cached client
func (cm *ClientManager) Reset() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.client = nil
}
