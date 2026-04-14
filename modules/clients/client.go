// Package clients provides the Auth0 Clients (Applications) module.
package clients

import (
	"context"
	"fmt"

	"github.com/samrocksc/a0hero/client"
	"github.com/samrocksc/a0hero/modules/edit"
)

// Client represents an Auth0 application/client.
type Client struct {
	ClientID       string   `json:"client_id,omitempty"`
	Name           string   `json:"name"`
	AppType        string   `json:"app_type,omitempty"`
	Description    string   `json:"description,omitempty"`
	Callbacks      []string `json:"callbacks,omitempty"`
	RedirectURIs  []string `json:"redirect_uris,omitempty"`
	AllowedOrigins []string `json:"allowed_origins,omitempty"`
	WebOrigins     []string `json:"web_origins,omitempty"`
	LogoURI        string   `json:"logo_uri,omitempty"`
	GrantTypes     []string `json:"grant_types,omitempty"`
	IsFirstParty   bool     `json:"is_first_party,omitempty"`
	IsGlobal       bool     `json:"is_global,omitempty"`
	LoginURI       string   `json:"login_uri,omitempty"`
	LoginOrigin    string   `json:"login_origin,omitempty"`
	LogoutURLs     []string `json:"logout_urls,omitempty"`
	CustomLoginPagePreview string `json:"custom_login_page_preview,omitempty"`
	AllowedClients []string `json:"allowed_clients,omitempty"`
	Mobile         string   `json:"mobile,omitempty"`
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

// EntityService implementation for edit.EditSession

// EntityType returns the entity type name.
func (ac *Auth0Client) EntityType() string {
	return "client"
}

// GetFields returns the field definitions for clients.
func (ac *Auth0Client) GetFields() []edit.FieldDef {
	return ClientFields
}

// Fetch retrieves the current state of a client.
func (ac *Auth0Client) Fetch(ctx context.Context, id string) (map[string]interface{}, error) {
	c, err := ac.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Convert to map
	result := map[string]interface{}{
		"client_id":                  c.ClientID,
		"name":                      c.Name,
		"app_type":                  c.AppType,
		"description":               c.Description,
		"callbacks":                 c.Callbacks,
		"redirect_uris":             c.RedirectURIs,
		"web_origins":               c.WebOrigins,
		"allowed_origins":           c.AllowedOrigins,
		"logo_uri":                  c.LogoURI,
		"grant_types":               c.GrantTypes,
		"is_first_party":            c.IsFirstParty,
		"is_global":                 c.IsGlobal,
		"login_uri":                 c.LoginURI,
		"login_origin":              c.LoginOrigin,
		"logout_urls":              c.LogoutURLs,
		"custom_login_page_preview": c.CustomLoginPagePreview,
		"allowed_clients":           c.AllowedClients,
		"mobile":                    c.Mobile,
	}

	return result, nil
}

// Update applies changes to a client and returns the updated entity.
func (ac *Auth0Client) Update(ctx context.Context, id string, changes map[string]interface{}) (map[string]interface{}, error) {
	// PATCH /api/v2/clients/{id}
	var result Client
	if err := ac.c.Patch(ctx, "/api/v2/clients/"+id, changes, &result); err != nil {
		return nil, fmt.Errorf("clients: Update: %w", err)
	}

	// Return as map
	return ac.makeClientMap(&result), nil
}

// makeClientMap converts a Client to a generic map.
func (ac *Auth0Client) makeClientMap(c *Client) map[string]interface{} {
	return map[string]interface{}{
		"client_id":                  c.ClientID,
		"name":                      c.Name,
		"app_type":                  c.AppType,
		"description":               c.Description,
		"callbacks":                 c.Callbacks,
		"redirect_uris":              c.RedirectURIs,
		"web_origins":               c.WebOrigins,
		"allowed_origins":           c.AllowedOrigins,
		"logo_uri":                  c.LogoURI,
		"grant_types":               c.GrantTypes,
		"is_first_party":            c.IsFirstParty,
		"is_global":                 c.IsGlobal,
		"login_uri":                 c.LoginURI,
		"login_origin":              c.LoginOrigin,
		"logout_urls":               c.LogoutURLs,
		"custom_login_page_preview": c.CustomLoginPagePreview,
		"allowed_clients":           c.AllowedClients,
		"mobile":                    c.Mobile,
	}
}