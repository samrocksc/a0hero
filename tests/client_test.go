package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/samrocksc/a0hero/client"
	usermod "github.com/samrocksc/a0hero/modules/users"
	clientmod "github.com/samrocksc/a0hero/modules/clients"
)

// ---------------------------------------------------------------------------
// Client HTTP tests
// ---------------------------------------------------------------------------

func TestClient_RequestIncludesAuthorizationHeader(t *testing.T) {
	mock := NewAuth0MockServer(t)

	// Register a handler that returns an empty user list
	mock.RespondWithJSON("/api/v2/users", http.StatusOK, map[string]any{
		"users": []map[string]any{},
	})

	// Create authenticated client pointing at mock
	c, err := client.NewClient(mock.URL(), "test-client-id", "test-client-secret", "test-tenant")
	require.NoError(t, err)

	// Make a request through the client
	err = c.Get(context.Background(), "/api/v2/users", nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, mock.TokenRequestCount(), 1, "a token should have been acquired")

	// Assert that the API request included the Authorization header
	mock.RequireBearerToken(t, "/api/v2/users", http.MethodGet, mock.token)
}

func TestClient_RequestsGoToCorrectBaseURLAndPath(t *testing.T) {
	mock := NewAuth0MockServer(t)

	var receivedPath string
	mock.Handle("/api/v2/clients", func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		mock.RespondWithJSON("/api/v2/clients", http.StatusOK, map[string]any{
			"clients": []map[string]any{},
		})
	})

	c, err := client.NewClient(mock.URL(), "test-client-id", "test-client-secret", "test-tenant")
	require.NoError(t, err)

	err = c.Get(context.Background(), "/api/v2/clients", nil)
	require.NoError(t, err)

	require.Equal(t, "/api/v2/clients", receivedPath, "request should go to the exact path specified")
}

func TestClient_Auth0Error4xx_ReturnsAPIError(t *testing.T) {
	mock := NewAuth0MockServer(t)

	mock.RespondWithAuth0Error("/api/v2/clients/non-existent-id", http.StatusNotFound, "resource_not_found", "Client not found")

	c, err := client.NewClient(mock.URL(), "test-client-id", "test-client-secret", "test-tenant")
	require.NoError(t, err)

	err = c.Get(context.Background(), "/api/v2/clients/non-existent-id", nil)
	require.Error(t, err, "4xx response should be returned as an error")

	apiErr, ok := err.(*client.APIError)
	require.True(t, ok, "error should be of type *client.APIError")
	require.Equal(t, 404, apiErr.StatusCode, "StatusCode should be 404")
	require.Equal(t, "resource_not_found", apiErr.Code, "Code should match Auth0 error code")
	require.NotEmpty(t, apiErr.Message, "Message should not be empty")
}

func TestClient_Auth0Error5xx_ReturnsAPIError(t *testing.T) {
	mock := NewAuth0MockServer(t)

	mock.RespondWithAuth0Error("/api/v2/users", http.StatusInternalServerError, "internal_error", "Internal server error")

	c, err := client.NewClient(mock.URL(), "test-client-id", "test-client-secret", "test-tenant")
	require.NoError(t, err)

	err = c.Get(context.Background(), "/api/v2/users", nil)
	require.Error(t, err, "5xx response should be returned as an error")

	apiErr, ok := err.(*client.APIError)
	require.True(t, ok, "error should be of type *client.APIError")
	require.Equal(t, 500, apiErr.StatusCode, "StatusCode should be 500")
	require.Equal(t, "internal_error", apiErr.Code, "Code should be 'internal_error'")
}

func TestClient_NetworkError_Propagates(t *testing.T) {
	mock := NewAuth0MockServer(t)
	c, err := client.NewClient(mock.URL(), "test-client-id", "test-client-secret", "test-tenant")
	require.NoError(t, err)

	// Close the server to simulate network error
	mock.ts.Close()

	err = c.Get(context.Background(), "/api/v2/users", nil)
	require.Error(t, err, "network error should propagate as an error")
}

func TestClient_AuthFailure401_PropagatesAsAPIError(t *testing.T) {
	mock := NewAuth0MockServer(t)

	// Token endpoint returns 401
	mock.Handle("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"statusCode": 401,
			"error":      "unauthorized",
			"message":    "Invalid client credentials",
		})
	})

	// NewClient fails because token validation happens during construction
	_, err := client.NewClient(mock.URL(), "bad-id", "bad-secret", "test-tenant")
	require.Error(t, err, "NewClient should fail when token endpoint returns 401")

	apiErr, ok := err.(*client.APIError)
	require.True(t, ok, "auth failure should be *client.APIError, got %T: %v", err, err)
	require.Equal(t, 401, apiErr.StatusCode)
	require.Equal(t, "unauthorized", apiErr.Code)
}

func TestClient_ModuleClientsUseSharedAuth(t *testing.T) {
	mock := NewAuth0MockServer(t)

	// Register handlers for multiple module endpoints
	var usersCalled, clientsCalled bool
	mock.Handle("/api/v2/users", func(w http.ResponseWriter, r *http.Request) {
		usersCalled = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	})
	mock.Handle("/api/v2/clients", func(w http.ResponseWriter, r *http.Request) {
		clientsCalled = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	})

	c, err := client.NewClient(mock.URL(), "test-client-id", "test-client-secret", "test-tenant")
	require.NoError(t, err)

	// Use module clients (they all share the same authenticated http.Client)
	users := usermod.New(c)
	clients := clientmod.New(c)

	_, err = users.List(context.Background(), 0, 25)
	require.NoError(t, err)
	_, err = clients.List(context.Background())
	require.NoError(t, err)

	require.True(t, usersCalled, "/api/v2/users should have been called via UsersClient")
	require.True(t, clientsCalled, "/api/v2/clients should have been called via ClientsClient")

	// Both module calls should have used the same token (only 1 token request)
	require.Equal(t, 1, mock.TokenRequestCount(),
		"both modules should share the same token cache; expected 1 token request, got %d", mock.TokenRequestCount())
}