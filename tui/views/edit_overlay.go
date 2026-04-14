package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/samrocksc/a0hero/logger"
	"github.com/samrocksc/a0hero/modules/edit"
)

// EditOverlay handles the inline edit view for an entity.
type EditOverlay struct {
	entityType string
	entityID   string
	fields     []edit.FieldDef
	session    *edit.EditSession
	service    edit.EntityService

	// UI state
	fieldInputs   map[string]string   // current values for text/scalar fields
	tagInputs     map[string][]string  // current values for tag array fields
	focusedField  int
	mode          editMode
	errors        []string
	successMsg    string
	loading       bool
	dirty         bool

	// Field editing sub-state (when typing a value)
	editing       bool   // true when actively typing into a field
	editBuffer    string // the text being typed
	editFieldKey  string // which field key we're editing
	editCursorPos int    // cursor position in edit buffer

	// Dimensions
	width  int
	height int

	// Callbacks
	onClose    func() tea.Msg
	onSave     func(map[string]interface{}) tea.Msg
	historyDir string
}

type editMode int

const (
	modeView editMode = iota
	modeEdit   // navigating fields, press enter to start typing
	modeSaving
)

// Messages
type EditOverlayReady struct {
	Session    *edit.EditSession
	HistoryDir string
}

type EditOverlayError struct {
	Message string
}

type EditOverlaySaved struct {
	Data map[string]interface{}
}

// NewEditOverlay creates a new edit overlay.
func NewEditOverlay(cfg EditOverlayConfig) (*EditOverlay, tea.Cmd) {
	e := &EditOverlay{
		entityType: cfg.EntityType,
		entityID:   cfg.EntityID,
		fields:     cfg.Fields,
		service:    cfg.Service,
		onClose:    cfg.OnClose,
		onSave:     cfg.OnSave,
		historyDir: cfg.HistoryDir,
		mode:       modeView,
		loading:    true,
	}
	return e, e.fetchCurrentState(cfg.HistoryDir)
}

// SetDimensions sets the overlay dimensions.
func (e *EditOverlay) SetDimensions(width, height int) {
	e.width = width
	if height > 6 {
		e.height = height - 4
	} else {
		e.height = height
	}
}

// EntityID returns the entity ID.
func (e *EditOverlay) EntityID() string { return e.entityID }

// HandleReady handles the EditOverlayReady message.
func (e *EditOverlay) HandleReady(session *edit.EditSession) {
	e.session = session
	e.loading = false
	e.dirty = false
	e.fieldInputs = make(map[string]string)
	e.tagInputs = make(map[string][]string)
	for _, field := range e.fields {
		val := e.getFieldValue(session, field.Key)
		switch field.Type {
		case edit.FieldTagArray:
			if arr, ok := val.([]string); ok {
				e.tagInputs[field.Key] = arr
			} else {
				e.tagInputs[field.Key] = []string{}
			}
		default:
			e.fieldInputs[field.Key] = fmt.Sprintf("%v", val)
		}
	}
}

// HandleError handles edit overlay errors.
func (e *EditOverlay) HandleError(msg string) {
	e.loading = false
	e.errors = append(e.errors, msg)
}

// HandleSaved handles the save completion.
func (e *EditOverlay) HandleSaved() {
	e.loading = false
	e.successMsg = "Saved!"
	e.mode = modeView
	e.dirty = false
}

// Loading returns whether the overlay is still loading.
func (e *EditOverlay) Loading() bool { return e.loading }

// GetSession returns the edit session.
func (e *EditOverlay) GetSession() *edit.EditSession { return e.session }

// GetErrors returns any errors.
func (e *EditOverlay) GetErrors() []string { return e.errors }

// Init initializes the overlay.
func (e *EditOverlay) Init() tea.Cmd { return nil }

// Update handles messages.
func (e *EditOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		e.width = msg.Width
		e.height = msg.Height
		return e, nil
	case EditOverlayReady:
		e.HandleReady(msg.Session)
		return e, nil
	case EditOverlayError:
		e.HandleError(msg.Message)
		return e, nil
	case EditOverlaySaved:
		e.HandleSaved()
		return e, nil
	case tea.KeyMsg:
		return e.handleKey(msg)
	}
	return e, nil
}

// handleKey routes key events based on current state.
func (e *EditOverlay) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If actively editing a field value, handle input
	if e.editing {
		return e.handleFieldInput(msg)
	}

	switch e.mode {
	case modeView:
		return e.handleViewKey(msg)
	case modeEdit:
		return e.handleEditKey(msg)
	case modeSaving:
		return e, nil
	}
	return e, nil
}

func (e *EditOverlay) handleViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		return e, e.onClose
	case "e":
		e.mode = modeEdit
		e.focusedField = 0
	case "up", "k":
		if e.focusedField > 0 {
			e.focusedField--
		}
	case "down", "j":
		if e.focusedField < len(e.fields)-1 {
			e.focusedField++
		}
	}
	return e, nil
}

