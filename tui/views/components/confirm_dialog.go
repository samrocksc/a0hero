package components

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmDialog displays a confirmation prompt.
type ConfirmDialog struct {
	title    string
	message  string
	buttons  []string // e.g., ["Yes", "No", "Cancel"]
	selected int
	width    int
	height   int
}

// NewConfirmDialog creates a new confirmation dialog.
func NewConfirmDialog(title, message string, buttons ...string) *ConfirmDialog {
	if len(buttons) == 0 {
		buttons = []string{"Yes", "No"}
	}
	return &ConfirmDialog{
		title:    title,
		message:  message,
		buttons:  buttons,
		selected: 0,
	}
}

// SetWidth sets the dialog width.
func (c *ConfirmDialog) SetWidth(w int) {
	c.width = w
}

// View renders the dialog.
func (c *ConfirmDialog) View() string {
	if c.width == 0 {
		c.width = 40
	}

	// Title bar
	titleBar := confirmTitleStyle.Render(c.title)

	// Message
	message := lipgloss.NewStyle().
		Width(c.width - 4).
		Render(c.message)

	// Buttons
	var buttonStrs []string
	for i, btn := range c.buttons {
		style := confirmButtonStyle
		if i == c.selected {
			style = confirmButtonSelectedStyle
		}
		buttonStrs = append(buttonStrs, style.Render("["+btn+"]"))
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, buttonStrs...)

	content := lipgloss.Place(
		c.width,
		0,
		lipgloss.Center,
		lipgloss.Center,
		lipgloss.JoinVertical(
			lipgloss.Center,
			"",
			titleBar,
			"",
			message,
			"",
			buttons,
			"",
		),
	)

	return lipgloss.Place(
		c.width,
		c.height,
		lipgloss.Center,
		lipgloss.Center,
		confirmBorder.Render(content),
	)
}

// Update handles input events.
func (c *ConfirmDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h", "shift+tab":
			if c.selected > 0 {
				c.selected--
			}

		case "right", "l", "tab":
			if c.selected < len(c.buttons)-1 {
				c.selected++
			}

		case "enter", " ":
			// Return result
			return c, func() tea.Msg {
				return ConfirmResult{
					Button:    c.buttons[c.selected],
					ButtonIdx: c.selected,
				}
			}

		case "esc", "q":
			// Cancel/close (typically last button)
			return c, func() tea.Msg {
				return ConfirmResult{
					Button:    "Cancel",
					ButtonIdx: len(c.buttons) - 1,
					Cancelled: true,
				}
			}
		}
	}
	return c, nil
}

// Init initializes the dialog.
func (c *ConfirmDialog) Init() tea.Cmd {
	return nil
}

// ConfirmResult is sent when a button is pressed.
type ConfirmResult struct {
	Button    string
	ButtonIdx int
	Cancelled bool
}

// Styles
var (
	confirmBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C58CB")).
		Background(lipgloss.Color("#1a1a1a")).
		Padding(1)

	confirmTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7C58CB")).
		Bold(true)

	confirmButtonStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	confirmButtonSelectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7C58CB"))
)

// DirtyConfirmDialog is a helper for dirty state confirmation.
func DirtyConfirmDialog() *ConfirmDialog {
	return NewConfirmDialog(
		"Discard Changes?",
		"You have unsaved changes.\nAre you sure you want to discard them?",
		"Discard",
		"Keep Editing",
	)
}
