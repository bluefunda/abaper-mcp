package main

import "fmt"

// Config holds the service configuration
type Config struct {
	AbaperTSURL   string // URL of the abaper-ts ADT backend (e.g., http://abaper-ts:8080)
	S4TemporalURL string // URL of the s4-temporal API (e.g., http://localhost:8081)
	NATSUrl       string
	NATSCred      string
}

// Validate ensures all required configuration is present
func (c *Config) Validate() error {
	if c.AbaperTSURL == "" {
		return fmt.Errorf("ABAPER_TS_URL environment variable is required")
	}
	return nil
}