func (e *EditOverlay) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		e.mode = modeView
		return e, nil
	case "ctrl+s":
		return e, e.submit()
	case "ctrl+z":
		if e.session != nil && e.session.Undo() {
			e.syncInputsFromSession()
		}
		return e, nil
	case "up", "k":
		if e.focusedField > 0 {
			e.focusedField--
		}
	case "down", "j", "tab":
		if e.focusedField < len(e.fields)-1 {
			e.focusedField++
		}
	case "shift+tab":
		if e.focusedField > 0 {
			e.focusedField--
		}
	case "enter":
		// Start editing the focused field
		field := e.fields[e.focusedField]
		if field.ReadOnly {
			return e, nil
		}
		e.editing = true
		e.editFieldKey = field.Key
		e.editCursorPos = 0
		// Pre-fill with current value
		switch field.Type {
		case edit.FieldTagArray:
			if tags, ok := e.tagInputs[field.Key]; ok {
				e.editBuffer = strings.Join(tags, ", ")
			}
		default:
			e.editBuffer = e.fieldInputs[field.Key]
		}
		e.editCursorPos = len(e.editBuffer)
	}
	return e, nil
}

// handleFieldInput handles keystrokes while typing into a field.
func (e *EditOverlay) handleFieldInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Save the typed value back to state, stop editing, return to field nav
		e.commitEditBuffer()
		e.editing = false
		e.dirty = true
		return e, nil
	case "enter":
		// For tag arrays, add a tag; for text, commit and move to next field
		field := e.getFieldDef(e.editFieldKey)
		if field != nil && field.Type == edit.FieldTagArray {
			// Add the buffer as a new tag
			tag := strings.TrimSpace(e.editBuffer)
			if tag != "" {
				e.tagInputs[e.editFieldKey] = append(e.tagInputs[e.editFieldKey], tag)
				e.dirty = true
			}
			e.editBuffer = ""
			e.editCursorPos = 0
		} else {
			// Commit text value and move down
			e.commitEditBuffer()
			e.editing = false
			e.dirty = true
			// Move to next field
			if e.focusedField < len(e.fields)-1 {
				e.focusedField++
			}
		}
		return e, nil
	case "ctrl+z":
		// Undo during editing: revert to original
		if e.session != nil {
			val := e.getFieldValue(e.session, e.editFieldKey)
			switch e.getFieldDef(e.editFieldKey).Type {
			case edit.FieldTagArray:
				if arr, ok := val.([]string); ok {
					e.editBuffer = strings.Join(arr, ", ")
				}
			default:
				e.editBuffer = fmt.Sprintf("%v", val)
			}
			e.editCursorPos = len(e.editBuffer)
		}
		return e, nil
	case "backspace":
		if e.editCursorPos > 0 {
			e.editBuffer = e.editBuffer[:e.editCursorPos-1] + e.editBuffer[e.editCursorPos:]
			e.editCursorPos--
		}
	case "left":
		if e.editCursorPos > 0 {
			e.editCursorPos--
		}
	case "right":
		if e.editCursorPos < len(e.editBuffer) {
			e.editCursorPos++
		}
	default:
		// Type the character into the buffer
		ch := msg.String()
		// Filter out multi-char special keys
		if len(ch) == 1 || (len(ch) > 1 && ch[0] != '[' && ch[0] != 27) {
			// For tag arrays, don't insert raw keys
			if len(ch) == 1 {
				e.editBuffer = e.editBuffer[:e.editCursorPos] + ch + e.editBuffer[e.editCursorPos:]
				e.editCursorPos++
			}
		}
	}
	return e, nil
}

// commitEditBuffer writes the edit buffer back to the field value store.
func (e *EditOverlay) commitEditBuffer() {
	field := e.getFieldDef(e.editFieldKey)
	if field == nil {
		return
	}
	switch field.Type {
	case edit.FieldTagArray:
		// Parse comma-separated tags
		parts := strings.Split(e.editBuffer, ",")
		tags := []string{}
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				tags = append(tags, p)
			}
		}
		e.tagInputs[e.editFieldKey] = tags
	default:
		e.fieldInputs[e.editFieldKey] = e.editBuffer
	}
}

func (e *EditOverlay) getEditableFields() []edit.FieldDef {
	var result []edit.FieldDef
	for _, f := range e.fields {
		if !f.ReadOnly {
			result = append(result, f)
		}
	}
	return result
}

func (e *EditOverlay) getFieldDef(key string) *edit.FieldDef {
	for i := range e.fields {
		if e.fields[i].Key == key {
			return &e.fields[i]
		}
	}
	return nil
}

// submit saves the changes.
func (e *EditOverlay) submit() tea.Cmd {
	e.mode = modeSaving
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
	if e.onSave != nil {
		return func() tea.Msg { return e.onSave(changes) }
	}
	return nil
}

