package tests

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/samjjx/a0hero/client"
)

// ---------------------------------------------------------------------------
// Authenticator tests
// ---------------------------------------------------------------------------

func TestAuthenticator_SuccessfulTokenFetchAndCache(t *testing.T) {
	mock := NewAuth0MockServer(t)

	auth, err := client.NewAuthenticator(mock.URL(), "test-client-id", "test-client-secret", "test-tenant")
	require.NoError(t, err)

	// First call should fetch a token
	token1, err := auth.GetToken(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, token1, "GetToken should return a non-empty token")
	require.Equal(t, 1, mock.TokenRequestCount(), "exactly one token request should have been made")

	// Second call should reuse cached token (no new token request)
	token2, err := auth.GetToken(context.Background())
	require.NoError(t, err)
	require.Equal(t, token1, token2, "cached token should be returned")
	require.Equal(t, 1, mock.TokenRequestCount(), "token should be reused from cache, no new request")
}

func TestAuthenticator_TokenReusedWithinExpiryWindow(t *testing.T) {
	mock := NewAuth0MockServer(t)

	auth, err := client.NewAuthenticator(mock.URL(), "test-client-id", "test-client-secret", "test-tenant")
	require.NoError(t, err)

	// Fetch token (populates cache with 1h expiry by default from mock)
	token1, err := auth.GetToken(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, mock.TokenRequestCount())

	// Token should still be cached — no new request
	token2, err := auth.GetToken(context.Background())
	require.NoError(t, err)
	require.Equal(t, token1, token2)
	require.Equal(t, 1, mock.TokenRequestCount(), "no new token request should be made within expiry window")
}

func TestAuthenticator_TokenRefreshedWithin5MinutesOfExpiry(t *testing.T) {
	mock := NewAuth0MockServer(t)

	auth, err := client.NewAuthenticator(mock.URL(), "test-client-id", "test-client-secret", "test-tenant")
	require.NoError(t, err)

	// Seed the authenticator with a token that expires in 4 minutes (within 5-min threshold)
	ctx := context.Background()
	auth.SetToken(ctx, "old-token", time.Now().Add(4*time.Minute))

	// Next call should detect near-expiry and refresh
	newToken, err := auth.GetToken(ctx)
	require.NoError(t, err)
	require.NotEqual(t, "old-token", newToken, "token should be refreshed when within 5 minutes of expiry")
	require.Equal(t, 1, mock.TokenRequestCount(), "a new token request should have been made")
}

func TestAuthenticator_ConcurrentTokenFetchNoDoubleRefresh(t *testing.T) {
	mock := NewAuth0MockServer(t)

	auth, err := client.NewAuthenticator(mock.URL(), "test-client-id", "test-client-secret", "test-tenant")
	require.NoError(t, err)

	// Seed with an expired token so all goroutines will try to refresh
	ctx := context.Background()
	auth.SetToken(ctx, "expired-token", time.Now().Add(-1*time.Hour))

	var wg sync.WaitGroup
	tokenChan := make(chan string, 100)

	// Launch 50 concurrent GetToken calls
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := auth.GetToken(ctx)
			require.NoError(t, err)
			tokenChan <- token
		}()
	}
	wg.Wait()
	close(tokenChan)

	// Collect tokens — all should be the same (single refresh)
	var tokens []string
	for tok := range tokenChan {
		tokens = append(tokens, tok)
	}
	require.Len(t, tokens, 50, "all goroutines should have received a token")
	firstToken := tokens[0]
	for _, tok := range tokens[1:] {
		require.Equal(t, firstToken, tok, "all concurrent calls should receive the same token (no double-refresh)")
	}

	// With double-checked locking, exactly 1 token request should have been made
	require.Equal(t, 1, mock.TokenRequestCount(),
		"double-checked locking should prevent multiple token refreshes; got %d requests", mock.TokenRequestCount())
}

func TestAuthenticator_AuthFailure401(t *testing.T) {
	mock := NewAuth0MockServer(t)

	// Configure token endpoint to return 401
	mock.Handle("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
	})

	auth, err := client.NewAuthenticator(mock.URL(), "bad-client-id", "bad-client-secret", "test-tenant")
	require.NoError(t, err)

	_, err = auth.GetToken(context.Background())
	require.Error(t, err, "401 from token endpoint should return an error")

	// Error should be wrapped as APIError
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok, "error should be of type *client.APIError")
	require.Equal(t, 401, apiErr.StatusCode, "APIError.StatusCode should be 401")
	require.Equal(t, "unauthorized", apiErr.Code, "APIError.Code should be 'unauthorized'")
}

func TestAuthenticator_TokenWithPastExpiryTriggersImmediateRefresh(t *testing.T) {
	mock := NewAuth0MockServer(t)

	auth, err := client.NewAuthenticator(mock.URL(), "test-client-id", "test-client-secret", "test-tenant")
	require.NoError(t, err)

	ctx := context.Background()

	// Manually set a token that expired 1 hour ago
	auth.SetToken(ctx, "past-token", time.Now().Add(-1*time.Hour))

	// First GetToken call should detect expiry and refresh immediately
	token, err := auth.GetToken(ctx)
	require.NoError(t, err)
	require.NotEqual(t, "past-token", token, "expired token should trigger immediate refresh")
	require.Equal(t, 1, mock.TokenRequestCount(), "expired token should cause immediate refresh")
}
