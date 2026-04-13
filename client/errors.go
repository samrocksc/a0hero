// Package client provides an authenticated HTTP client for the Auth0 Management API.
package client

import "fmt"

// APIError wraps an Auth0 API error response.
type APIError struct {
	StatusCode int
	Message    string
	Code       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("auth0: %s (status %d, code %s)", e.Message, e.StatusCode, e.Code)
}

// Unwrap allows errors.Is / errors.As to work with wrapped errors.
func (e *APIError) Unwrap() error {
	return nil
}
