// Package models defines shared types and interfaces for the a0hero domain.
package models

// Pagination holds page parameters for list requests.
type Pagination struct {
	Page int
	Size int
}

// ListResponse is a generic paginated list response.
type ListResponse[T any] struct {
	Items []T
	Total int
	Next  string // cursor for next page
}

// TenantInfo describes a loaded tenant configuration.
type TenantInfo struct {
	Name   string
	Domain string
}
