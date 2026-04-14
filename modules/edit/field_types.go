// Package edit provides extensible editing capabilities for Auth0 entities.
package edit

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// FieldType represents the type of an editable field.
type FieldType int

const (
	FieldText FieldType = iota
	FieldTextarea
	FieldURL
	FieldTagArray
	FieldBool
	FieldSelect
	FieldPassword
	FieldDatetime
)

// String returns a human-readable name for the field type.
func (f FieldType) String() string {
	switch f {
	case FieldText:
		return "text"
	case FieldTextarea:
		return "textarea"
	case FieldURL:
		return "url"
	case FieldTagArray:
		return "tag_array"
	case FieldBool:
		return "bool"
	case FieldSelect:
		return "select"
	case FieldPassword:
		return "password"
	case FieldDatetime:
		return "datetime"
	default:
		return "unknown"
	}
}

// FieldDef defines a single editable field.
type FieldDef struct {
	// Key is the JSON/API field name (e.g., "callbacks", "app_type")
	Key string

	// Label is the human-readable label shown in the UI
	Label string

	// Type determines the UI component and validation
	Type FieldType

	// Required indicates if the field must have a value
	Required bool

	// ReadOnly fields cannot be edited (shown as display only)
	ReadOnly bool

	// Options for select fields
	Options []string

	// Placeholder text
	Placeholder string

	// Help text shown below the field
	Help string
}

// ValidationError represents a single validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (v ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", v.Field, v.Message)
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return ""
	}
	var msgs []string
	for _, e := range v {
		msgs = append(msgs, fmt.Sprintf("• %s", e.Message))
	}
	return strings.Join(msgs, "\n")
}

func (v ValidationErrors) HasErrors() bool {
	return len(v) > 0
}

// Validator is a function that validates a field value.
type Validator func(value interface{}) error

// ValidateURL validates a single URL string.
func ValidateURL(value interface{}) error {
	if value == nil || value == "" {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected string value")
	}

	parsed, err := url.ParseRequestURI(str)
	if err != nil {
		return fmt.Errorf("%q is not a valid URL", str)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme")
	}

	if parsed.Host == "" {
		return fmt.Errorf("URL must include a host")
	}

	return nil
}

// ValidateURLArray validates an array of URLs.
func ValidateURLArray(value interface{}) error {
	if value == nil {
		return nil
	}

	arr, ok := value.([]string)
	if !ok {
		return fmt.Errorf("expected array of strings")
	}

	seen := make(map[string]bool)
	for _, u := range arr {
		if err := ValidateURL(u); err != nil {
			return err
		}
		if seen[u] {
			return fmt.Errorf("%q is a duplicate", u)
		}
		seen[u] = true
	}

	return nil
}

// ValidateNoDuplicates checks for duplicate strings in an array.
func ValidateNoDuplicates(value interface{}) error {
	if value == nil {
		return nil
	}

	arr, ok := value.([]string)
	if !ok {
		return fmt.Errorf("expected array of strings")
	}

	seen := make(map[string]bool)
	for _, s := range arr {
		if seen[s] {
			return fmt.Errorf("%q is a duplicate", s)
		}
		seen[s] = true
	}

	return nil
}

// ValidateRequired ensures a required field has a value.
func ValidateRequired(value interface{}) error {
	if value == nil {
		return fmt.Errorf("this field is required")
	}

	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("this field is required")
		}
	case []string:
		if len(v) == 0 {
			return fmt.Errorf("at least one value is required")
		}
	}

	return nil
}

// ValidateEmail validates an email string.
func ValidateEmail(value interface{}) error {
	if value == nil || value == "" {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected string value")
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(str) {
		return fmt.Errorf("%q is not a valid email address", str)
	}

	return nil
}

// CombineValidators returns a validator that runs all provided validators.
func CombineValidators(validators ...Validator) Validator {
	return func(value interface{}) error {
		for _, v := range validators {
			if err := v(value); err != nil {
				return err
			}
		}
		return nil
	}
}
