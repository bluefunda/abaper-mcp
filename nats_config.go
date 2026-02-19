package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

// NATSConfig holds NATS connection configuration
type NATSConfig struct {
	URL              string
	CredsFile        string
	KVBucket         string
	ConnectTimeout   time.Duration
	ReconnectWait    time.Duration
	MaxReconnects    int
	ServerName       string
	EnableKV         bool
	EnableMessaging  bool
}

// NATSConnection wraps NATS connection with helper methods
type NATSConnection struct {
	nc     *nats.Conn
	js     nats.JetStreamContext
	kv     nats.KeyValue
	config *NATSConfig
}

// SAPConfig represents SAP configuration from NATS KV
type SAPConfig struct {
	Host     string `json:"host"`
	Client   string `json:"client"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// NewNATSConfig creates a new NATS configuration from environment variables
func NewNATSConfig() *NATSConfig {
	return &NATSConfig{
		URL:              os.Getenv("NATS_URL"),
		CredsFile:        os.Getenv("NATS_CREDS"),
		KVBucket:         getEnvOrDefault("NATS_KV_BUCKET", "AbaperMCPConfigBucket"),
		ConnectTimeout:   120 * time.Second,
		ReconnectWait:    2 * time.Second,
		MaxReconnects:    60, // Try for 2 minutes
		ServerName:       "abaper-mcp-server",
		EnableKV:         getEnvOrDefault("NATS_ENABLE_KV", "false") == "true",
		EnableMessaging:  getEnvOrDefault("NATS_ENABLE_MESSAGING", "false") == "true",
	}
}

// Connect establishes connection to NATS server
func (nc *NATSConfig) Connect() (*NATSConnection, error) {
	opts := []nats.Option{
		nats.Name(nc.ServerName),
		nats.Timeout(nc.ConnectTimeout),
		nats.ReconnectWait(nc.ReconnectWait),
		nats.MaxReconnects(nc.MaxReconnects),
		nats.DisconnectErrHandler(func(c *nats.Conn, err error) {
			if err != nil {
				fmt.Printf("NATS disconnected: %v\n", err)
			}
		}),
		nats.ReconnectHandler(func(c *nats.Conn) {
			fmt.Printf("NATS reconnected to %s\n", c.ConnectedUrl())
		}),
		nats.ClosedHandler(func(c *nats.Conn) {
			fmt.Println("NATS connection closed")
		}),
	}

	// Add credentials if provided
	if nc.CredsFile != "" {
		fmt.Printf("Using NATS credentials file: %s\n", nc.CredsFile)
		opts = append(opts, nats.UserCredentials(nc.CredsFile))
	} else {
		fmt.Println("Warning: NATS_CREDS not set - attempting anonymous connection")
	}

	// Connect to NATS
	conn, err := nats.Connect(nc.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	fmt.Printf("Successfully connected to NATS at %s\n", nc.URL)

	natsConn := &NATSConnection{
		nc:     conn,
		config: nc,
	}

	// Initialize JetStream if KV or messaging is enabled
	if nc.EnableKV || nc.EnableMessaging {
		js, err := conn.JetStream()
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create JetStream context: %w", err)
		}
		natsConn.js = js
		fmt.Println("JetStream context created")
	}

	// Initialize KV store if enabled
	if nc.EnableKV {
		kv, err := natsConn.js.KeyValue(nc.KVBucket)
		if err != nil {
			// Try to create the bucket if it doesn't exist
			kv, err = natsConn.js.CreateKeyValue(&nats.KeyValueConfig{
				Bucket:      nc.KVBucket,
				Description: "ABAPER MCP configuration storage",
				TTL:         0,                // No expiration
				MaxBytes:    1024 * 1024 * 10, // 10MB max size
			})
			if err != nil {
				// KV store is optional - log warning but don't fail
				fmt.Printf("Warning: Failed to access/create KV bucket: %v\n", err)
				fmt.Println("Continuing without KV store - will use environment variables for configuration")
			} else {
				natsConn.kv = kv
				fmt.Printf("Created KV bucket: %s\n", nc.KVBucket)
			}
		} else {
			natsConn.kv = kv
			fmt.Printf("KV bucket '%s' ready\n", nc.KVBucket)
		}
	}

	return natsConn, nil
}

// Close closes the NATS connection
func (nc *NATSConnection) Close() {
	if nc.nc != nil {
		_ = nc.nc.Drain()
		nc.nc.Close()
		fmt.Println("NATS connection closed")
	}
}

// GetSAPConfig retrieves SAP configuration from NATS KV store
func (nc *NATSConnection) GetSAPConfig() (*SAPConfig, error) {
	if nc.kv == nil {
		return nil, fmt.Errorf("KV store not initialized")
	}

	entry, err := nc.kv.Get("SAPConfig")
	if err != nil {
		if err == nats.ErrKeyNotFound {
			return nil, fmt.Errorf("SAPConfig not found in KV store")
		}
		return nil, fmt.Errorf("failed to get SAPConfig: %w", err)
	}

	var config SAPConfig
	if err := json.Unmarshal(entry.Value(), &config); err != nil {
		return nil, fmt.Errorf("failed to parse SAPConfig: %w", err)
	}

	fmt.Println("Successfully retrieved SAPConfig from NATS KV")
	return &config, nil
}

// PutSAPConfig stores SAP configuration in NATS KV store
func (nc *NATSConnection) PutSAPConfig(config *SAPConfig) error {
	if nc.kv == nil {
		return fmt.Errorf("KV store not initialized")
	}

	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal SAPConfig: %w", err)
	}

	_, err = nc.kv.Put("SAPConfig", data)
	if err != nil {
		return fmt.Errorf("failed to put SAPConfig: %w", err)
	}

	fmt.Println("Successfully stored SAPConfig in NATS KV")
	return nil
}

// GetConfig retrieves generic configuration from NATS KV store
func (nc *NATSConnection) GetConfig(key string) (map[string]interface{}, error) {
	if nc.kv == nil {
		return nil, fmt.Errorf("KV store not initialized")
	}

	entry, err := nc.kv.Get(key)
	if err != nil {
		if err == nats.ErrKeyNotFound {
			return nil, fmt.Errorf("key '%s' not found in KV store", key)
		}
		return nil, fmt.Errorf("failed to get key '%s': %w", key, err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(entry.Value(), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config for key '%s': %w", key, err)
	}

	return config, nil
}

// Subscribe subscribes to a NATS subject with a handler
func (nc *NATSConnection) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	if nc.nc == nil {
		return nil, fmt.Errorf("NATS connection not established")
	}

	sub, err := nc.nc.Subscribe(subject, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to '%s': %w", subject, err)
	}

	fmt.Printf("Subscribed to subject: %s\n", subject)
	return sub, nil
}

// Publish publishes a message to a NATS subject
func (nc *NATSConnection) Publish(subject string, data []byte) error {
	if nc.nc == nil {
		return fmt.Errorf("NATS connection not established")
	}

	return nc.nc.Publish(subject, data)
}

// Request sends a request and waits for a response
func (nc *NATSConnection) Request(subject string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	if nc.nc == nil {
		return nil, fmt.Errorf("NATS connection not established")
	}

	return nc.nc.Request(subject, data, timeout)
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
