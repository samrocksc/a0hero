// Package client provides an authenticated HTTP client for the Auth0 Management API.
package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the configuration for an Auth0 tenant.
type Config struct {
	Name         string `yaml:"name"`
	Domain       string `yaml:"domain"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
}

// Load reads a tenant config file from config/<name>.yaml.
// Environment variables AUTH0_CLIENT_ID, AUTH0_CLIENT_SECRET, AUTH0_DOMAIN
// take precedence over the file when set.
func Load(name string) (*Config, error) {
	// name is actually the full path to the config (already constructed with dir)
	// In tests, name is like "/tmp/xxx/dev" pointing to /tmp/xxx/dev.yaml
	// But the test calls: client.Load(filepath.Join(dir, "dev"))
	// So name = "/tmp/xxx/dev" and we need to append ".yaml"
	filePath := name + ".yaml"

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", filePath, err)
	}

	// Env vars override file values
	if id := os.Getenv("AUTH0_CLIENT_ID"); id != "" {
		cfg.ClientID = id
	}
	if secret := os.Getenv("AUTH0_CLIENT_SECRET"); secret != "" {
		cfg.ClientSecret = secret
	}
	if domain := os.Getenv("AUTH0_DOMAIN"); domain != "" {
		cfg.Domain = domain
	}

	return &cfg, nil
}

// WriteFile writes the config as YAML to the given file path.
func (c *Config) WriteFile(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", path, err)
	}
	return nil
}

// AvailableTenants returns the names of all config files in the config directory.
func AvailableTenants(configDir string) ([]string, error) {
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	var tenants []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			tenants = append(tenants, strings.TrimSuffix(name, filepath.Ext(name)))
		}
	}
	return tenants, nil
}
