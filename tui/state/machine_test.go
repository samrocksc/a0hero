// Package state provides event-driven state management for TUI sections.
package state

import (
	"context"
	"fmt"
	"testing"
)

// mockEntityService is a test double for EntityService.
type mockEntityService struct {
	fetchCalled   bool
	updateCalled  bool
	lastFetchID   string
	lastUpdateID  string
	lastChanges   map[string]interface{}
	fetchResult   map[string]interface{}
	updateResult  map[string]interface{}
	fetchError    error
	updateError   error
}

func (m *mockEntityService) EntityType() string { return "test_entity" }

func (m *mockEntityService) GetFields() []interface{} {
	return []interface{}{}
}

func (m *mockEntityService) Fetch(ctx context.Context, id string) (map[string]interface{}, error) {
	m.fetchCalled = true
	m.lastFetchID = id
	if m.fetchError != nil {
		return nil, m.fetchError
	}
	return m.fetchResult, nil
}

func (m *mockEntityService) Update(ctx context.Context, id string, changes map[string]interface{}) (map[string]interface{}, error) {
	m.updateCalled = true
	m.lastUpdateID = id
	m.lastChanges = changes
	if m.updateError != nil {
		return nil, m.updateError
	}
	return m.updateResult, nil
}

// TestSaveFlowEndToEnd validates the complete save flow from submit to completion.
func TestSaveFlowEndToEnd(t *testing.T) {
	// Setup mock service
	mockSvc := &mockEntityService{
		updateResult: map[string]interface{}{
			"id":   "client-123",
			"name": "Updated App",
			"type": "spa",
		},
	}

	// Create machine with mock service
	machine := NewSectionMachine(SecClients, WithService(mockSvc))

	// Initial state should be idle
	if machine.GetState() != StateIdle {
		t.Fatalf("Expected initial state idle, got %s", machine.GetState())
	}

	// Step 1: Start editing
	_, _ = machine.ProcessEvent(EventStartEdit{
		Section:      SecClients,
		EntityType:   "client",
		EntityID:     "client-123",
		InitialState: map[string]interface{}{"name": "Original App", "type": "spa"},
	})

	if machine.GetState() != StateEditing {
		t.Fatalf("Expected state editing, got %s", machine.GetState())
	}

	// Step 2: Make a field change
	_, _ = machine.ProcessEvent(EventFieldChange{
		Section:  SecClients,
		EntityID: "client-123",
		Field:    "name",
		Value:    "Updated App",
	})

	if machine.GetState() != StatePendingEdit {
		t.Fatalf("Expected state pending_edit, got %s", machine.GetState())
	}

	if !machine.GetSession().HasChanges() {
		t.Error("Expected session HasChanges() to be true")
	}

	// Step 3: Submit the save - this should return a command
	newState, cmd := machine.ProcessEvent(EventSubmit{
		Section:  SecClients,
		EntityID: "client-123",
	})

	if newState != StateAPICall {
		t.Fatalf("Expected state api_call, got %s", newState)
	}

	if cmd == nil {
		t.Fatal("Expected command from EventSubmit, got nil")
	}

	if !machine.HasPendingAction() {
		t.Error("Expected HasPendingAction() to be true")
	}

	// Step 4: Execute the command (simulates Bubble Tea framework)
	msg := cmd()

	// Command should return EventAPICallComplete
	event, ok := msg.(EventAPICallComplete)
	if !ok {
		t.Fatalf("Expected EventAPICallComplete, got %T", msg)
	}

	// Verify the update was called
	if !mockSvc.updateCalled {
		t.Error("Expected Update() to be called on service")
	}

	if mockSvc.lastUpdateID != "client-123" {
		t.Errorf("Expected update ID 'client-123', got %s", mockSvc.lastUpdateID)
	}

	if event.EntityID != "client-123" {
		t.Errorf("Expected event EntityID 'client-123', got %s", event.EntityID)
	}

	if event.Err != nil {
		t.Errorf("Expected no error, got %v", event.Err)
	}

	// Step 5: Process the completion event
	finalState, _ := machine.ProcessEvent(event)

	if finalState != StateIdle {
		t.Fatalf("Expected final state idle, got %s", finalState)
	}

	if machine.HasPendingAction() {
		t.Error("Expected pending action to be cleared")
	}
}

