// Package state provides event-driven state management for TUI sections.
package state

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/samrocksc/a0hero/logger"
	"github.com/samrocksc/a0hero/modules/edit"
)

// EntityService defines the interface for entity operations.
type EntityService interface {
	Fetch(ctx context.Context, id string) (map[string]interface{}, error)
	Update(ctx context.Context, id string, changes map[string]interface{}) (map[string]interface{}, error)
	EntityType() string
	GetFields() []interface{}
}

// SectionMachine manages state for a single section.
type SectionMachine struct {
	section       Section
	state         SectionState
	session       *EditSession
	pendingAction PendingAction
	data          interface{} // current view data (list or detail)
	error         error     // current error if any
	service       EntityService
	callInFlight  string // ID of active API call
}

// SectionMachineOption configures a SectionMachine.
type SectionMachineOption func(*SectionMachine)

// WithService sets the entity service for the machine.
func WithService(svc EntityService) SectionMachineOption {
	return func(m *SectionMachine) {
		m.service = svc
	}
}

// NewSectionMachine creates a new state machine for a section.
func NewSectionMachine(section Section, opts ...SectionMachineOption) *SectionMachine {
	m := &SectionMachine{
		section: section,
		state:   StateIdle,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// ProcessEvent handles an event and transitions state.
// Returns any command to execute and the resulting state.
func (m *SectionMachine) ProcessEvent(evt SectionEvent) (SectionState, tea.Cmd) {
	// Validate event belongs to this section
	if evt.GetSection() != m.section {
		return m.state, nil // ignore events for other sections
	}

	logger.Info("ProcessEvent ENTRY", "section", m.section, "state", m.state, "eventType", fmt.Sprintf("%T", evt))

	// Process based on current state
	switch m.state {
	case StateIdle:
		return m.handleIdleEvent(evt)
	case StateLoading:
		return m.handleLoadingEvent(evt)
	case StateEditing:
		return m.handleEditingEvent(evt)
	case StatePendingEdit:
		return m.handlePendingEditEvent(evt)
	case StateAPICall:
		return m.handleAPICallEvent(evt)
	case StateError:
		return m.handleErrorEvent(evt)
	default:
		return m.state, nil
	}
}

// GetState returns the current state.
func (m *SectionMachine) GetState() SectionState {
	return m.state
}

// GetSession returns the current edit session.
func (m *SectionMachine) GetSession() *EditSession {
	return m.session
}

// GetData returns the current view data.
func (m *SectionMachine) GetData() interface{} {
	return m.data
}

// GetError returns the current error.
func (m *SectionMachine) GetError() error {
	return m.error
}

// GetSection returns the section.
func (m *SectionMachine) GetSection() Section {
	return m.section
}

// HasEditSession returns true if there's an active edit session.
func (m *SectionMachine) HasEditSession() bool {
	return m.session != nil
}

// HasPendingAction returns true if there's a pending API call.
func (m *SectionMachine) HasPendingAction() bool {
	return m.pendingAction.ID != ""
}

// GetPendingActionID returns the ID of the pending action.
func (m *SectionMachine) GetPendingActionID() string {
	return m.pendingAction.ID
}

// State Handlers

func (m *SectionMachine) handleIdleEvent(evt SectionEvent) (SectionState, tea.Cmd) {
	switch e := evt.(type) {
	case EventStartEdit:
		m.startEditing(e.EntityType, e.EntityID, e.InitialState)
		logger.Info("handleIdleEvent: transitioning to StateEditing", "section", m.section)
		return m.transitionTo(StateEditing)

	case EventEditingStarted:
		// Session is already set, just transition to editing state
		logger.Info("handleIdleEvent: EventEditingStarted, transitioning to StateEditing", "section", m.section)
		return m.transitionTo(StateEditing)

	case EventDataLoadStart:
		return m.transitionTo(StateLoading)

	case EventNavigateAway:
		// Nothing to preserve, just mark
		return StateIdle, nil

	default:
		// Log unexpected events in idle state
		logger.Warn("handleIdleEvent: unexpected event", "section", m.section, "eventType", fmt.Sprintf("%T", evt))
		return StateIdle, nil
	}
}

func (m *SectionMachine) handleLoadingEvent(evt SectionEvent) (SectionState, tea.Cmd) {
	switch e := evt.(type) {
	case EventDataLoadComplete:
		if e.Err != nil {
			m.error = e.Err
			return m.transitionTo(StateError)
		}
		m.data = e.Data
		return m.transitionTo(StateIdle)

	case EventError:
		m.error = e.Err
		return m.transitionTo(StateError)

	default:
		return StateLoading, nil
	}
}

func (m *SectionMachine) handleEditingEvent(evt SectionEvent) (SectionState, tea.Cmd) {
	switch e := evt.(type) {
	case EventFieldChange:
		logger.Debug("EventFieldChange in handleEditingEvent", "field", e.Field, "sessionNil", m.session == nil)
		if m.session != nil {
			changed := m.session.SetFieldValue(e.Field, e.Value)
			logger.Debug("SetFieldValue result", "changed", changed, "hasChanges", m.session.HasChanges())
			if changed && m.session.HasChanges() {
				logger.Info("handleEditingEvent: transitioning to StatePendingEdit", "section", m.section)
				return m.transitionTo(StatePendingEdit)
			}
		} else {
			logger.Warn("handleEditingEvent: session is nil!", "section", m.section)
		}
		return StateEditing, nil

	case EventSubmit:
		logger.Info("EventSubmit in handleEditingEvent", "section", m.section, "sessionNil", m.session == nil, "hasChanges", m.session != nil && m.session.HasChanges())
		if m.session != nil && m.session.HasChanges() {
			// Queue the API call and transition
			m.queueSaveAction()
			logger.Info("handleEditingEvent: transitioning to StateAPICall", "section", m.section)
			return m.transitionTo(StateAPICall)
		}
		logger.Warn("EventSubmit ignored - no session or no changes", "sessionNil", m.session == nil)
		return StateEditing, nil

	case EventCancelEdit:
		m.clearSession()
		return m.transitionTo(StateIdle)

	case EventNavigateAway:
		// Session stays but mark as paused
		if m.session != nil {
			m.session.Paused = true
		}
		return StateEditing, nil

	default:
		return StateEditing, nil
	}
}

func (m *SectionMachine) handlePendingEditEvent(evt SectionEvent) (SectionState, tea.Cmd) {
	switch e := evt.(type) {
	case EventFieldChange:
		if m.session != nil {
			m.session.SetFieldValue(e.Field, e.Value)
			// Check if we reverted back to original
			if !m.session.HasChanges() {
				return m.transitionTo(StateEditing)
			}
		}
		return StatePendingEdit, nil

	case EventSubmit:
		m.queueSaveAction()
		return m.transitionTo(StateAPICall)

	case EventCancelEdit:
		m.clearSession()
		return m.transitionTo(StateIdle)

	case EventNavigateAway:
		if m.session != nil {
			m.session.Paused = true
		}
		return StatePendingEdit, nil

	default:
		return StatePendingEdit, nil
	}
}

func (m *SectionMachine) handleAPICallEvent(evt SectionEvent) (SectionState, tea.Cmd) {
	switch e := evt.(type) {
	case EventAPICallComplete:
		m.callInFlight = ""

		if e.Err != nil {
			m.error = e.Err
			if m.session != nil {
				m.session.SetError(e.Err)
			}
			return m.transitionTo(StateError)
		}

		// Success - update session with result
		if m.session != nil {
			m.session.UpdateFromResult(e.Result)
			m.data = e.Result
		}
		m.clearPendingAction()
		return m.transitionTo(StateIdle)

	case EventNavigateAway:
		// API call continues in background
		return StateAPICall, nil

	default:
		return StateAPICall, nil
	}
}

func (m *SectionMachine) handleErrorEvent(evt SectionEvent) (SectionState, tea.Cmd) {
	switch evt.(type) {
	case EventRetry:
		// Retry the failed operation
		if m.pendingAction.ID != "" {
			return m.transitionTo(StateAPICall)
		}
		if m.session != nil {
			// Retry edit
			return m.transitionTo(StateEditing)
		}
		// Retry load
		return m.transitionTo(StateLoading)

	case EventDismissError:
		m.error = nil
		if m.session != nil {
			m.session.SetError(nil)
			return m.transitionTo(StateEditing)
		}
		return m.transitionTo(StateIdle)

	case EventCancelEdit:
		m.clearSession()
		m.error = nil
		return m.transitionTo(StateIdle)

	case EventDataLoadStart:
		m.error = nil
		return m.transitionTo(StateLoading)

	default:
		return StateError, nil
	}
}

// Helper Methods

func (m *SectionMachine) startEditing(entityType, entityID string, initial map[string]interface{}) {
	m.session = NewEditSession(entityType, entityID, nil, initial)
}

func (m *SectionMachine) clearSession() {
	m.session = nil
	m.clearPendingAction()
}

// SetSession initializes the session directly (for edit overlay setup)
func (m *SectionMachine) SetSession(entityType, entityID string, fields []edit.FieldDef, initialState map[string]interface{}) {
	m.session = NewEditSession(entityType, entityID, fields, initialState)
	logger.Info("SetSession: session initialized", "section", m.section, "entityID", entityID)
}

func (m *SectionMachine) queueSaveAction() {
	if m.session == nil {
		return
	}

	m.pendingAction = PendingAction{
		ID:        generateCallID(),
		Type:      "update",
		EntityID:  m.session.EntityID,
		Changes:   m.session.GetChanges(),
		StartedAt: time.Now().Unix(),
	}
}

func (m *SectionMachine) clearPendingAction() {
	m.pendingAction = PendingAction{}
}

func (m *SectionMachine) transitionTo(newState SectionState) (SectionState, tea.Cmd) {
	oldState := m.state

	logger.Info("transitionTo called", "section", m.section, "from", oldState, "to", newState)

	// Validate transition is allowed
	if !CanTransition(oldState, newState) {
		// Log invalid transition attempt but stay in current state
		logger.Error("INVALID STATE TRANSITION", "section", m.section, "from", oldState, "to", newState)
		m.error = fmt.Errorf("invalid state transition from %s to %s", oldState, newState)
		return oldState, nil
	}

	m.state = newState
	m.error = nil // Clear any previous transition error

	// Generate command based on new state
	var cmd tea.Cmd

	switch newState {
	case StateLoading:
		cmd = m.createLoadCmd()
	case StateAPICall:
		logger.Info("transitionTo: about to call createAPICallCmd", "section", m.section)
		cmd = m.createAPICallCmd()
		logger.Info("transitionTo: createAPICallCmd returned", "section", m.section, "cmdNil", cmd == nil)
	}

	// Log transition
	_ = oldState // could log transition here

	return newState, cmd
}

func (m *SectionMachine) createLoadCmd() tea.Cmd {
	if m.service == nil {
		return nil
	}

	// This would be implemented by the caller
	// Return a no-op for now
	return nil
}

func (m *SectionMachine) createAPICallCmd() tea.Cmd {
	logger.Info("createAPICallCmd ENTER", "section", m.section, "serviceNil", m.service == nil, "pendingActionEmpty", m.pendingAction.IsEmpty())
	if m.service == nil {
		logger.Error("createAPICallCmd: service is NIL!", "section", m.section)
		return nil
	}

	action := m.pendingAction
	if action.IsEmpty() {
		logger.Error("createAPICallCmd: pendingAction is EMPTY!", "section", m.section)
		return nil
	}

	m.callInFlight = action.ID
	logger.Info("createAPICallCmd: about to return API call function", "section", m.section, "entityID", action.EntityID)

	return func() tea.Msg {
		logger.Info("createAPICallCmd function EXECUTING", "section", m.section, "entityID", action.EntityID)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		logger.Info("executing API call", "section", m.section, "entityID", action.EntityID, "changes", action.Changes)
		result, err := m.service.Update(ctx, action.EntityID, action.Changes)

		logger.Info("API call COMPLETE", "section", m.section, "entityID", action.EntityID, "err", err)
		return EventAPICallComplete{
			Section:  m.section,
			CallID:   action.ID,
			EntityID: action.EntityID,
			Result:   result,
			Err:      err,
		}
	}
}

func generateCallID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ShouldShowEdit returns true if edit should be visible.
func (m *SectionMachine) ShouldShowEdit() bool {
	return m.state == StateEditing ||
		m.state == StatePendingEdit ||
		m.state == StateAPICall
}

// GetSaveStatus returns a human-readable save status.
func (m *SectionMachine) GetSaveStatus() string {
	switch m.state {
	case StateAPICall:
		return "Saving..."
	case StatePendingEdit:
		return "Unsaved changes"
	case StateEditing:
		return "Editing"
	case StateError:
		if m.error != nil {
			return fmt.Sprintf("Error: %v", m.error)
		}
		return "Error occurred"
	default:
		return ""
	}
}
