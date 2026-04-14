// Package connections provides the Auth0 Connections module.
package connections

import (
	"context"
	"fmt"

	"github.com/samrocksc/a0hero/client"
)

// Connection represents an Auth0 connection.
type Connection struct {
	ID             string         `json:"id,omitempty"`
	Name           string         `json:"name"`
	Strategy       string         `json:"strategy"`
	Options        map[string]any `json:"options,omitempty"`
	EnabledClients []string       `json:"enabled_clients,omitempty"`
}

// ConnClient wraps the Auth0 /api/v2/connections endpoints.
type ConnClient struct {
	c *client.Client
}

// New creates a new Connections module client.
func New(c *client.Client) *ConnClient {
	return &ConnClient{c: c}
}

// List returns all connections from the Auth0 tenant.
func (cc *ConnClient) List(ctx context.Context) ([]Connection, error) {
	var result struct {
		Connections []Connection `json:"connections"`
		Total      int          `json:"total,omitempty"`
	}
	if err := cc.c.GetWithQuery(ctx, "/api/v2/connections", "include_totals=true", &result); err != nil {
		return nil, fmt.Errorf("connections: List: %w", err)
	}
	return result.Connections, nil
}

// Get returns a single connection by ID.
func (cc *ConnClient) Get(ctx context.Context, connectionID string) (*Connection, error) {
	var conn Connection
	if err := cc.c.Get(ctx, "/api/v2/connections/"+connectionID, &conn); err != nil {
		return nil, fmt.Errorf("connections: Get: %w", err)
	}
	return &conn, nil
}

// Row converts a Connection into a human-friendly table row.
func (c Connection) Row() []string {
	enabled := fmt.Sprintf("%d clients", len(c.EnabledClients))
	return []string{c.ID, c.Name, c.Strategy, enabled}
}

// Columns returns the column headers for a connection table.
func Columns() []string {
	return []string{"ID", "Name", "Strategy", "Enabled"}
}