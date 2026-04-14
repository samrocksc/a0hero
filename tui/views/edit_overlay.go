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
	"github.com/samrocksc/a0hero/tui/views/components"
)

// EditOverlay handles the edit mode overlay.
type EditOverlay struct {
	entityType string
	entityID   string
	fields     []edit.FieldDef
	session    *edit.EditSession
	service    edit.EntityService

	// UI state
	fieldInputs   map[string]components.TagInputModel
	focusedField  int
	mode          editMode // view, edit, saving
	errors        []string
	showErrorPopup bool
	errorPopup    *components.ErrorPopup
	confirmDialog *components.ConfirmDialog
	successMsg    string
	loading       bool
	inputValue    string // for text inputs

	// Dimensions
	width  int
	height int

	// Callbacks
	onClose func() tea.Msg
	onSave  func(map[string]interface{}) tea.Msg
}

// editMode represents the current editing mode.
type editMode int

const (
	modeView editMode = iota
	modeEdit
	modeSaving
)

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

// NewEditOverlay creates a new edit overlay.
func NewEditOverlay(cfg EditOverlayConfig) (*EditOverlay, tea.Cmd) {
	e := &EditOverlay{
		entityType: cfg.EntityType,
		entityID:   cfg.EntityID,
		fields:     cfg.Fields,
		service:    cfg.Service,
		onClose:    cfg.OnClose,
		onSave:     cfg.OnSave,
		fieldInputs: make(map[string]components.TagInputModel),
		mode:       modeView,
		loading:    true,
	}

	// Start fetching current state
	cmd := e.fetchCurrentState(cfg.HistoryDir)

	return e, cmd
}

// fetchCurrentState fetches the current entity state from the API.
func (e *EditOverlay) fetchCurrentState(historyDir string) tea.Cmd {
	return func() tea.Msg {
		// Create a context with 10 second timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Fetch current state
		state, err := e.service.Fetch(ctx, e.entityID)
		if err != nil {
			logger.Error("failed to fetch entity", "type", e.entityType, "id", e.entityID, "error", err)
			// Check if it was a timeout
			if ctx.Err() == context.DeadlineExceeded {
				return EditOverlayError{Message: "Request timed out after 10 seconds"}
			}
			return EditOverlayError{Message: fmt.Sprintf("Failed to fetch: %v", err)}
		}

		// Create edit session
		session := edit.NewSession(e.entityType, e.entityID, e.fields, state)

		// Initialize tag inputs
		for _, field := range e.fields {
			if field.Type == edit.FieldTagArray {
				val := e.getFieldValue(session, field.Key)
				if arr, ok := val.([]string); ok {
					e.fieldInputs[field.Key] = components.NewTagInputModel(field.Label, arr)
				}
			}
		}

		return EditOverlayReady{
			Session:    session,
			HistoryDir: historyDir,
		}
	}
}

// Init initializes the overlay.
func (e *EditOverlay) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (e *EditOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		e.width = msg.Width
		e.height = msg.Height
		return e, nil

	case EditOverlayReady:
		e.session = msg.Session
		e.loading = false
		// Initialize tag inputs
		for _, field := range e.fields {
			if field.Type == edit.FieldTagArray {
				val := e.getFieldValue(e.session, field.Key)
				if arr, ok := val.([]string); ok {
					e.fieldInputs[field.Key] = components.NewTagInputModel(field.Label, arr)
				}
			}
		}
		return e, nil

	case EditOverlayError:
		e.loading = false
		e.errors = append(e.errors, msg.Message)
		return e, nil

	case tea.KeyMsg:
		return e.handleKey(msg)
	}

	// Handle error popup
	if e.showErrorPopup && e.errorPopup != nil {
		popup, _ := e.errorPopup.Update(msg)
		e.errorPopup = popup.(*components.ErrorPopup)
		return e, nil
	}

	// Handle confirm dialog
	if e.confirmDialog != nil {
		dlg, _ := e.confirmDialog.Update(msg)
		e.confirmDialog = dlg.(*components.ConfirmDialog)

		// Check for result via tea.Msg
		switch msg.(type) {
		case components.ConfirmResult:
			result := msg.(components.ConfirmResult)
			if result.Button == "Discard" {
				return e, e.onClose
			}
			e.confirmDialog = nil
		}
		return e, nil
	}

	return e, tea.Batch(cmds...)
}

