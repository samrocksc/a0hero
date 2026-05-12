// Package state provides event-driven state management for TUI sections.
package state

// SectionState represents the current state of a section.
type SectionState string

const (
	// StateIdle - viewing, no pending work
	StateIdle SectionState = "idle"

	// StateLoading - fetching list or detail data
	StateLoading SectionState = "loading"

	// StateEditing - in edit overlay, no changes yet
	StateEditing SectionState = "editing"

	// StatePendingEdit - in edit overlay, changes staged but not sent
	StatePendingEdit SectionState = "pending_edit"

	// StateAPICall - REST call in flight
	StateAPICall SectionState = "api_call"

	// StateError - error occurred, needs handling
	StateError SectionState = "error"
)

// StateInfo holds metadata about a state.
type StateInfo struct {
	State       SectionState
	Description string
	CanEdit     bool
	CanNavigate bool
	CanSave     bool
	IsTerminal  bool
}

// StateMap provides information about each state.
var StateMap = map[SectionState]StateInfo{
	StateIdle: {
		State:       StateIdle,
		Description: "Viewing, no pending work",
		CanEdit:     true,
		CanNavigate: true,
		CanSave:     false,
		IsTerminal:  true,
	},
	StateLoading: {
		State:       StateLoading,
		Description: "Fetching data from API",
		CanEdit:     false,
		CanNavigate: true,
		CanSave:     false,
		IsTerminal:  false,
	},
	StateEditing: {
		State:       StateEditing,
		Description: "In edit mode, no changes yet",
		CanEdit:     true,
		CanNavigate: true,
		CanSave:     false,
		IsTerminal:  false,
	},
	StatePendingEdit: {
		State:       StatePendingEdit,
		Description: "Changes staged, ready to save",
		CanEdit:     true,
		CanNavigate: true,
		CanSave:     true,
		IsTerminal:  false,
	},
	StateAPICall: {
		State:       StateAPICall,
		Description: "API call in progress",
		CanEdit:     false,
		CanNavigate: true,
		CanSave:     false,
		IsTerminal:  false,
	},
	StateError: {
		State:       StateError,
		Description: "Error occurred, needs attention",
		CanEdit:     false,
		CanNavigate: true,
		CanSave:     false,
		IsTerminal:  false,
	},
}

// CanTransition returns true if transitioning from -> to state is valid.
func CanTransition(from, to SectionState) bool {
	// Define valid transitions
	validTransitions := map[SectionState][]SectionState{
		StateIdle:         {StateLoading, StateEditing},
		StateLoading:      {StateIdle, StateError},
		StateEditing:      {StateIdle, StatePendingEdit, StateAPICall},
		StatePendingEdit:  {StateEditing, StateAPICall, StateIdle},
		StateAPICall:      {StateIdle, StateError},
		StateError:        {StateIdle, StateLoading, StateAPICall, StateEditing},
	}

	if valid, ok := validTransitions[from]; ok {
		for _, s := range valid {
			if s == to {
				return true
			}
		}
	}
	return false
}

// String returns the state name.
func (s SectionState) String() string {
	return string(s)
}

// IsIdle returns true if state is idle.
func (s SectionState) IsIdle() bool {
	return s == StateIdle
}

// IsLoading returns true if state is loading.
func (s SectionState) IsLoading() bool {
	return s == StateLoading
}

// IsEditing returns true if in edit or pending edit state.
func (s SectionState) IsEditing() bool {
	return s == StateEditing || s == StatePendingEdit
}

// HasPendingChanges returns true if there are unsaved changes.
func (s SectionState) HasPendingChanges() bool {
	return s == StatePendingEdit
}

// IsAPICallInProgress returns true if API call is active.
func (s SectionState) IsAPICallInProgress() bool {
	return s == StateAPICall
}

// IsError returns true if in error state.
func (s SectionState) IsError() bool {
	return s == StateError
}