// TestSaveFlowWithError validates error handling during save.
func TestSaveFlowWithError(t *testing.T) {
	// Setup mock service that returns an error
	mockSvc := &mockEntityService{
		updateError: fmt.Errorf("API error: rate limit exceeded"),
	}

	machine := NewSectionMachine(SecClients, WithService(mockSvc))

	// Start editing and make a change
	_, _ = machine.ProcessEvent(EventStartEdit{
		Section:      SecClients,
		EntityType:   "client",
		EntityID:     "client-456",
		InitialState: map[string]interface{}{"name": "Test App"},
	})

	_, _ = machine.ProcessEvent(EventFieldChange{
		Section:  SecClients,
		Field:    "name",
		Value:    "Modified App",
	})

	// Submit save
	_, cmd := machine.ProcessEvent(EventSubmit{
		Section:  SecClients,
		EntityID: "client-456",
	})

	if cmd == nil {
		t.Fatal("Expected command from EventSubmit")
	}

	// Execute command
	msg := cmd()
	event := msg.(EventAPICallComplete)

	// Verify error was returned
	if event.Err == nil {
		t.Error("Expected error in EventAPICallComplete")
	}

	// Process error - should transition to error state
	newState, _ := machine.ProcessEvent(event)

	if newState != StateError {
		t.Fatalf("Expected state error, got %s", newState)
	}

	if machine.GetError() == nil {
		t.Error("Expected machine to have error set")
	}
}

// TestAPICallWithoutChanges validates that submit without changes doesn't trigger API call.
func TestAPICallWithoutChanges(t *testing.T) {
	mockSvc := &mockEntityService{}
	machine := NewSectionMachine(SecClients, WithService(mockSvc))

	// Start editing but don't make changes
	_, _ = machine.ProcessEvent(EventStartEdit{
		Section:      SecClients,
		EntityType:   "client",
		EntityID:     "client-789",
		InitialState: map[string]interface{}{"name": "Same App"},
	})

	// Submit without changes
	_, cmd := machine.ProcessEvent(EventSubmit{
		Section:  SecClients,
		EntityID: "client-789",
	})

	// Should stay in editing state
	if machine.GetState() != StateEditing {
		t.Errorf("Expected state editing, got %s", machine.GetState())
	}

	// No command should be generated
	if cmd != nil {
		t.Error("Expected no command when submitting without changes")
	}

	if mockSvc.updateCalled {
		t.Error("Update should not be called without changes")
	}
}

// TestCancelEdit validates that canceling clears session and pending changes.
func TestCancelEdit(t *testing.T) {
	machine := NewSectionMachine(SecClients)

	// Start editing and make changes
	_, _ = machine.ProcessEvent(EventStartEdit{
		Section:      SecClients,
		EntityType:   "client",
		EntityID:     "client-abc",
		InitialState: map[string]interface{}{"name": "Original"},
	})

	_, _ = machine.ProcessEvent(EventFieldChange{
		Section: SecClients,
		Field:   "name",
		Value:   "Changed",
	})

	if !machine.HasEditSession() {
		t.Error("Expected edit session to exist")
	}

	// Cancel
	newState, _ := machine.ProcessEvent(EventCancelEdit{
		Section:  SecClients,
		EntityID: "client-abc",
	})

	if newState != StateIdle {
		t.Fatalf("Expected state idle after cancel, got %s", newState)
	}

	if machine.HasEditSession() {
		t.Error("Expected edit session to be cleared")
	}

	if machine.GetSession() != nil && machine.GetSession().HasChanges() {
		t.Error("Expected pending changes to be cleared")
	}
}

// TestRetryAfterError validates that retry works from error state.
func TestRetryAfterError(t *testing.T) {
	mockSvc := &mockEntityService{
		updateError: fmt.Errorf("network error"),
	}

	machine := NewSectionMachine(SecClients, WithService(mockSvc))

	// Get to error state
	_, _ = machine.ProcessEvent(EventStartEdit{
		Section:      SecClients,
		EntityType:   "client",
		EntityID:     "client-retry",
		InitialState: map[string]interface{}{"name": "Test"},
	})

	_, _ = machine.ProcessEvent(EventFieldChange{
		Section: SecClients,
		Field:   "name",
		Value:   "Modified",
	})

	_, cmd := machine.ProcessEvent(EventSubmit{
		Section:  SecClients,
		EntityID: "client-retry",
	})

	// Force to error by executing command
	msg := cmd()
	_, _ = machine.ProcessEvent(msg.(EventAPICallComplete))

	if machine.GetState() != StateError {
		t.Fatalf("Expected state error, got %s", machine.GetState())
	}

	// Now fix the service and retry
	mockSvc.updateError = nil
	mockSvc.updateResult = map[string]interface{}{"name": "Modified"}

	// Retry should trigger the API call again
	_, retryCmd := machine.ProcessEvent(EventRetry{
		Section: SecClients,
	})

	if retryCmd == nil {
		t.Fatal("Expected command from retry")
	}

	// Execute retry
	msg = retryCmd()
	event := msg.(EventAPICallComplete)

	if event.Err != nil {
		t.Errorf("Expected success on retry, got error: %v", event.Err)
	}

	if !mockSvc.updateCalled {
		t.Error("Expected Update() to be called on retry")
	}
}