// handleKey handles key events.
func (e *EditOverlay) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle mode switching
	switch e.mode {
	case modeView:
		switch msg.String() {
		case "e":
			e.mode = modeEdit
			e.focusedField = 0
		case "esc", "q":
			return e, e.onClose
		}

	case modeEdit:
		switch msg.String() {
		case "esc":
			if e.session.IsDirty() {
				e.confirmDialog = components.DirtyConfirmDialog()
				e.confirmDialog.SetWidth(50)
			} else {
				e.mode = modeView
			}

		case "ctrl+s":
			cmds = append(cmds, e.submit())

		case "ctrl+z":
			if e.session.Undo() {
				e.syncInputsFromSession()
			}

		case "up", "k":
			if e.focusedField > 0 {
				e.focusedField--
			}

		case "down", "j", "tab":
			if e.focusedField < len(e.getEditableFields())-1 {
				e.focusedField++
			}

		case "shift+tab":
			if e.focusedField > 0 {
				e.focusedField--
			}

		default:
			// Forward to focused field
			e.handleFieldInput(msg)
		}

	case modeSaving:
		// Ignore input while saving
	}

	return e, tea.Batch(cmds...)
}

// handleFieldInput handles input for the currently focused field.
func (e *EditOverlay) handleFieldInput(msg tea.KeyMsg) {
	fields := e.getEditableFields()
	if e.focusedField >= len(fields) {
		return
	}

	field := fields[e.focusedField]

	switch field.Type {
	case edit.FieldTagArray:
		if input, ok := e.fieldInputs[field.Key]; ok {
			input.Focus()
			updated, _ := input.Update(msg)
			e.fieldInputs[field.Key] = updated.(components.TagInputModel)

			// Sync to session
			e.session.SetValue(field.Key, e.fieldInputs[field.Key].Value())
		}

	case edit.FieldText, edit.FieldURL, edit.FieldTextarea:
		switch msg.String() {
		case "backspace":
			if len(e.inputValue) > 0 {
				e.inputValue = e.inputValue[:len(e.inputValue)-1]
				e.session.SetValue(field.Key, e.inputValue)
			}
		default:
			// Handle regular character input
			s := msg.String()
			if len(s) == 1 && s >= " " && s <= "~" {
				e.inputValue += s
				e.session.SetValue(field.Key, e.inputValue)
			}
		}
	}
}

// syncInputsFromSession syncs inputs from session values.
func (e *EditOverlay) syncInputsFromSession() {
	e.inputValue = ""

	for _, field := range e.fields {
		if field.Type == edit.FieldTagArray {
			val := e.getFieldValue(e.session, field.Key)
			if arr, ok := val.([]string); ok {
				e.fieldInputs[field.Key] = components.NewTagInputModel(field.Label, arr)
			}
		} else if field.Type == edit.FieldText || field.Type == edit.FieldURL {
			val := e.getFieldValue(e.session, field.Key)
			if str, ok := val.(string); ok {
				e.inputValue = str
			}
		}
	}
}

// getEditableFields returns editable fields.
func (e *EditOverlay) getEditableFields() []edit.FieldDef {
	var fields []edit.FieldDef
	for _, f := range e.fields {
		if !f.ReadOnly {
			fields = append(fields, f)
		}
	}
	return fields
}

// getFieldValue gets a field value from the current state.
func (e *EditOverlay) getFieldValue(session *edit.EditSession, key string) interface{} {
	if session == nil {
		return nil
	}
	return session.Current[key]
}

// submit saves the changes.
func (e *EditOverlay) submit() tea.Cmd {
	// Collect changes from session
	changes := e.session.GetChangedFields()

	// Validate
	errors := e.validate(changes)
	if len(errors) > 0 {
		e.errors = errors
		e.showErrorPopup = true
		e.errorPopup = components.NewErrorPopup("Validation Failed", errors)
		e.errorPopup.SetWidth(e.width - 20)
		return nil
	}

	// Save
	e.mode = modeSaving

	return func() tea.Msg {
		ctx := context.Background()

		start := time.Now()

		// Call onSave callback
		result, err := e.service.Update(ctx, e.entityID, changes)

		duration := time.Since(start).Milliseconds()

		if err != nil {
			// Write failed history
			e.writeHistory(edit.StatusFailed, &edit.APIResponse{
				StatusCode: 0,
				DurationMs: duration,
				Error:      err.Error(),
			})

			return EditOverlayError{Message: fmt.Sprintf("Save failed: %v", err)}
		}

		// Write success history
		e.writeHistory(edit.StatusSaved, &edit.APIResponse{
			StatusCode: 200,
			DurationMs: duration,
		})

		e.successMsg = "Changes saved successfully"
		e.mode = modeView

		return EditOverlaySaved{
			Entity: result,
		}
	}
}

