package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/samjjx/a0hero/client"
)

// ---------------------------------------------------------------------------
// Config loading tests
// ---------------------------------------------------------------------------

func TestLoadConfig_FileExists(t *testing.T) {
	// Setup: write a valid config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "dev.yaml")
	err := os.WriteFile(configPath, []byte(`name: dev
domain: dev-tenant.auth0.com
client_id: test-client-id
client_secret: test-client-secret
`), 0644)
	require.NoError(t, err)

	// Load reads config from config/<name>.yaml
	cfg, err := client.Load(filepath.Join(dir, "dev"))
	require.NoError(t, err, "Load should succeed when config file exists")
	require.NotNil(t, cfg, "Config should not be nil on success")
	require.Equal(t, "dev", cfg.Name, "Name should match")
	require.Equal(t, "dev-tenant.auth0.com", cfg.Domain, "Domain should match")
	require.Equal(t, "test-client-id", cfg.ClientID, "ClientID should match")
	require.Equal(t, "test-client-secret", cfg.ClientSecret, "ClientSecret should match")
}

func TestLoadConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	// File does not exist — path refers to <name>, not a full path
	_, err := client.Load(filepath.Join(dir, "nonexistent"))
	require.Error(t, err, "Load should return an error when config file does not exist")
}

func TestLoadConfig_EnvVarsOverrideFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "dev.yaml")
	err := os.WriteFile(configPath, []byte(`name: dev
domain: dev-tenant.auth0.com
client_id: file-client-id
client_secret: file-client-secret
`), 0644)
	require.NoError(t, err)

	// Set env vars that should take precedence
	t.Setenv("AUTH0_CLIENT_ID", "env-client-id")
	t.Setenv("AUTH0_CLIENT_SECRET", "env-client-secret")
	t.Setenv("AUTH0_DOMAIN", "env-tenant.auth0.com")

	cfg, err := client.Load(filepath.Join(dir, "dev"))
	require.NoError(t, err, "Load should succeed even with env vars set")
	require.NotNil(t, cfg)
	require.Equal(t, "env-client-id", cfg.ClientID, "AUTH0_CLIENT_ID env var should override file")
	require.Equal(t, "env-client-secret", cfg.ClientSecret, "AUTH0_CLIENT_SECRET env var should override file")
	require.Equal(t, "env-tenant.auth0.com", cfg.Domain, "AUTH0_DOMAIN env var should override file")
}

func TestAvailableTenants_ReturnsCorrectList(t *testing.T) {
	dir := t.TempDir()

	// Create multiple config files
	for _, name := range []string{"dev", "tst", "prod"} {
		err := os.WriteFile(filepath.Join(dir, name+".yaml"), []byte(`name: `+name+`
domain: `+name+`-tenant.auth0.com
client_id: id
client_secret: secret
`), 0644)
		require.NoError(t, err)
	}

	tenants, err := client.AvailableTenants(dir)
	require.NoError(t, err, "AvailableTenants should not error on valid directory")
	require.ElementsMatch(t, []string{"dev", "tst", "prod"}, tenants,
		"AvailableTenants should return all .yaml filenames without extension")
}

func TestAvailableTenants_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	tenants, err := client.AvailableTenants(dir)
	require.NoError(t, err, "AvailableTenants should not error on empty directory")
	require.Empty(t, tenants, "AvailableTenants should return empty slice for empty directory")
}

func TestAvailableTenants_NonExistentDirectory(t *testing.T) {
	_, err := client.AvailableTenants("/nonexistent/path/tenant")
	require.Error(t, err, "AvailableTenants should return an error for non-existent directory")
}

func TestNewClient_MissingClientSecret(t *testing.T) {
	// When client_secret is empty, client construction should fail.
	c, err := client.NewClient("https://dev-tenant.auth0.com", "some-id", "", "test-tenant")
	require.Error(t, err, "NewClient should return an error when client_secret is empty")
	require.Nil(t, c)
}
