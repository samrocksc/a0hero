// Package users provides the Auth0 Users module.
package users

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/samrocksc/a0hero/client"
)

// User represents an Auth0 user.
type User struct {
	ID            string         `json:"user_id,omitempty"`
	Email         string         `json:"email"`
	Name          string         `json:"name"`
	Nickname      string         `json:"nickname,omitempty"`
	Picture       string         `json:"picture,omitempty"`
	CreatedAt     *time.Time     `json:"created_at,omitempty"`
	UpdatedAt     *time.Time     `json:"updated_at,omitempty"`
	LastLogin     *time.Time     `json:"last_login,omitempty"`
	EmailVerified bool           `json:"email_verified"`
	AppMetadata   map[string]any `json:"app_metadata,omitempty"`
	UserMetadata  map[string]any `json:"user_metadata,omitempty"`
	Blocked       bool           `json:"blocked,omitempty"`
	Connections   []string       `json:"identities,omitempty"`
}

// UserListResponse is the response from the Auth0 users list endpoint.
// Auth0 returns users as a flat JSON array — not wrapped.
type UserListResponse struct {
	Users []User `json:"users"`
	Total int    `json:"total,omitempty"`
}

// Client wraps the Auth0 /api/v2/users endpoints.
type Client struct {
	c *client.Client
}

// New creates a new Users module client.
func New(c *client.Client) *Client {
	return &Client{c: c}
}

// List returns users from the Auth0 tenant.
func (uc *Client) List(ctx context.Context, page, perPage int) ([]User, error) {
	query := fmt.Sprintf("page=%d&per_page=%d&include_totals=true", page, perPage)
	var raw json.RawMessage
	if err := uc.c.GetWithQuery(ctx, "/api/v2/users", query, &raw); err != nil {
		return nil, fmt.Errorf("users: List: %w", err)
	}

	// Auth0 returns users as a flat array (with totals in headers or included)
	var users []User
	if err := json.Unmarshal(raw, &users); err != nil {
		return nil, fmt.Errorf("users: List: unmarshal: %w", err)
	}
	return users, nil
}

// Search returns users matching a Lucene query.
func (uc *Client) Search(ctx context.Context, query string, page, perPage int) ([]User, error) {
	apiQuery := fmt.Sprintf("q=%s&page=%d&per_page=%d", query, page, perPage)
	var raw json.RawMessage
	if err := uc.c.GetWithQuery(ctx, "/api/v2/users", apiQuery, &raw); err != nil {
		return nil, fmt.Errorf("users: Search: %w", err)
	}
	var users []User
	if err := json.Unmarshal(raw, &users); err != nil {
		return nil, fmt.Errorf("users: Search: unmarshal: %w", err)
	}
	return users, nil
}

// Get returns a single user by ID.
func (uc *Client) Get(ctx context.Context, userID string) (*User, error) {
	var user User
	if err := uc.c.Get(ctx, "/api/v2/users/"+userID, &user); err != nil {
		return nil, fmt.Errorf("users: Get: %w", err)
	}
	return &user, nil
}

// Row converts a User into a human-friendly table row.
func (u User) Row() []string {
	lastLogin := ""
	if u.LastLogin != nil {
		lastLogin = u.LastLogin.Format("2006-01-02 15:04")
	}
	email := u.Email
	if u.EmailVerified {
		email += " ✓"
	}
	return []string{u.ID, email, u.Name, lastLogin}
}

// Columns returns the column headers for a user table.
func Columns() []string {
	return []string{"ID", "Email", "Name", "Last Login"}
}
