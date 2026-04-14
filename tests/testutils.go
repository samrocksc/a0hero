// Package tests holds integration-style tests for the client package.
// It uses httptest to mock the Auth0 API and testify/require for assertions.
package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock Auth0 server helpers
// ---------------------------------------------------------------------------

// Auth0MockServer simulates Auth0's API for testing.
// It records requests and returns configurable responses.
type Auth0MockServer struct {
	ts          *httptest.Server
	mu          sync.Mutex
	requests    []RecordedRequest
	token       string
	tokenExpiry time.Time
	tokenReqs   int                         // count of token requests received
	handlers    map[string]http.HandlerFunc // custom handlers per path
}

// RecordedRequest captures incoming requests for assertion.
type RecordedRequest struct {
	Method string
	Path   string
	Header http.Header
	Body   []byte
}

// NewAuth0MockServer creates a mock Auth0 API server.
func NewAuth0MockServer(t *testing.T) *Auth0MockServer {
	m := &Auth0MockServer{
		token:       "test-access-token-valid-for-1h",
		tokenExpiry: time.Now().Add(1 * time.Hour),
		handlers:    make(map[string]http.HandlerFunc),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		m.captureRequest(r)

		if h, ok := m.handlers[r.URL.Path]; ok {
			h(w, r)
			return
		}

		if r.URL.Path == "/oauth/token" && r.Method == http.MethodPost {
			m.handleToken(w, r)
			return
		}

		http.NotFound(w, r)
	})

	m.ts = httptest.NewServer(mux)
	t.Cleanup(m.ts.Close)

	return m
}

// Handle registers a custom handler for a given path.
func (m *Auth0MockServer) Handle(path string, h http.HandlerFunc) {
	m.handlers[path] = h
}

// URL returns the base URL of the mock server.
func (m *Auth0MockServer) URL() string {
	return m.ts.URL
}

// Requests returns all recorded requests.
func (m *Auth0MockServer) Requests() []RecordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requests
}

// TokenRequestCount returns how many times the token endpoint was called.
func (m *Auth0MockServer) TokenRequestCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tokenReqs
}

// SetTokenExpiry sets when the mock server thinks the token expires.
func (m *Auth0MockServer) SetTokenExpiry(expiry time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokenExpiry = expiry
}

// SetExpiredToken makes subsequent token requests return an already-expired token.
func (m *Auth0MockServer) SetExpiredToken(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token = token
	m.tokenExpiry = time.Now().Add(-1 * time.Hour)
}

func (m *Auth0MockServer) captureRequest(r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	body, _ := io.ReadAll(r.Body)
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))

	m.requests = append(m.requests, RecordedRequest{
		Method: r.Method,
		Path:   r.URL.Path,
		Header: r.Header,
		Body:   body,
	})
}

func (m *Auth0MockServer) handleToken(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	m.tokenReqs++
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"access_token": m.token,
		"token_type":   "Bearer",
		"expires_in":   int(m.tokenExpiry.Sub(time.Now()).Seconds()),
	})
}

// RequireAuthHeader asserts all non-token requests carry a Bearer token.
func (m *Auth0MockServer) RequireAuthHeader(t *testing.T, token string) {
	for _, req := range m.Requests() {
		if req.Path == "/oauth/token" {
			continue
		}
		require.Equal(t, fmt.Sprintf("Bearer %s", token), req.Header.Get("Authorization"),
			"request to %s %s missing or had wrong Authorization header", req.Method, req.Path)
	}
}

// RequireBearerToken asserts a specific request had the expected Bearer token.
func (m *Auth0MockServer) RequireBearerToken(t *testing.T, path, method, token string) {
	for _, req := range m.Requests() {
		if req.Path == path && req.Method == method {
			require.Equal(t, fmt.Sprintf("Bearer %s", token), req.Header.Get("Authorization"),
				"Authorization header mismatch for %s %s", method, path)
			return
		}
	}
	t.Errorf("no request found for %s %s", method, path)
}

// RespondWithAuth0Error registers a handler that returns an Auth0-formatted error.
func (m *Auth0MockServer) RespondWithAuth0Error(path string, status int, errCode, message string) {
	m.handlers[path] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		// Marshal first then write to avoid buffering issues with wrapped ResponseWriter
		body, _ := json.Marshal(map[string]any{
			"statusCode": status,
			"error":      errCode,
			"message":    message,
		})
		w.Write(body)
	}
}

// RespondWithJSON registers a handler that returns the given status and JSON body.
// The handler marshals and writes the body directly to avoid buffering with wrapped ResponseWriter.
func (m *Auth0MockServer) RespondWithJSON(path string, status int, body any) {
	data, err := json.Marshal(body)
	if err != nil {
		data = []byte(fmt.Sprintf(`{"error": "marshal failed: %v"}`, err))
	}
	m.handlers[path] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(data)
	}
}
