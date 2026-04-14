package edit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Change represents a single field change.
type Change struct {
	Timestamp time.Time              `json:"timestamp"`
	Field     string                 `json:"field"`
	OldValue  interface{}            `json:"old_value"`
	NewValue  interface{}            `json:"new_value"`
}

// HistoryEntry represents a complete edit session.
type HistoryEntry struct {
	Version    int                    `json:"version"`
	SessionID  string                 `json:"session_id"`
	EntityType string                 `json:"entity_type"`
	EntityID   string                 `json:"entity_id"`
	StartedAt  time.Time              `json:"started_at"`
	EndedAt    time.Time              `json:"ended_at"`
	UserAgent  string                 `json:"user_agent"`
	Changes    []Change               `json:"changes"`
	FinalState map[string]interface{} `json:"final_state,omitempty"`
	Status     string                 `json:"status"` // "saved", "cancelled", "failed"
	APIResponse *APIResponse          `json:"api_response,omitempty"`
}

// APIResponse captures the API call result.
type APIResponse struct {
	StatusCode int    `json:"status_code"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

// Status constants
const (
	StatusSaved     = "saved"
	StatusCancelled = "cancelled"
	StatusFailed    = "failed"
)

// Redacted fields (case-insensitive match)
var redactedFields = map[string]bool{
	"client_secret":   true,
	"encryption_key":  true,
}

// IsRedacted checks if a field name should be redacted.
func IsRedacted(fieldName string) bool {
	if redactedFields[fieldName] {
		return true
	}
	lower := fmt.Sprintf("%s", fieldName)
	return len(lower) > 8 && lower[len(lower)-8:] == "password"
}

// RedactValue returns a redacted value for sensitive fields.
func RedactValue(fieldName string, value interface{}) interface{} {
	if IsRedacted(fieldName) {
		return "[REDACTED]"
	}
	return value
}

// HistoryWriter handles writing edit history to disk.
type HistoryWriter struct {
	baseDir string
}

// NewHistoryWriter creates a new history writer.
func NewHistoryWriter(baseDir string) *HistoryWriter {
	return &HistoryWriter{baseDir: baseDir}
}

// Write writes a history entry to disk.
func (w *HistoryWriter) Write(entry *HistoryEntry) error {
	// Create directory structure: <baseDir>/<entityType>/<entityID>/
	dir := filepath.Join(w.baseDir, entry.EntityType, entry.EntityID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create history dir: %w", err)
	}

	// Generate filename: <timestamp>.json
	filename := fmt.Sprintf("%s.json", entry.StartedAt.Format("2006-01-02T150405"))
	path := filepath.Join(dir, filename)

	// Marshal to JSON
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal history: %w", err)
	}

	// Write to disk
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write history file: %w", err)
	}

	return nil
}

// Read reads a history entry from disk.
func (w *HistoryWriter) Read(entityType, entityID, filename string) (*HistoryEntry, error) {
	path := filepath.Join(w.baseDir, entityType, entityID, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read history file: %w", err)
	}

	var entry HistoryEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("parse history file: %w", err)
	}

	return &entry, nil
}

// List returns all history entries for an entity.
func (w *HistoryWriter) List(entityType, entityID string) ([]HistoryEntry, error) {
	dir := filepath.Join(w.baseDir, entityType, entityID)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read history dir: %w", err)
	}

	var results []HistoryEntry
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		e, err := w.Read(entityType, entityID, entry.Name())
		if err != nil {
			continue // Skip invalid entries
		}
		results = append(results, *e)
	}

	return results, nil
}

// ListAll returns all history entries grouped by entity, sorted by date (newest first).
func (w *HistoryWriter) ListAll(entityType string) (map[string][]HistoryEntry, error) {
	typeDir := filepath.Join(w.baseDir, entityType)

	entries, err := os.ReadDir(typeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read type dir: %w", err)
	}

	results := make(map[string][]HistoryEntry)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		entityID := entry.Name()
		history, err := w.List(entityType, entityID)
		if err != nil {
			continue
		}
		if len(history) > 0 {
			results[entityID] = history
		}
	}

	return results, nil
}

// EditSession tracks an active editing session.
type EditSession struct {
	SessionID   string
	EntityType  string
	EntityID    string
	StartedAt   time.Time
	Fields      []FieldDef
	Original    map[string]interface{}
	Current     map[string]interface{}
	Changes     []Change
	UserAgent   string
}

// NewSession creates a new edit session.
func NewSession(entityType, entityID string, fields []FieldDef, currentState map[string]interface{}) *EditSession {
	return &EditSession{
		SessionID:  fmt.Sprintf("%s-editor", time.Now().Format(time.RFC3339)),
		EntityType: entityType,
		EntityID:   entityID,
		StartedAt:  time.Now(),
		Fields:     fields,
		Original:   deepCopy(currentState),
		Current:   deepCopy(currentState),
		Changes:   []Change{},
		UserAgent: "a0hero",
	}
}

// SetValue sets a field value and records the change.
func (s *EditSession) SetValue(fieldKey string, newValue interface{}) {
	oldValue := s.Current[fieldKey]

	// Skip if no actual change
	if fmt.Sprintf("%v", oldValue) == fmt.Sprintf("%v", newValue) {
		return
	}

	s.Current[fieldKey] = newValue
	s.Changes = append(s.Changes, Change{
		Timestamp: time.Now(),
		Field:     fieldKey,
		OldValue:  RedactValue(fieldKey, oldValue),
		NewValue:  RedactValue(fieldKey, newValue),
	})
}

// Undo undoes the last change.
func (s *EditSession) Undo() bool {
	if len(s.Changes) == 0 {
		return false
	}

	last := s.Changes[len(s.Changes)-1]
	s.Changes = s.Changes[:len(s.Changes)-1]
	s.Current[last.Field] = last.OldValue
	return true
}

// IsDirty returns true if there are unsaved changes.
func (s *EditSession) IsDirty() bool {
	return len(s.Changes) > 0
}

// GetChangeCount returns the number of changes.
func (s *EditSession) GetChangeCount() int {
	return len(s.Changes)
}

// ToHistoryEntry creates a history entry from the session.
func (s *EditSession) ToHistoryEntry(status string, apiResp *APIResponse) *HistoryEntry {
	// Redact sensitive fields in final state
	finalState := make(map[string]interface{})
	for k, v := range s.Current {
		finalState[k] = RedactValue(k, v)
	}

	return &HistoryEntry{
		Version:    1,
		SessionID:  s.SessionID,
		EntityType: s.EntityType,
		EntityID:   s.EntityID,
		StartedAt:  s.StartedAt,
		EndedAt:    time.Now(),
		UserAgent:  s.UserAgent,
		Changes:    s.Changes,
		FinalState: finalState,
		Status:     status,
		APIResponse: apiResp,
	}
}

// HasFieldChanged checks if a specific field has been modified.
func (s *EditSession) HasFieldChanged(fieldKey string) bool {
	for _, c := range s.Changes {
		if c.Field == fieldKey {
			return true
		}
	}
	return false
}

// GetChangedFields returns a map of only changed fields.
func (s *EditSession) GetChangedFields() map[string]interface{} {
	result := make(map[string]interface{})
	for _, c := range s.Changes {
		result[c.Field] = c.NewValue
	}
	return result
}

// deepCopy creates a deep copy of a map.
func deepCopy(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			result[k] = deepCopy(val)
		case []interface{}:
			result[k] = deepCopySlice(val)
		default:
			result[k] = v
		}
	}
	return result
}

func deepCopySlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))
	copy(result, s)
	return result
}