// fetchCurrentState fetches the current entity state.
func (e *EditOverlay) fetchCurrentState(historyDir string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		state, err := e.service.Fetch(ctx, e.entityID)
		if err != nil {
			logger.Error("failed to fetch entity", "type", e.entityType, "id", e.entityID, "error", err)
			if ctx.Err() == context.DeadlineExceeded {
				return EditOverlayError{Message: "Request timed out after 10 seconds"}
			}
			return EditOverlayError{Message: fmt.Sprintf("Failed to fetch: %v", err)}
		}

		session := edit.NewSession(e.entityType, e.entityID, e.fields, state)
		return EditOverlayReady{Session: session, HistoryDir: historyDir}
	}
}

// View renders the overlay.
func (e *EditOverlay) View() string {
	if e.width == 0 || e.height == 0 {
		return "Loading..."
	}

	if e.loading {
		return lipgloss.NewStyle().Width(e.width).Render("Loading...")
	}

	if len(e.errors) > 0 {
		return e.renderErrors()
	}

	var b strings.Builder

	// Header
	modeLabel := "VIEW"
	modeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	if e.mode == modeEdit || e.editing {
		modeLabel = "EDIT"
		modeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true)
	}
	if e.mode == modeSaving {
		modeLabel = "SAVING..."
		modeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8800")).Bold(true)
	}

	header := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("  %s: %s", e.entityType, e.entityID))
	b.WriteString(header)
	b.WriteString("  ")
	b.WriteString(modeStyle.Render(fmt.Sprintf("[%s]", modeLabel)))
	b.WriteString("\n\n")

	// Success message
	if e.successMsg != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("  ✓ " + e.successMsg))
		b.WriteString("\n\n")
		e.successMsg = ""
	}

	// Fields
	for i, field := range e.fields {
		isFocused := i == e.focusedField
		b.WriteString(e.renderField(field, isFocused))
		b.WriteString("\n")
	}

	// Footer help
	b.WriteString("\n")
	if e.editing {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(
			" esc: confirm & back  •  enter: confirm & next  •  ctrl+z: revert"))
	} else if e.mode == modeEdit {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(
			" esc: back to view  •  enter: edit field  •  ctrl+s: save all  •  ctrl+z: undo  •  ↑↓: navigate"))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(
			" esc: close  •  e: edit  •  ↑↓: navigate"))
	}

	return b.String()
}

func (e *EditOverlay) renderErrors() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true).Render("  ✗ Error"))
	b.WriteString("\n\n")
	for _, err := range e.errors {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6666")).Render(fmt.Sprintf("  • %s", err)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("  Press esc to close"))
	return b.String()
}

func (e *EditOverlay) renderField(field edit.FieldDef, focused bool) string {
	labelWidth := 24
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Width(labelWidth)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))
	focusedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#333355"))
	readOnlyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	editingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#224488"))

	label := labelStyle.Render(field.Label + ":")

	// We're actively editing this field — show the cursor
	if e.editing && e.editFieldKey == field.Key {
		before := e.editBuffer[:e.editCursorPos]
		after := e.editBuffer[e.editCursorPos:]
		cursor := "▎"
		rendered := editingStyle.Render(before + cursor + after)
		return fmt.Sprintf("%s %s", label, rendered)
	}

	// Get current value
	var value string
	switch field.Type {
	case edit.FieldTagArray:
		if tags, ok := e.tagInputs[field.Key]; ok {
			if len(tags) == 0 {
				value = "(none)"
			} else {
				value = strings.Join(tags, ", ")
			}
		}
	default:
		if v, ok := e.fieldInputs[field.Key]; ok {
			value = v
		}
	}

	// Redact sensitive fields
	if field.Sensitive {
		value = "••••••••"
	}

	// Style based on state
	if field.ReadOnly {
		return fmt.Sprintf("%s %s", label, readOnlyStyle.Render(value))
	}

	if e.mode == modeEdit && focused {
		prefix := "▶ "
		return fmt.Sprintf("%s %s", label, focusedStyle.Render(prefix+value))
	}

	if e.mode == modeEdit {
		return fmt.Sprintf("%s %s", label, valueStyle.Render(value))
	}

	return fmt.Sprintf("%s %s", label, valueStyle.Render(value))
}

func (e *EditOverlay) getFieldValue(session *edit.EditSession, key string) interface{} {
	if session == nil || session.Current == nil {
		return nil
	}
	return session.Current[key]
}

func (e *EditOverlay) syncInputsFromSession() {
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
			e.fieldInputs[field.Key] = fmt.Sprintf("%v", val)
		}
	}
}

// EditOverlayConfig holds configuration for creating an edit overlay.
type EditOverlayConfig struct {
	EntityType string
	EntityID   string
	Fields     []edit.FieldDef
	Service    edit.EntityService
	OnClose    func() tea.Msg
	OnSave     func(map[string]interface{}) tea.Msg
	HistoryDir string
}