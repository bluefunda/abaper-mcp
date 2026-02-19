package main

import (
	"os"

	vault "github.com/bluefunda/go-vault"
)

// loadSecretsFromVault loads SAP credentials and NATS config from Vault, overriding env-based values.
func (c *Config) loadSecretsFromVault() error {
	vc, err := vault.NewClientFromEnv()
	if err != nil {
		return err
	}

	// Service secrets: SAP config + NATS URL
	svcSecrets, err := vc.GetSecret("apps/bluefunda-ai/services/abaper-mcp")
	if err == nil {
		if v, ok := svcSecrets["sap_host"]; ok && v != "" {
			c.ADTHost = v
		}
		if v, ok := svcSecrets["sap_client"]; ok && v != "" {
			c.ADTClient = v
		}
		if v, ok := svcSecrets["sap_username"]; ok && v != "" {
			c.ADTUsername = v
		}
		if v, ok := svcSecrets["sap_password"]; ok && v != "" {
			c.ADTPassword = v
		}
	}

	// NATS URL from shared infra config (single source of truth for all services)
	natsConfig, err := vc.GetSecret("infra/nats/config")
	if err == nil {
		if v, ok := natsConfig["url"]; ok && v != "" {
			c.NATSUrl = v
		}
	}

	// NATS credentials from account-based path (individual account)
	credsContent, err := vc.GetField("infra/nats/creds/individual/admin", "creds_file")
	if err == nil && credsContent != "" {
		tmpFile, err := os.CreateTemp("", "nats-creds-*.creds")
		if err == nil {
			if _, err := tmpFile.WriteString(credsContent); err != nil {
				_ = tmpFile.Close()
				_ = os.Remove(tmpFile.Name())
			} else {
				_ = tmpFile.Close()
				c.NATSCred = tmpFile.Name()
			}
		}
	}

	return nil
}
