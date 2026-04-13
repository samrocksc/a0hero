// Package client provides an authenticated HTTP client for the Auth0 Management API.
package client

import (
	"context"
	"net/http"
	"time"
)

// Config holds the configuration for an Auth0 tenant.
type Config struct {
	Name         string
	Domain       string
	ClientID     string
	ClientSecret string
}

// Load reads a tenant config file from config/<name>.yaml.
// Environment variables AUTH0_CLIENT_ID, AUTH0_CLIENT_SECRET, AUTH0_DOMAIN
// take precedence over the file when set.
func Load(name string) (*Config, error) {
	// TODO: implement
	return nil, nil
}

// AvailableTenants returns the names of all config files in the config directory.
func AvailableTenants(configDir string) ([]string, error) {
	// TODO: implement
	return nil, nil
}

// ---------------------------------------------------------------------------
// Authenticator
// ---------------------------------------------------------------------------

// Authenticator performs client credentials OAuth against Auth0
// and caches tokens with automatic refresh near expiry.
type Authenticator struct {
	// TODO: implement
}

// NewAuthenticator creates a new Authenticator for the given credentials.
func NewAuthenticator(domain, clientID, clientSecret, tenant string) (*Authenticator, error) {
	// TODO: implement
	return nil, nil
}

// GetToken returns a valid access token, fetching or refreshing as needed.
// It is safe for concurrent use.
func (a *Authenticator) GetToken(ctx context.Context) (string, error) {
	// TODO: implement
	return "", nil
}

// SetToken manually sets the cached token and expiry (for testing).
func (a *Authenticator) SetToken(ctx context.Context, token string, expiresAt time.Time) {
	// TODO: implement
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

// Client wraps an HTTP client with Auth0 authentication and provides
// typed methods for Auth0 Management API resources.
type Client struct {
	httpClient *http.Client
	baseURL   string
	auth      *Authenticator
	tenant    string
}

// NewClient creates a new Auth0 API client for the given tenant config.
// It performs an initial token fetch to verify credentials.
func NewClient(baseURL, clientID, clientSecret, tenant string) (*Client, error) {
	// TODO: implement
	return nil, nil
}

// NewClientFromConfig creates a client from a Config struct.
func NewClientFromConfig(baseURL string, cfg *Config) (*Client, error) {
	// TODO: implement
	return nil, nil
}

// Tenant returns the configured tenant name.
func (c *Client) Tenant() string {
	return c.tenant
}

// Get issues a GET request to the given path with the authenticated transport.
func (c *Client) Get(ctx context.Context, path string, result any) error {
	// TODO: implement
	return nil
}

// Post issues a POST request with the given body.
func (c *Client) Post(ctx context.Context, path string, body, result any) error {
	// TODO: implement
	return nil
}

// Patch issues a PATCH request with the given body.
func (c *Client) Patch(ctx context.Context, path string, body, result any) error {
	// TODO: implement
	return nil
}

// Delete issues a DELETE request to the given path.
func (c *Client) Delete(ctx context.Context, path string) error {
	// TODO: implement
	return nil
}

// ---------------------------------------------------------------------------
// Module sub-clients
// ---------------------------------------------------------------------------

// NewUsersClient returns a UsersClient backed by c.
func NewUsersClient(c *Client) *UsersClient {
	return &UsersClient{c: c}
}

// UsersClient wraps the Auth0 /api/v2/users endpoints.
type UsersClient struct {
	c *Client
}

// List returns the first page of users.
func (c *UsersClient) List(ctx context.Context) (any, error) {
	// TODO: implement
	return nil, nil
}

// NewClientsClient returns a ClientsClient backed by c.
func NewClientsClient(c *Client) *ClientsClient {
	return &ClientsClient{c: c}
}

// ClientsClient wraps the Auth0 /api/v2/clients endpoints.
type ClientsClient struct {
	c *Client
}

// List returns the first page of clients.
func (c *ClientsClient) List(ctx context.Context) (any, error) {
	// TODO: implement
	return nil, nil
}

// NewRolesClient returns a RolesClient backed by c.
func NewRolesClient(c *Client) *RolesClient {
	return &RolesClient{c: c}
}

// RolesClient wraps the Auth0 /api/v2/roles endpoints.
type RolesClient struct {
	c *Client
}

// NewConnectionsClient returns a ConnectionsClient backed by c.
func NewConnectionsClient(c *Client) *ConnectionsClient {
	return &ConnectionsClient{c: c}
}

// ConnectionsClient wraps the Auth0 /api/v2/connections endpoints.
type ConnectionsClient struct {
	c *Client
}

// NewLogsClient returns a LogsClient backed by c.
func NewLogsClient(c *Client) *LogsClient {
	return &LogsClient{c: c}
}

// LogsClient wraps the Auth0 /api/v2/logs endpoints.
type LogsClient struct {
	c *Client
}
