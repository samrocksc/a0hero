// Package state provides event-driven state management for TUI sections.
package state

import (
	"reflect"

	"github.com/samrocksc/a0hero/modules/edit"
)

// EditSession tracks the state of an in-progress edit.
type EditSession struct {
	EntityType   string
	EntityID     string
	Fields       []edit.FieldDef
	Original     map[string]interface{} // original values from API
	Current      map[string]interface{} // current edited values
	Paused       bool                   // true if navigated away
	Error        error                  // last error if any
	FieldErrors  map[string]string      // per-field validation errors
}

// NewEditSession creates a new edit session.
func NewEditSession(entityType, entityID string, fields []edit.FieldDef, initial map[string]interface{}) *EditSession {
	// Deep copy initial to avoid modifying the original
	original := make(map[string]interface{})
	for k, v := range initial {
		original[k] = v
	}

	return &EditSession{
		EntityType:  entityType,
		EntityID:    entityID,
		Fields:      fields,
		Original:    original,
		Current:     copyMap(initial),
		FieldErrors: make(map[string]string),
	}
}

// GetFieldValue returns the current value of a field.
func (s *EditSession) GetFieldValue(key string) interface{} {
	if s == nil {
		return nil
	}
	return s.Current[key]
}

// SetFieldValue updates a field value and returns true if changed.
func (s *EditSession) SetFieldValue(key string, value interface{}) bool {
	if s == nil {
		return false
	}

	oldValue := s.Current[key]
	if !reflect.DeepEqual(oldValue, value) {
		s.Current[key] = value
		delete(s.FieldErrors, key) // clear error on change
		return true
	}
	return false
}

// HasChanges returns true if any field differs from original.
func (s *EditSession) HasChanges() bool {
	if s == nil {
		return false
	}
	for k, v := range s.Current {
		if !reflect.DeepEqual(v, s.Original[k]) {
			return true
		}
	}
	return false
}

// GetChanges returns only the changed fields as a map.
func (s *EditSession) GetChanges() map[string]interface{} {
	changes := make(map[string]interface{})
	for k, v := range s.Current {
		if !reflect.DeepEqual(v, s.Original[k]) {
			changes[k] = v
		}
	}
	return changes
}

// Reset reverts all changes to original values.
func (s *EditSession) Reset() {
	if s == nil {
		return
	}
	s.Current = copyMap(s.Original)
	s.FieldErrors = make(map[string]string)
	s.Error = nil
}

// UpdateFromResult updates session with save result from API.
func (s *EditSession) UpdateFromResult(result map[string]interface{}) {
	if s == nil || result == nil {
		return
	}
	s.Original = copyMap(result)
	s.Current = copyMap(result)
	s.FieldErrors = make(map[string]string)
	s.Error = nil
}

// GetFieldError returns the error for a specific field.
func (s *EditSession) GetFieldError(key string) string {
	if s == nil {
		return ""
	}
	return s.FieldErrors[key]
}

// SetFieldError sets an error for a specific field.
func (s *EditSession) SetFieldError(key, err string) {
	if s == nil {
		return
	}
	s.FieldErrors[key] = err
}

// SetError sets the session-wide error.
func (s *EditSession) SetError(err error) {
	if s == nil {
		return
	}
	s.Error = err
}

// Validate runs validation on current values.
func (s *EditSession) Validate() map[string]string {
	if s == nil {
		return nil
	}

	helper := edit.NewFieldHelper(s.Fields)
	errors := helper.ValidateAll(s.Current)

	s.FieldErrors = make(map[string]string)
	for _, e := range errors {
		s.FieldErrors[e.Field] = e.Message
	}

	return s.FieldErrors
}

// IsValid returns true if no validation errors exist.
func (s *EditSession) IsValid() bool {
	if s == nil {
		return true
	}
	return len(s.FieldErrors) == 0
}

// copyMap creates a shallow copy of a map.
func copyMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return make(map[string]interface{})
	}
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// PendingAction represents a queued async operation.
type PendingAction struct {
	ID        string                   // unique call ID
	Type      string                   // "update", "create", "delete"
	EntityID  string                   // target entity
	Changes   map[string]interface{}   // data to send
	StartedAt int64                    // unix timestamp
}

// IsEmpty returns true if no action is pending.
func (pa PendingAction) IsEmpty() bool {
	return pa.ID == ""
}
