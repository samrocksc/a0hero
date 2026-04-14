// Package client provides an authenticated HTTP client for the Auth0 Management API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/samrocksc/a0hero/logger"
)

// ---------------------------------------------------------------------------
// Authenticator
// ---------------------------------------------------------------------------

// Authenticator performs client credentials OAuth against Auth0
// and caches tokens with automatic refresh near expiry.
type Authenticator struct {
	domain       string
	clientID     string
	clientSecret string
	tenant       string
	httpClient   *http.Client

	mu          sync.RWMutex
	cachedToken string
	expiresAt   time.Time
}

// NewAuthenticator creates a new Authenticator for the given credentials.
func NewAuthenticator(domain, clientID, clientSecret, tenant string) (*Authenticator, error) {
	if clientSecret == "" {
		return nil, fmt.Errorf("client secret is required")
	}
	logger.Info("authenticator created", "domain", domain, "tenant", tenant)
	return &Authenticator{
		domain:       domain,
		clientID:     clientID,
		clientSecret: clientSecret,
		tenant:       tenant,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// GetToken returns a valid access token, fetching or refreshing as needed.
// It is safe for concurrent use via double-checked locking.
func (a *Authenticator) GetToken(ctx context.Context) (string, error) {
	// Fast path: check cache without lock
	a.mu.RLock()
	if a.cachedToken != "" && time.Now().Add(5*time.Minute).Before(a.expiresAt) {
		token := a.cachedToken
		a.mu.RUnlock()
		return token, nil
	}
	a.mu.RUnlock()

	// Slow path: refresh if needed (double-checked locking)
	a.mu.Lock()
	defer a.mu.Unlock()

	// Re-check after acquiring write lock
	if a.cachedToken != "" && time.Now().Add(5*time.Minute).Before(a.expiresAt) {
		return a.cachedToken, nil
	}

	token, expiresAt, err := a.refresh(ctx)
	if err != nil {
		logger.Error("token refresh failed", "domain", a.domain, "error", err)
		return "", err
	}
	a.cachedToken = token
	a.expiresAt = expiresAt
	logger.Info("token refreshed", "domain", a.domain, "expires_at", expiresAt.Format(time.RFC3339))
	return token, nil
}

// refresh fetches a new token from Auth0's oauth/token endpoint.
func (a *Authenticator) refresh(ctx context.Context) (string, time.Time, error) {
	reqBody := map[string]string{
		"client_id":     a.clientID,
		"client_secret": a.clientSecret,
		"grant_type":    "client_credentials",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/oauth/token", a.domain)
	logger.Debug("requesting token", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		logger.Error("token request network error", "url", url, "error", err)
		return "", time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warn("token request failed", "url", url, "status", resp.StatusCode)
		var errResp struct {
			StatusCode int    `json:"statusCode"`
			Error      string `json:"error"`
			Message    string `json:"message"`
		}
		if decodeErr := json.NewDecoder(resp.Body).Decode(&errResp); decodeErr == nil {
			return "", time.Time{}, &APIError{
				StatusCode: errResp.StatusCode,
				Code:       errResp.Error,
				Message:    errResp.Message,
			}
		}
		// Fallback: use response status code when no parseable body
		return "", time.Time{}, &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("token request failed with status %d", resp.StatusCode),
			Code:       "unknown",
		}
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", time.Time{}, fmt.Errorf("decode token response: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	return tokenResp.AccessToken, expiresAt, nil
}

// SetToken manually sets the cached token and expiry (for testing).
func (a *Authenticator) SetToken(ctx context.Context, token string, expiresAt time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cachedToken = token
	a.expiresAt = expiresAt
}
