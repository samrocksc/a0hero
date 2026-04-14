// Package client provides an authenticated HTTP client for the Auth0 Management API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/samrocksc/a0hero/logger"
)

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

// Client wraps an HTTP client with Auth0 authentication and provides
// typed methods for Auth0 Management API resources.
type Client struct {
	httpClient *http.Client
	baseURL    string
	auth       *Authenticator
	tenant     string
}

// authTransport injects the Bearer token on every request.
type authTransport struct {
	auth  *Authenticator
	inner http.RoundTripper
}

// RoundTrip implements http.RoundTripper — fetches token and injects Authorization.
func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.auth.GetToken(req.Context())
	if err != nil {
		// Propagate the token fetch error — it is already an *APIError
		return nil, err
	}

	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+token)

	return t.inner.RoundTrip(req)
}

// NewClient creates a new Auth0 API client for the given tenant config.
// It performs an initial token fetch to verify credentials.
func NewClient(baseURL, clientID, clientSecret, tenant string) (*Client, error) {
	if clientSecret == "" {
		return nil, fmt.Errorf("client secret is required")
	}

	auth, err := NewAuthenticator(baseURL, clientID, clientSecret, tenant)
	if err != nil {
		return nil, err
	}

	c := &Client{
		httpClient: &http.Client{
			Transport: &authTransport{auth: auth, inner: http.DefaultTransport},
		},
		baseURL: baseURL,
		auth:    auth,
		tenant:  tenant,
	}

	// Verify credentials by fetching a token upfront
	_, err = auth.GetToken(context.Background())
	if err != nil {
		return nil, err
	}

	return c, nil
}

// NewClientFromConfig creates a Client from a loaded Config.
func NewClientFromConfig(cfg *Config) (*Client, error) {
	if cfg.Domain == "" {
		return nil, fmt.Errorf("domain is required in config")
	}
	baseURL := cfg.Domain
	if baseURL[:4] != "http" {
		baseURL = "https://" + baseURL
	}
	return NewClient(baseURL, cfg.ClientID, cfg.ClientSecret, cfg.Name)
}

// Tenant returns the configured tenant name.
func (c *Client) Tenant() string {
	return c.tenant
}

// BaseURL returns the configured base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// rawRequest performs an HTTP request with the given method and path.
// The path is joined with c.baseURL. result is filled from the response body.
func (c *Client) rawRequest(ctx context.Context, method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error("HTTP request failed", "method", method, "path", path, "error", err)
		return fmt.Errorf("auth transport error: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("failed to read response body", "method", method, "path", path, "error", err)
		return fmt.Errorf("read response body: %w", err)
	}

	logger.Debug("HTTP response", "method", method, "path", path, "status", resp.StatusCode, "bytes", len(respBody))

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Code != "" {
			apiErr.StatusCode = resp.StatusCode
			logger.Warn("API error", "method", method, "path", path, "status", resp.StatusCode, "code", apiErr.Code, "message", apiErr.Message)
			return &apiErr
		}
		// Fallback if we can't parse error body
		logger.Warn("API error (unparseable)", "method", method, "path", path, "status", resp.StatusCode, "body", string(respBody))
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Code:       "unknown",
		}
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal response body: %w", err)
		}
	}

	return nil
}

// Get issues a GET request to the given path with the authenticated transport.
func (c *Client) Get(ctx context.Context, path string, result any) error {
	return c.rawRequest(ctx, http.MethodGet, path, nil, result)
}

// GetWithQuery issues a GET request with query parameters appended to the path.
func (c *Client) GetWithQuery(ctx context.Context, path string, query string, result any) error {
	fullPath := path
	if query != "" {
		fullPath = path + "?" + query
	}
	return c.rawRequest(ctx, http.MethodGet, fullPath, nil, result)
}

// Post issues a POST request with the given body.
func (c *Client) Post(ctx context.Context, path string, body, result any) error {
	return c.rawRequest(ctx, http.MethodPost, path, body, result)
}

// Patch issues a PATCH request with the given body.
func (c *Client) Patch(ctx context.Context, path string, body, result any) error {
	return c.rawRequest(ctx, http.MethodPatch, path, body, result)
}

// Delete issues a DELETE request to the given path.
func (c *Client) Delete(ctx context.Context, path string) error {
	return c.rawRequest(ctx, http.MethodDelete, path, nil, nil)
}