// validate validates the changes.
func (e *EditOverlay) validate(changes map[string]interface{}) []string {
	var errors []string

	for key, value := range changes {
		field := e.getFieldDef(key)
		if field == nil {
			continue
		}

		// Run validators based on field type
		switch field.Type {
		case edit.FieldURL:
			if err := edit.ValidateURL(value); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", field.Label, err))
			}
		case edit.FieldTagArray:
			if arr, ok := value.([]string); ok {
				if err := edit.ValidateURLArray(arr); err != nil {
					errors = append(errors, fmt.Sprintf("%s: %v", field.Label, err))
				}
			}
		}
	}

	return errors
}

// getFieldDef returns a field definition by key.
func (e *EditOverlay) getFieldDef(key string) *edit.FieldDef {
	for i := range e.fields {
		if e.fields[i].Key == key {
			return &e.fields[i]
		}
	}
	return nil
}

// writeHistory writes the history entry.
func (e *EditOverlay) writeHistory(status string, apiResp *edit.APIResponse) {
	if e.session == nil {
		return
	}
	entry := e.session.ToHistoryEntry(status, apiResp)
	writer := edit.NewHistoryWriter("")
	if err := writer.Write(entry); err != nil {
		logger.Error("failed to write history", "error", err)
	}
}

// View renders the overlay.
func (e *EditOverlay) View() string {
	if e.width == 0 {
		return "Loading..."
	}

	// Loading state
	if e.loading {
		return lipgloss.NewStyle().
			Width(e.width).
			Height(e.height).
			Render("Loading...")
	}

	// Error state
	if len(e.errors) > 0 && !e.showErrorPopup {
		return e.renderErrorView()
	}

	// Success message
	if e.successMsg != "" {
		// Show briefly then clear
	}

	var content strings.Builder

	// Header
	content.WriteString(e.renderHeader())
	content.WriteString("\n\n")

	// Fields
	if e.mode == modeEdit {
		content.WriteString(e.renderEditFields())
	} else {
		content.WriteString(e.renderDetailFields())
	}

	// Footer
	content.WriteString("\n")
	content.WriteString(e.renderFooter())

	// Modal overlays
	if e.showErrorPopup && e.errorPopup != nil {
		return lipgloss.Place(
			e.width,
			e.height,
			lipgloss.Center,
			lipgloss.Center,
			e.errorPopup.View(),
		)
	}

	if e.confirmDialog != nil {
		return lipgloss.Place(
			e.width,
			e.height,
			lipgloss.Center,
			lipgloss.Center,
			e.confirmDialog.View(),
		)
	}

	return e.wrapInOverlay(content.String())
}

// renderHeader renders the header.
func (e *EditOverlay) renderHeader() string {
	title := fmt.Sprintf("Edit: %s (%s)", e.entityType, e.entityID)

	modeStr := "[View]"
	if e.mode == modeEdit {
		modeStr = "[Edit]"
	} else if e.mode == modeSaving {
		modeStr = "[Saving...]"
	}

	return headerStyle.Render(title) + " " + modeStyle.Render(modeStr)
}

// renderFooter renders the footer.
func (e *EditOverlay) renderFooter() string {
	switch e.mode {
	case modeView:
		return footerStyle.Render("e: edit  •  esc: close")
	case modeEdit:
		changeCount := 0
		if e.session != nil {
			changeCount = e.session.GetChangeCount()
		}
		changes := ""
		if changeCount > 0 {
			changes = fmt.Sprintf(" (%d changes)", changeCount)
		}
		return footerStyle.Render(fmt.Sprintf("esc: cancel%s  •  ctrl+s: save  •  ctrl+z: undo  •  ↑↓: field", changes))
	case modeSaving:
		return footerStyle.Render("Saving...")
	}
	return ""
}

