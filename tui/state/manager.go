// Package state provides event-driven state management for TUI sections.
package state

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/samrocksc/a0hero/logger"
)

// StateManager coordinates state machines for all sections.
type StateManager struct {
	machines map[Section]*SectionMachine
	current  Section
}

// NewStateManager creates a new manager with machines for all sections.
func NewStateManager(services map[Section]EntityService) *StateManager {
	m := &StateManager{
		machines: make(map[Section]*SectionMachine),
		current:  SecUsers, // default
	}

	for sec := Section(0); sec < SecCount; sec++ {
		var svc EntityService
		if services != nil {
			svc = services[sec]
		}
		m.machines[sec] = NewSectionMachine(sec, WithService(svc))
	}

	return m
}

// ProcessEvent routes an event to the appropriate section's machine.
func (sm *StateManager) ProcessEvent(evt SectionEvent) tea.Cmd {
	section := evt.GetSection()
	logger.Info("StateManager.ProcessEvent ENTER", "section", section)
	machine, ok := sm.machines[section]
	if !ok {
		logger.Warn("StateManager.ProcessEvent: no machine for section", "section", section)
		return nil
	}

	logger.Info("StateManager.ProcessEvent: found machine, calling ProcessEvent", "section", section)
	_, cmd := machine.ProcessEvent(evt)
	logger.Info("StateManager.ProcessEvent EXIT", "section", section, "cmdNil", cmd == nil)
	return cmd
}

// GetMachine returns the state machine for a section.
func (sm *StateManager) GetMachine(section Section) *SectionMachine {
	if m, ok := sm.machines[section]; ok {
		return m
	}
	return nil
}

// GetCurrentMachine returns the machine for the current section.
func (sm *StateManager) GetCurrentMachine() *SectionMachine {
	return sm.GetMachine(sm.current)
}

// SetCurrentSection updates the current section and emits navigation events.
func (sm *StateManager) SetCurrentSection(section Section) []tea.Cmd {
	var cmds []tea.Cmd

	if sm.current != section {
		// Emit navigate away from old section
		if oldMachine := sm.GetCurrentMachine(); oldMachine != nil {
			_, cmd := oldMachine.ProcessEvent(EventNavigateAway{Section: sm.current})
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		sm.current = section

		// Emit navigate to new section
		if newMachine := sm.GetCurrentMachine(); newMachine != nil {
			_, cmd := newMachine.ProcessEvent(EventNavigateTo{Section: section})
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return cmds
}

// GetCurrentSection returns the current section.
func (sm *StateManager) GetCurrentSection() Section {
	return sm.current
}

// IsActive returns true if the section is currently active.
func (sm *StateManager) IsActive(section Section) bool {
	return sm.current == section
}

// HasPendingWork returns true if any section has unsaved changes or pending API calls.
func (sm *StateManager) HasPendingWork() bool {
	for _, machine := range sm.machines {
		if machine.GetState().HasPendingChanges() || machine.GetState().IsAPICallInProgress() {
			return true
		}
	}
	return false
}

// GetSectionsWithPendingWork returns sections that have unsaved work.
func (sm *StateManager) GetSectionsWithPendingWork() []Section {
	var sections []Section
	for sec, machine := range sm.machines {
		if machine.GetState().HasPendingChanges() || machine.GetState().IsAPICallInProgress() {
			sections = append(sections, sec)
		}
	}
	return sections
}

// UpdateService updates the service for a section.
func (sm *StateManager) UpdateService(section Section, svc EntityService) {
	if machine, ok := sm.machines[section]; ok {
		machine.service = svc
	}
}

// ClearAll resets all machines to idle state.
// Useful for testing or logout scenarios.
func (sm *StateManager) ClearAll() {
	for _, machine := range sm.machines {
		machine.state = StateIdle
		machine.clearSession()
		machine.clearPendingAction()
		machine.data = nil
		machine.error = nil
	}
}

// SubscribeToSection returns a channel for events from a specific section.
// This is a placeholder - in practice, events flow through tea.Msg.
type SectionStateSnapshot struct {
	Section   Section
	State     SectionState
	HasEdit   bool
	HasChanges bool
	IsSaving  bool
	Error     error
}

// Snapshot returns the current state snapshot for all sections.
func (sm *StateManager) Snapshot() []SectionStateSnapshot {
	var snaps []SectionStateSnapshot
	for sec, machine := range sm.machines {
		state := machine.GetState()
		snaps = append(snaps, SectionStateSnapshot{
			Section:    sec,
			State:      state,
			HasEdit:    machine.HasEditSession(),
			HasChanges: state.HasPendingChanges(),
			IsSaving:   state.IsAPICallInProgress(),
			Error:      machine.GetError(),
		})
	}
	return snaps
}
