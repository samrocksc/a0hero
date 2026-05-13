// Package state provides event-driven state management for TUI sections.
package state

import (
	"fmt"
)

// Section identifies which part of the app we're in.
type Section int

const (
	SecUsers Section = iota
	SecClients
	SecRoles
	SecConnections
	SecLogs
	SecConfigure
	SecCount // sentinel
)

// SectionEvent is the interface for all section-scoped events.
type SectionEvent interface {
	EventType() string
	GetSection() Section
}

// EventStartEdit is sent when user enters edit mode for an entity.
type EventStartEdit struct {
	Section  Section
	EntityID string
	EntityType string
	InitialState map[string]interface{}
}

func (e EventStartEdit) EventType() string { return "start_edit" }
func (e EventStartEdit) GetSection() Section { return e.Section }

// EventFieldChange is sent when a field value changes in edit mode.
type EventFieldChange struct {
	Section Section
	EntityID string
	Field   string
	Value   interface{}
}

func (e EventFieldChange) EventType() string { return "field_change" }
func (e EventFieldChange) GetSection() Section { return e.Section }

// EventSubmit is sent when user submits (Ctrl+S) pending changes.
type EventSubmit struct {
	Section Section
	EntityID string
}

func (e EventSubmit) EventType() string { return "submit" }
func (e EventSubmit) GetSection() Section { return e.Section }

// EventCancelEdit is sent when user cancels edit.
type EventCancelEdit struct {
	Section Section
	EntityID string
}

func (e EventCancelEdit) EventType() string { return "cancel_edit" }
func (e EventCancelEdit) GetSection() Section { return e.Section }

// EventAPICallStart is sent when an async API call begins.
type EventAPICallStart struct {
	Section Section
	CallID  string
	Operation string // e.g., "update", "create", "delete"
}

func (e EventAPICallStart) EventType() string { return "api_call_start" }
func (e EventAPICallStart) GetSection() Section { return e.Section }

// EventAPICallComplete is sent when an async API call finishes.
type EventAPICallComplete struct {
	Section  Section
	CallID   string
	EntityID string
	Result   map[string]interface{}
	Err      error
}

func (e EventAPICallComplete) EventType() string { return "api_call_complete" }
func (e EventAPICallComplete) GetSection() Section { return e.Section }

// EventNavigateAway is sent when user switches to another section.
type EventNavigateAway struct {
	Section Section
}

func (e EventNavigateAway) EventType() string { return "navigate_away" }
func (e EventNavigateAway) GetSection() Section { return e.Section }

// EventNavigateTo is sent when user switches to this section.
type EventNavigateTo struct {
	Section Section
}

func (e EventNavigateTo) EventType() string { return "navigate_to" }
func (e EventNavigateTo) GetSection() Section { return e.Section }

// EventDataLoadStart is sent when fetching list/detail data begins.
type EventDataLoadStart struct {
	Section Section
	View    string // "list" or "detail"
}

func (e EventDataLoadStart) EventType() string { return "data_load_start" }
func (e EventDataLoadStart) GetSection() Section { return e.Section }

// EventDataLoadComplete is sent when data fetching finishes.
type EventDataLoadComplete struct {
	Section Section
	View    string
	Data    interface{}
	Err     error
}

func (e EventDataLoadComplete) EventType() string { return "data_load_complete" }
func (e EventDataLoadComplete) GetSection() Section { return e.Section }

// EventError is a generic error event that can occur in any state.
type EventError struct {
	Section Section
	Err     error
}

func (e EventError) EventType() string { return "error" }
func (e EventError) GetSection() Section { return e.Section }

// EventRetry is sent when user requests retry after error.
type EventRetry struct {
	Section Section
}

func (e EventRetry) EventType() string { return "retry" }
func (e EventRetry) GetSection() Section { return e.Section }

// EventDismissError is sent when user dismisses an error.
type EventDismissError struct {
	Section Section
}

func (e EventDismissError) EventType() string { return "dismiss_error" }
func (e EventDismissError) GetSection() Section { return e.Section }

// EventEditingStarted is sent when edit overlay is ready and user can start editing.
type EventEditingStarted struct {
	Section  Section
	EntityID string
}

func (e EventEditingStarted) EventType() string { return "editing_started" }
func (e EventEditingStarted) GetSection() Section { return e.Section }

// String returns a human-readable representation of a section.
func (s Section) String() string {
	names := []string{"Users", "Clients", "Roles", "Connections", "Logs", "Configure"}
	if int(s) >= 0 && int(s) < len(names) {
		return names[s]
	}
	return fmt.Sprintf("Unknown(%d)", s)
}
