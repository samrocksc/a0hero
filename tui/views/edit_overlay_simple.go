package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samrocksc/a0hero/modules/edit"
	"github.com/samrocksc/a0hero/tui/state"
)

// EditSubmitMsg is sent when user confirms save (ctrl+s).
// This is a proper tea.Msg that carries all data needed for the save operation.
type EditSubmitMsg struct {
	Section  state.Section
	EntityID string
	Changes  map[string]interface{}
}

// IsStateEvent implements state.SectionEvent for type safety.
func (e EditSubmitMsg) IsStateEvent() {}

// EditCancelMsg is sent when user cancels edit (esc).
type EditCancelMsg struct {
	Section  state.Section
	EntityID string
}

// SimpleEditOverlay is a streamlined edit view without mode switching.
// Navigation: j/k or ↑/↓ to move between fields
// Edit: enter or 'e' to edit focused field
// Save: ctrl+s to save all changes
// Cancel: esc to close without saving
type SimpleEditOverlay struct {
	entityType string
	entityID   string
	fields     []edit.FieldDef
	session    *edit.EditSession

	// Event-driven state machine support
	section   int
	emitEvent func(evt interface{}) tea.Msg

	// UI state - simplified, no modes
	fieldInputs   map[string]string   // current values for text/scalar fields
	tagInputs     map[string][]string // current values for tag array fields
	focusedField  int                 // which field is focused
	editingField  bool                // actively typing into focused field
	editBuffer    string              // text being typed
	editCursorPos int                 // cursor position in edit buffer

	// Status
	saving      bool
	savingStart time.Time
	successMsg  string
	errors      []string

	// Dimensions
	width  int
	height int

	// Callbacks
	onClose    func() tea.Msg
	historyDir string
}

// SimpleEditOverlayConfig for creating a new overlay
type SimpleEditOverlayConfig struct {
	EntityType string
	EntityID   string
	Fields     []edit.FieldDef
	OnClose    func() tea.Msg
	Section    int
	EmitEvent  func(evt interface{}) tea.Msg
	HistoryDir string
}

// NewSimpleEditOverlay creates a new simplified edit overlay.
func NewSimpleEditOverlay(cfg SimpleEditOverlayConfig) (*SimpleEditOverlay, tea.Cmd) {
	e := &SimpleEditOverlay{
		entityType: cfg.EntityType,
		entityID:   cfg.EntityID,
		fields:     cfg.Fields,
		onClose:    cfg.OnClose,
		section:    cfg.Section,
		emitEvent:  cfg.EmitEvent,
		historyDir: cfg.HistoryDir,
		fieldInputs: make(map[string]string),
		tagInputs:   make(map[string][]string),
	}
	return e, nil
}

// SetDimensions sets the overlay dimensions.
func (e *SimpleEditOverlay) SetDimensions(width, height int) {
	e.width = width
	e.height = height - 4 // Account for header/footer
}

// EntityID returns the entity being edited
func (e *SimpleEditOverlay) EntityID() string {
	return e.entityID
}

// HandleReady populates the overlay with session data.
func (e *SimpleEditOverlay) HandleReady(session *edit.EditSession) {
	e.session = session
	e.fieldInputs = make(map[string]string)
	e.tagInputs = make(map[string][]string)

	for _, field := range e.fields {
		val := e.getFieldValue(session, field.Key)
		switch field.Type {
		case edit.FieldTagArray:
			if arr, ok := val.([]string); ok {
				e.tagInputs[field.Key] = arr
			} else if arr, ok := val.([]interface{}); ok {
				strs := make([]string, len(arr))
				for i, v := range arr {
					strs[i] = fmt.Sprint(v)
				}
				e.tagInputs[field.Key] = strs
			} else {
				e.tagInputs[field.Key] = []string{}
			}
		default:
			e.fieldInputs[field.Key] = fmt.Sprint(val)
		}
	}
}

// HandleError displays an error message.
func (e *SimpleEditOverlay) HandleError(msg string) {
	e.errors = append(e.errors, msg)
	e.saving = false // Clear saving state on error
}

// Init implements tea.Model
func (e *SimpleEditOverlay) Init() tea.Cmd {
	return nil
}

// Update handles messages - returns tea.Model for interface compliance.
func (e *SimpleEditOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return e.handleKey(msg)
	}
	return e, nil
}

