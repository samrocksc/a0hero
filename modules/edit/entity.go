package edit

import (
	"context"
)

// EntityService defines the interface for editable entities.
type EntityService interface {
	// EntityType returns the type name (e.g., "client", "user")
	EntityType() string

	// GetFields returns the field definitions for this entity type
	GetFields() []FieldDef

	// Fetch retrieves the current state of an entity by ID
	Fetch(ctx context.Context, id string) (map[string]interface{}, error)

	// Update applies changes and returns the updated entity
	Update(ctx context.Context, id string, changes map[string]interface{}) (map[string]interface{}, error)
}

// BaseService provides common functionality for entity services.
type BaseService struct {
	HistoryDir string
}

// NewBaseService creates a new base service with a history directory.
func NewBaseService(historyDir string) *BaseService {
	return &BaseService{
		HistoryDir: historyDir,
	}
}

// GetHistoryWriter returns a configured history writer.
func (s *BaseService) GetHistoryWriter() *HistoryWriter {
	return NewHistoryWriter(s.HistoryDir)
}

// FieldHelper provides utilities for working with fields.
type FieldHelper struct {
	Fields []FieldDef
}

// NewFieldHelper creates a new field helper.
func NewFieldHelper(fields []FieldDef) *FieldHelper {
	return &FieldHelper{Fields: fields}
}

// GetField returns a field definition by key.
func (h *FieldHelper) GetField(key string) *FieldDef {
	for i := range h.Fields {
		if h.Fields[i].Key == key {
			return &h.Fields[i]
		}
	}
	return nil
}

// IsReadOnly checks if a field is read-only.
func (h *FieldHelper) IsReadOnly(key string) bool {
	f := h.GetField(key)
	if f == nil {
		return true
	}
	return f.ReadOnly
}

// Validate validates a single field value.
func (h *FieldHelper) Validate(key string, value interface{}) error {
	field := h.GetField(key)
	if field == nil {
		return nil
	}

	var validators []Validator

	if field.Required {
		validators = append(validators, ValidateRequired)
	}

	switch field.Type {
	case FieldURL:
		validators = append(validators, ValidateURL)
	case FieldTagArray:
		// Determine if it's a URL array by looking at placeholder or key
		if containsURLHint(key) {
			validators = append(validators, ValidateURLArray)
		} else {
			validators = append(validators, ValidateNoDuplicates)
		}
	case FieldText:
		if field.Key == "email" {
			validators = append(validators, ValidateEmail)
		}
	}

	if len(validators) == 0 {
		return nil
	}

	combined := CombineValidators(validators...)
	return combined(value)
}

// ValidateAll validates all fields in a map.
func (h *FieldHelper) ValidateAll(values map[string]interface{}) ValidationErrors {
	var errors ValidationErrors

	for _, field := range h.Fields {
		if field.ReadOnly {
			continue
		}

		value := values[field.Key]
		if err := h.Validate(field.Key, value); err != nil {
			errors = append(errors, ValidationError{
				Field:   field.Label,
				Message: err.Error(),
			})
		}
	}

	return errors
}

// containsURLHint checks if a key suggests URL content.
func containsURLHint(key string) bool {
	urlKeys := []string{"url", "uri", "origin", "callback", "logout", "redirect"}
	for _, k := range urlKeys {
		if contains(key, k) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
