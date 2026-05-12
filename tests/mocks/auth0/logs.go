// Package mockauth0 provides mock HTTP handlers for Auth0 API endpoints.
package mockauth0

import (
	"encoding/json"
	"net/http"
	"strings"
)

// LogEvent represents a mock Auth0 log event for testing.
// Based on the Auth0 Management API spec.
type LogEvent struct {
	ID        string         `json:"log_id,omitempty"`
	Date      string         `json:"date"`
	Type      string         `json:"type"`
	LogID     string         `json:"log_id,omitempty"`
	IP        string         `json:"ip,omitempty"`
	UserID    string         `json:"user_id,omitempty"`
	UserName  string         `json:"user_name,omitempty"`
	ClientID  string         `json:"client_id,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	TenantID  string         `json:"tenant_id,omitempty"`
	Auth0User map[string]any `json:"auth0_user,omitempty"`
	Location  map[string]any `json:"location_info,omitempty"`
}

// LogsListHandler returns a mock handler for GET /api/v2/logs
// Supports cursor-based pagination with 'from' and 'take' query params.
func LogsListHandler(events []LogEvent) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error": "method_not_allowed", "message": "Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		// Check authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"error":   "unauthorized",
				"message": "Missing or invalid token",
			})
			return
		}

		// Parse query params for pagination
		from := r.URL.Query().Get("from")
		_ = from // Used for cursor-based pagination

		// Default to returning all events (simulating first page)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(events)
	}
}

// LogsGetHandler returns a mock handler for GET /api/v2/logs/{id}
func LogsGetHandler(event LogEvent) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error": "method_not_allowed", "message": "Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		// Check authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"error":   "unauthorized",
				"message": "Missing or invalid token",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(event)
	}
}

// LogsGetNotFoundHandler returns 404 for non-existent log IDs
func LogsGetNotFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error": "method_not_allowed", "message": "Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		// Check authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"error":   "unauthorized",
				"message": "Missing or invalid token",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"statusCode": 404,
			"error":      "not_found",
			"message":    "Log event not found",
		})
	}
}

// LogsErrorHandler returns a mock handler that returns specified error status
func LogsErrorHandler(statusCode int, errorCode, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]any{
			"statusCode": statusCode,
			"error":      errorCode,
			"message":    message,
		})
	}
}

// MockLogEvents returns sample log events for testing
func MockLogEvents() []LogEvent {
	return []LogEvent{
		{
			ID:       "900000000000000000000000000000000000000000000",
			LogID:    "900000000000000000000000000000000000000000000",
			Date:     "2026-04-13T14:23:01.000Z",
			Type:     "s",
			IP:       "192.168.1.100",
			UserID:   "auth0|abc123",
			UserName: "user1@example.com",
			ClientID: "client123def456",
			Details: map[string]any{
				"description": "Success Login",
			},
			Data: map[string]any{
				"user_name": "user1@example.com",
			},
		},
		{
			ID:       "900000000000000000000000000000000000000000001",
			LogID:    "900000000000000000000000000000000000000000001",
			Date:     "2026-04-13T14:22:30.000Z",
			Type:     "felo",
			IP:       "192.168.1.101",
			UserID:   "auth0|def456",
			UserName: "user2@example.com",
			ClientID: "client789ghi012",
			Details: map[string]any{
				"description": "Failed Login (incorrect password or MFA)",
			},
			Data: map[string]any{
				"user_name": "user2@example.com",
			},
		},
		{
			ID:       "900000000000000000000000000000000000000000002",
			LogID:    "900000000000000000000000000000000000000000002",
			Date:     "2026-04-13T14:20:15.000Z",
			Type:     "sapi",
			IP:       "10.0.0.50",
			UserID:   "",
			UserName: "",
			ClientID: "management-cli",
			Details: map[string]any{
				"description": "Success API Call",
			},
			Data: map[string]any{
				"method": "GET",
				"path":   "/api/v2/users",
			},
		},
	}
}

// MockLogEvent returns a single mock log event
func MockLogEvent() LogEvent {
	return LogEvent{
		ID:       "900000000000000000000000000000000000000000000",
		LogID:    "900000000000000000000000000000000000000000000",
		Date:     "2026-04-13T14:23:01.000Z",
		Type:     "s",
		IP:       "192.168.1.100",
		UserID:   "auth0|abc123",
		UserName: "user1@example.com",
		ClientID: "client123def456",
		Details: map[string]any{
			"description": "Success Login",
		},
		Data: map[string]any{
			"user_name": "user1@example.com",
		},
	}
}