func (e *SimpleEditOverlay) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// While saving, only allow cancel
	if e.saving {
		switch msg.String() {
		case "esc":
			return e.emitCancel()
		default:
			return e, nil // Ignore other keys while saving
		}
	}

	// If actively editing a field
	if e.editingField {
		return e.handleFieldEdit(msg)
	}

	// Navigation and actions
	switch msg.String() {
	case "esc", "q":
		return e, e.onClose
	case "up", "k":
		if e.focusedField > 0 {
			e.focusedField--
		}
	case "down", "j":
		if e.focusedField < len(e.fields)-1 {
			e.focusedField++
		}
	case "enter", "e":
		return e.startFieldEdit()
	case "ctrl+s":
		return e, e.submit()
	case "ctrl+z":
		if e.session != nil && e.session.Undo() {
			e.syncFromSession()
		}
	}
	return e, nil
}

func (e *SimpleEditOverlay) handleFieldEdit(msg tea.KeyMsg) (*SimpleEditOverlay, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel edit, restore original value
		e.editingField = false
		e.editBuffer = ""
		return e, nil
	case "enter":
		// Confirm edit
		return e.confirmFieldEdit()
	case "left":
		if e.editCursorPos > 0 {
			e.editCursorPos--
		}
	case "right":
		if e.editCursorPos < len(e.editBuffer) {
			e.editCursorPos++
		}
	case "home":
		e.editCursorPos = 0
	case "end":
		e.editCursorPos = len(e.editBuffer)
	case "backspace":
		if e.editCursorPos > 0 {
			e.editBuffer = e.editBuffer[:e.editCursorPos-1] + e.editBuffer[e.editCursorPos:]
			e.editCursorPos--
		}
	case "delete":
		if e.editCursorPos < len(e.editBuffer) {
			e.editBuffer = e.editBuffer[:e.editCursorPos] + e.editBuffer[e.editCursorPos+1:]
		}
	case "ctrl+u":
		// Clear to beginning
		e.editBuffer = e.editBuffer[e.editCursorPos:]
		e.editCursorPos = 0
	case "ctrl+k":
		// Clear to end
		e.editBuffer = e.editBuffer[:e.editCursorPos]
	default:
		// Insert character
		if msg.Type == tea.KeyRunes {
			runes := []rune(e.editBuffer)
			newRunes := append(runes[:e.editCursorPos], msg.Runes...)
			newRunes = append(newRunes, runes[e.editCursorPos:]...)
			e.editBuffer = string(newRunes)
			e.editCursorPos += len(msg.Runes)
		}
	}
	return e, nil
}

func (e *SimpleEditOverlay) startFieldEdit() (*SimpleEditOverlay, tea.Cmd) {
	field := e.fields[e.focusedField]
	if field.ReadOnly {
		return e, nil
	}

	e.editingField = true
	switch field.Type {
	case edit.FieldTagArray:
		tags := e.tagInputs[field.Key]
		e.editBuffer = strings.Join(tags, ", ")
	default:
		e.editBuffer = e.fieldInputs[field.Key]
	}
	e.editCursorPos = len(e.editBuffer)
	return e, nil
}

