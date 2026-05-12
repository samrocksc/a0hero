// Package state provides event-driven state management for TUI sections.
package state

import (
	"fmt"
	"testing"
)

// ExampleSectionMachine demonstrates the state machine flow.
func ExampleSectionMachine() {
	// Create a machine for the Clients section
	machine := NewSectionMachine(SecClients)

	// Initial state is idle
	fmt.Println("Initial state:", machine.GetState())

	// Start editing
	machine.ProcessEvent(EventStartEdit{
		Section:      SecClients,
		EntityType:   "client",
		EntityID:     "client-123",
		InitialState: map[string]interface{}{"name": "Original"},
	})
	fmt.Println("After start edit:", machine.GetState())
	fmt.Println("Has session:", machine.HasEditSession())

	// Make a change
	machine.ProcessEvent(EventFieldChange{
		Section:  SecClients,
		EntityID: "client-123",
		Field:    "name",
		Value:    "Modified",
	})
	fmt.Println("After field change:", machine.GetState())
	fmt.Println("Has changes:", machine.GetSession().HasChanges())

	// Check save status
	fmt.Println("Save status:", machine.GetSaveStatus())

	// Output:
	// Initial state: idle
	// After start edit: editing
	// Has session: true
	// After field change: pending_edit
	// Has changes: true
	// Save status: Unsaved changes
}

// TestStateTransitions validates all valid state transitions.
func TestStateTransitions(t *testing.T) {
	tests := []struct {
		from  SectionState
		to    SectionState
		valid bool
		name  string
	}{
		{StateIdle, StateEditing, true, "idle to editing"},
		{StateIdle, StateLoading, true, "idle to loading"},
		{StateEditing, StatePendingEdit, true, "editing to pending_edit"},
		{StatePendingEdit, StateAPICall, true, "pending to api_call"},
		{StateAPICall, StateIdle, true, "api_call to idle"},
		{StateError, StateIdle, true, "error to idle"},
		{StateIdle, StateAPICall, false, "idle cannot go directly to api_call"},
		{StateLoading, StateEditing, false, "loading cannot go directly to editing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanTransition(tt.from, tt.to)
			if got != tt.valid {
				t.Errorf("CanTransition(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.valid)
			}
		})
	}
}

// TestEditSession tracks changes correctly.
func TestEditSession(t *testing.T) {
	session := NewEditSession("client", "client-123", nil,
		map[string]interface{}{"name": "Test", "enabled": true})

	// No changes initially
	if session.HasChanges() {
		t.Error("Expected no changes initially")
	}

	// Change a value
	session.SetFieldValue("name", "Modified")
	if !session.HasChanges() {
		t.Error("Expected HasChanges() to be true after modification")
	}

	// Get changes
	changes := session.GetChanges()
	if len(changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changes))
	}
	if changes["name"] != "Modified" {
		t.Errorf("Expected name='Modified', got %v", changes["name"])
	}

	// Revert to original
	session.Reset()
	if session.HasChanges() {
		t.Error("Expected no changes after reset")
	}

	// Original value restored
	if session.GetFieldValue("name") != "Test" {
		t.Errorf("Expected name='Test' after reset, got %v", session.GetFieldValue("name"))
	}
}

// TestSectionManager tracks multiple machines.
func TestSectionManager(t *testing.T) {
	manager := NewStateManager(nil)

	// Default section
	if manager.GetCurrentSection() != SecUsers {
		t.Errorf("Expected default section Users, got %s", manager.GetCurrentSection())
	}

	// All sections have machines
	for sec := Section(0); sec < SecCount; sec++ {
		if manager.GetMachine(sec) == nil {
			t.Errorf("Expected machine for section %s", sec)
		}
	}

	// Switch sections - will return empty commands since machines are idle
	cmds := manager.SetCurrentSection(SecClients)
	if manager.GetCurrentSection() != SecClients {
		t.Errorf("Expected current section Clients after switch")
	}
	_ = cmds
}

// TestStateInfo provides correct information.
func TestStateInfo(t *testing.T) {
	info, ok := StateMap[StateIdle]
	if !ok {
		t.Fatal("Expected StateIdle in StateMap")
	}
	if !info.CanNavigate {
		t.Error("Expected CanNavigate=true for Idle state")
	}
	if !info.IsTerminal {
		t.Error("Expected IsTerminal=true for Idle state")
	}

	// Loading state
	info, ok = StateMap[StateLoading]
	if !ok {
		t.Fatal("Expected StateLoading in StateMap")
	}
	if info.CanEdit {
		t.Error("Expected CanEdit=false for Loading state")
	}
}
