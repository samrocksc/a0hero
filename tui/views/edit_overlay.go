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
	fieldInputs   map[string]string // current input values for text fields
	tagInputs     map[string][]string // current values for tag fields
	focusedField  int
	mode          editMode
	errors        []string
	successMsg    string
	loading       bool
	dirty         bool

	// Dimensions
	width  int
	height int

	// Callbacks
	onClose func() tea.Msg
	onSave  func(map[string]interface{}) tea.Msg
	historyDir string
}

type editMode int

const (
	modeView editMode = iota
	modeEdit
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
		e.height = height - 4 // reserve for tabs/info/help
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
	// Populate input values from session
	e.fieldInputs = make(map[string]string)
	e.tagInputs = make(map[string][]string)
	for _, field := range e.fields {
		val := e.getFieldValue(session, field.Key)
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

// handleKey handles key events.
func (e *EditOverlay) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch e.mode {
	case modeView:
		return e.handleViewKey(key)
	case modeEdit:
		return e.handleEditKey(key)
	case modeSaving:
		// No input while saving
		return e, nil
	}
	return e, nil
}

func (e *EditOverlay) handleViewKey(key string) (tea.Model, tea.Cmd) {
	switch key {
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

func (e *EditOverlay) handleEditKey(key string) (tea.Model, tea.Cmd) {
	editableFields := e.getEditableFields()
	
	switch key {
	case "esc":
		if e.dirty {
			e.mode = modeView
			e.dirty = false
		} else {
			e.mode = modeView
		}
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
		if e.focusedField < len(editableFields)-1 {
			e.focusedField++
		}
	case "shift+tab":
		if e.focusedField > 0 {
			e.focusedField--
		}
	}
	return e, nil
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

// submit saves the changes.
func (e *EditOverlay) submit() tea.Cmd {
	e.mode = modeSaving
	e.errors = nil
	
	// Build changes map from inputs
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
		return lipgloss.NewStyle().
			Width(e.width).
			Render("Loading...")
	}

	if len(e.errors) > 0 {
		return e.renderErrors()
	}

	var b strings.Builder

	// Header with mode indicator
	modeLabel := "VIEW"
	modeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	if e.mode == modeEdit {
		modeLabel = "EDIT"
		modeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true)
	}
	if e.mode == modeSaving {
		modeLabel = "SAVING..."
		modeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8800")).Bold(true)
	}

	header := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("  %s %s", e.entityType, e.entityID))
	b.WriteString(header)
	b.WriteString("  ")
	b.WriteString(modeStyle.Render(fmt.Sprintf("[%s]", modeLabel)))
	b.WriteString("\n\n")

	// Success message
	if e.successMsg != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("  ✓ " + e.successMsg))
		b.WriteString("\n\n")
		e.successMsg = "" // Clear after showing
	}

	// Fields
	for i, field := range e.fields {
		isFocused := i == e.focusedField
		b.WriteString(e.renderField(field, isFocused))
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	helpText := " esc: close  •  e: edit  •  ↑↓: navigate"
	if e.mode == modeEdit {
		helpText = " esc: cancel  •  ctrl+s: save  •  ctrl+z: undo  •  ↑↓: navigate"
	}
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(helpText))

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
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Width(24)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))
	focusedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#333355"))
	readOnlyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))

	label := labelStyle.Render(field.Label + ":")

	// Get current value
	var value string
	switch field.Type {
	case edit.FieldTagArray:
		if tags, ok := e.tagInputs[field.Key]; ok {
			value = strings.Join(tags, ", ")
		}
	case edit.FieldBool:
		if v, ok := e.fieldInputs[field.Key]; ok {
			value = v
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
		return fmt.Sprintf("%s %s", label, focusedStyle.Render("▎"+value))
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