func (e *SimpleEditOverlay) confirmFieldEdit() (*SimpleEditOverlay, tea.Cmd) {
	field := e.fields[e.focusedField]

	switch field.Type {
	case edit.FieldTagArray:
		// Parse comma-separated values
		tags := []string{}
		for _, t := range strings.Split(e.editBuffer, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
		e.tagInputs[field.Key] = tags
	default:
		e.fieldInputs[field.Key] = e.editBuffer
	}

	e.editingField = false
	e.editBuffer = ""
	return e, nil
}

func (e *SimpleEditOverlay) emitCancel() (*SimpleEditOverlay, tea.Cmd) {
	// Return proper tea.Msg that App.Update will handle
	return e, func() tea.Msg {
		return EditCancelMsg{
			Section:  state.Section(e.section),
			EntityID: e.entityID,
		}
	}
}

func (e *SimpleEditOverlay) submit() tea.Cmd {
	e.saving = true
	e.savingStart = time.Now()
	e.successMsg = ""
	e.errors = nil

	changes := make(map[string]interface{})
	for _, field := range e.fields {
		if field.ReadOnly {
			continue
		}
		switch field.Type {
		case edit.FieldTagArray:
			if tags, ok := e.tagInputs[field.Key]; ok {
				changes[field.Key] = tags
			}
		default:
			if val, ok := e.fieldInputs[field.Key]; ok {
				changes[field.Key] = val
			}
		}
	}

	// Return proper tea.Msg that App.Update will handle
	return func() tea.Msg {
		return EditSubmitMsg{
			Section:  state.Section(e.section),
			EntityID: e.entityID,
			Changes:  changes,
		}
	}
}

func (e *SimpleEditOverlay) syncFromSession() {
	if e.session == nil {
		return
	}
	for _, field := range e.fields {
		val := e.getFieldValue(e.session, field.Key)
		switch field.Type {
		case edit.FieldTagArray:
			if arr, ok := val.([]string); ok {
				e.tagInputs[field.Key] = arr
			}
		default:
			e.fieldInputs[field.Key] = fmt.Sprint(val)
		}
	}
}

func (e *SimpleEditOverlay) getFieldValue(session *edit.EditSession, key string) interface{} {
	if session == nil {
		return ""
	}
	if session.Current != nil {
		if v, ok := session.Current[key]; ok {
			return v
		}
	}
	if session.Original != nil {
		if v, ok := session.Original[key]; ok {
			return v
		}
	}
	return ""
}

// View renders the overlay.
func (e *SimpleEditOverlay) View() string {
	if e.width == 0 || e.height == 0 {
		return ""
	}

	var b strings.Builder

	// Header
	title := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("  Edit %s: %s", e.entityType, e.entityID))
	b.WriteString(title)

	if e.saving {
		status := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8800")).Bold(true).Render(" [SAVING... Esc to cancel]")
		b.WriteString(status)
	}
	b.WriteString("\n")

	// Show errors if any
	for _, err := range e.errors {
		errStr := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Bold(true).
			Render(fmt.Sprintf("  ✗ %s", err))
		b.WriteString(errStr)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Content area
	contentHeight := e.height - 6 // Header + footer
	visibleFields := contentHeight / 3
	startIdx := 0
	if e.focusedField >= visibleFields {
		startIdx = e.focusedField - visibleFields + 1
	}
	endIdx := startIdx + visibleFields
	if endIdx > len(e.fields) {
		endIdx = len(e.fields)
	}

	// Show fields
	for i := startIdx; i < endIdx && i < len(e.fields); i++ {
		field := e.fields[i]
		b.WriteString(e.renderField(field, i == e.focusedField, i == e.focusedField && e.editingField))
		b.WriteString("\n")
	}

	// Footer help
	b.WriteString("\n")
	if e.editingField {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(
			" esc: cancel  •  enter: confirm  •  ←→: move cursor  •  backspace: delete"))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(
			" esc: close  •  ↑/k ↓/j: navigate  •  enter/e: edit  •  ctrl+s: save  •  ctrl+z: undo"))
	}

	return b.String()
}

func (e *SimpleEditOverlay) renderField(field edit.FieldDef, focused, editing bool) string {
	var b strings.Builder

	// Label style
	labelStyle := lipgloss.NewStyle()
	if focused && !editing {
		labelStyle = labelStyle.Foreground(lipgloss.Color("#00FFFF")).Bold(true)
	}
	if focused && editing {
		labelStyle = labelStyle.Foreground(lipgloss.Color("#FFD700")).Bold(true)
	}
	if field.ReadOnly {
		labelStyle = labelStyle.Foreground(lipgloss.Color("#888888"))
	}

	label := labelStyle.Render(fmt.Sprintf("%s:", field.Label))
	b.WriteString("  ")
	b.WriteString(label)
	b.WriteString(" ")

	// Value style
	valueStyle := lipgloss.NewStyle()
	if focused {
		valueStyle = valueStyle.Background(lipgloss.Color("#333333"))
	}
	if editing {
		valueStyle = valueStyle.Background(lipgloss.Color("#224422"))
	}

	// Get value to display
	var valueStr string
	if editing {
		// Show edit buffer with cursor
		valueStr = e.renderEditBuffer()
	} else {
		switch field.Type {
		case edit.FieldTagArray:
			tags := e.tagInputs[field.Key]
			if len(tags) == 0 {
				valueStr = "(none)"
			} else {
				valueStr = strings.Join(tags, ", ")
			}
		default:
			valueStr = e.fieldInputs[field.Key]
			if valueStr == "" {
				valueStr = "(empty)"
			}
		}
	}

	// Truncate if too long
	maxWidth := e.width - 30
	if len(valueStr) > maxWidth {
		valueStr = valueStr[:maxWidth-3] + "..."
	}

	b.WriteString(valueStyle.Render(valueStr))
	return b.String()
}

func (e *SimpleEditOverlay) renderEditBuffer() string {
	// Show cursor in edit buffer
	if e.editCursorPos < 0 || e.editCursorPos > len(e.editBuffer) {
		e.editCursorPos = len(e.editBuffer)
	}

	before := e.editBuffer[:e.editCursorPos]
	after := e.editBuffer[e.editCursorPos:]

	// Cursor character
	cursor := "█"
	if e.editCursorPos < len(e.editBuffer) {
		cursor = string(e.editBuffer[e.editCursorPos])
		cursor = lipgloss.NewStyle().Reverse(true).Render(cursor)
	}

	return before + cursor + after
}