// renderEditFields renders editable fields.
func (e *EditOverlay) renderEditFields() string {
	var b strings.Builder
	editableFields := e.getEditableFields()

	for i, field := range editableFields {
		isFocused := i == e.focusedField
		value := e.getFieldValue(e.session, field.Key)
		b.WriteString(e.renderEditField(field, value, isFocused))
		b.WriteString("\n")
	}

	return b.String()
}

// renderEditField renders a single editable field.
func (e *EditOverlay) renderEditField(field edit.FieldDef, value interface{}, focused bool) string {
	var b strings.Builder

	label := labelStyle.Render(field.Label)
	b.WriteString(label)
	b.WriteString("\n")

	switch field.Type {
	case edit.FieldTagArray:
		if input, ok := e.fieldInputs[field.Key]; ok {
			if focused {
				input.Focus()
			} else {
				input.Blur()
			}
			input.SetWidth(e.width - 20)
			b.WriteString(input.View())
		} else {
			b.WriteString(inputStyle.Render("(no value)"))
		}

	case edit.FieldText, edit.FieldURL:
		displayValue := e.inputValue
		if displayValue == "" {
			if v, ok := value.(string); ok {
				displayValue = v
			}
		}
		if focused {
			cursor := "_"
			if len(displayValue) > 0 {
				cursor = ""
			}
			b.WriteString(inputStyle.Render(displayValue + cursor))
		} else {
			b.WriteString(inputStyle.Render(displayValue))
		}

	default:
		b.WriteString(inputStyle.Render(fmt.Sprintf("%v", value)))
	}

	return b.String()
}

// renderDetailFields renders fields in detail view mode.
func (e *EditOverlay) renderDetailFields() string {
	var b strings.Builder

	for _, field := range e.fields {
		value := e.getFieldValue(e.session, field.Key)
		b.WriteString(e.renderField(field, value))
		b.WriteString("\n")
	}

	return b.String()
}

// renderField renders a single field.
func (e *EditOverlay) renderField(field edit.FieldDef, value interface{}) string {
	label := labelStyle.Render(field.Label)
	valueStr := e.formatValue(field.Type, value)
	return fmt.Sprintf("%s: %s", label, valueStr)
}

// formatValue formats a value for display.
func (e *EditOverlay) formatValue(fieldType edit.FieldType, value interface{}) string {
	switch fieldType {
	case edit.FieldBool:
		if b, ok := value.(bool); ok {
			if b {
				return successStyle.Render("✓")
			}
			return normalStyle.Render("✗")
		}
	case edit.FieldTagArray:
		if arr, ok := value.([]string); ok {
			if len(arr) == 0 {
				return normalStyle.Render("(none)")
			}
			return strings.Join(arr, ", ")
		}
	}
	if value == nil {
		return normalStyle.Render("(none)")
	}
	return normalStyle.Render(fmt.Sprintf("%v", value))
}

// renderErrorView renders the error state.
func (e *EditOverlay) renderErrorView() string {
	var b strings.Builder
	b.WriteString(errorHeaderStyle.Render(" ✗ Error"))
	b.WriteString("\n\n")
	for _, err := range e.errors {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  • %s", err)))
		b.WriteString("\n")
	}
	return e.wrapInOverlay(b.String())
}

// wrapInOverlay wraps content in an overlay border.
func (e *EditOverlay) wrapInOverlay(content string) string {
	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C58CB")).
		Padding(1, 2).
		Width(e.width - 4).
		Height(e.height - 2)

	return overlayStyle.Render(content)
}

// Styles
var (
	headerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7C58CB")).
		Bold(true).
		Padding(0, 1)

	modeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA"))

	footerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555"))

	labelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Bold(true)

	inputStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#555555")).
		Padding(0, 1)

	errorHeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5555")).
		Bold(true)

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5555"))

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#50FA7B"))

	normalStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#DDDDDD"))
)

// EntityID returns the entity ID being edited.
func (e *EditOverlay) EntityID() string {
	return e.entityID
}

// EntityType returns the entity type being edited.
func (e *EditOverlay) EntityType() string {
	return e.entityType
}

// Message types
type EditOverlayReady struct {
	Session    *edit.EditSession
	HistoryDir string
}

type EditOverlayError struct {
	Message string
}

type EditOverlaySaved struct {
	Entity map[string]interface{}
}
