// Package roles provides the Auth0 Roles module.
package roles

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/samrocksc/a0hero/client"
)

// Role represents an Auth0 role.
type Role struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// RoleListResponse wraps the paginated list response.
type RoleListResponse struct {
	Roles []Role `json:"roles"`
	Total int    `json:"total,omitempty"`
}

// Client wraps the Auth0 /api/v2/roles endpoints.
type Client struct {
	c *client.Client
}

// New creates a new Roles module client.
func New(c *client.Client) *Client {
	return &Client{c: c}
}

// List returns all roles from the Auth0 tenant.
func (rc *Client) List(ctx context.Context) ([]Role, error) {
	var raw json.RawMessage
	if err := rc.c.GetWithQuery(ctx, "/api/v2/roles", "include_totals=true", &raw); err != nil {
		return nil, fmt.Errorf("roles: List: %w", err)
	}
	var result RoleListResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		// Auth0 may return a flat array for roles
		var flat []Role
		if err2 := json.Unmarshal(raw, &flat); err2 != nil {
			return nil, fmt.Errorf("roles: List: unmarshal: %w", err)
		}
		return flat, nil
	}
	return result.Roles, nil
}

// Get returns a single role by ID.
func (rc *Client) Get(ctx context.Context, roleID string) (*Role, error) {
	var role Role
	if err := rc.c.Get(ctx, "/api/v2/roles/"+roleID, &role); err != nil {
		return nil, fmt.Errorf("roles: Get: %w", err)
	}
	return &role, nil
}

// Row converts a Role into a human-friendly table row.
func (r Role) Row() []string {
	desc := r.Description
	if desc == "" {
		desc = "—"
	}
	return []string{r.ID, r.Name, desc}
}

// Columns returns the column headers for a role table.
func Columns() []string {
	return []string{"ID", "Name", "Description"}
}
