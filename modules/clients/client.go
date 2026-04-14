// Package clients provides the Auth0 Clients (Applications) module.
package clients

import (
	"context"
	"fmt"

	"github.com/samrocksc/a0hero/client"
)

// Client represents an Auth0 application/client.
type Client struct {
	ClientID      string   `json:"client_id,omitempty"`
	Name          string   `json:"name"`
	AppType       string   `json:"app_type,omitempty"`
	Description   string   `json:"description,omitempty"`
	Callbacks     []string `json:"callbacks,omitempty"`
	RedirectURIs []string `json:"redirect_uris,omitempty"`
	AllowedOrigins []string `json:"allowed_origins,omitempty"`
	WebOrigins    []string `json:"web_origins,omitempty"`
	LogoURI       string   `json:"logo_uri,omitempty"`
	GrantTypes    []string `json:"grant_types,omitempty"`
}

// Auth0Client wraps the Auth0 /api/v2/clients endpoints.
type Auth0Client struct {
	c *client.Client
}

// New creates a new Clients module client.
func New(c *client.Client) *Auth0Client {
	return &Auth0Client{c: c}
}

// List returns all applications/clients from the Auth0 tenant.
func (ac *Auth0Client) List(ctx context.Context) ([]Client, error) {
	var result struct {
		Clients []Client `json:"clients"`
		Total  int    `json:"total,omitempty"`
	}
	if err := ac.c.GetWithQuery(ctx, "/api/v2/clients", "include_totals=true", &result); err != nil {
		return nil, fmt.Errorf("clients: List: %w", err)
	}
	return result.Clients, nil
}

// Get returns a single client by ID.
func (ac *Auth0Client) Get(ctx context.Context, clientID string) (*Client, error) {
	var result Client
	if err := ac.c.Get(ctx, "/api/v2/clients/"+clientID, &result); err != nil {
		return nil, fmt.Errorf("clients: Get: %w", err)
	}
	return &result, nil
}

// Row converts a Client into a human-friendly table row.
func (c Client) Row() []string {
	appType := c.AppType
	if appType == "" {
		appType = "—"
	}
	return []string{c.ClientID, c.Name, appType}
}

// Columns returns the column headers for a client table.
func Columns() []string {
	return []string{"Client ID", "Name", "Type"}
}