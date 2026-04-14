package components

import (
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ErrorPopup displays validation or API errors.
type ErrorPopup struct {
	title   string
	errors  []string
	width   int
	height  int
	focused bool
	button  int // 0 = OK
}

// NewErrorPopup creates a new error popup.
func NewErrorPopup(title string, errors []string) *ErrorPopup {
	return &ErrorPopup{
		title:  title,
		errors: errors,
		button: 0,
	}
}

// SetWidth sets the popup width.
func (e *ErrorPopup) SetWidth(w int) {
	e.width = w
}

// View renders the error popup.
func (e *ErrorPopup) View() string {
	if e.width == 0 {
		e.width = 50
	}

	contentWidth := e.width - 4

	// Build error lines
	var errorLines []string
	for _, err := range e.errors {
		if len(err) > contentWidth {
			lines := wrapText(err, contentWidth)
			errorLines = append(errorLines, lines...)
		} else {
			errorLines = append(errorLines, err)
		}
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		errorLines...,
	)

	// OK button
	okBtn := okButtonStyle
	if e.focused {
		okBtn = okButtonFocusedStyle
	}

	// Assemble popup using a simpler approach
	header := errorHeaderStyle.Render(" ✗ "+e.title)
	headerWidth := lipgloss.Width(header)

	padding := contentWidth - headerWidth
	if padding < 0 {
		padding = 0
	}

	body := lipgloss.NewStyle().
		Width(contentWidth).
		Render(content)

	buttonPad := (contentWidth - 6) / 2
	buttonRow := lipgloss.NewStyle().
		Width(contentWidth).
		Render(strings.Repeat(" ", buttonPad) + okBtn.Render("[ OK ]"))

	lines := []string{
		errorBorder.Render(""),
		header + strings.Repeat(" ", padding) + errorBorder.Render(""),
		errorBorder.Render("") + " " + body + " " + errorBorder.Render(""),
		errorBorder.Render("") + strings.Repeat(" ", contentWidth+1) + errorBorder.Render(""),
		errorBorder.Render("") + " " + buttonRow + " " + errorBorder.Render(""),
		errorBorder.Render(""),
	}

	return lipgloss.Place(
		e.width,
		8+len(errorLines),
		lipgloss.Center,
		lipgloss.Center,
		strings.Join(lines, "\n"),
	)
}

// Update handles input events.
func (e *ErrorPopup) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			return e, func() tea.Msg {
				return ErrorPopupDismissed{}
			}
		}
	}
	return e, nil
}

// Init initializes the popup.
func (e *ErrorPopup) Init() tea.Cmd {
	return nil
}

// ErrorPopupDismissed is sent when the popup is dismissed.
type ErrorPopupDismissed struct{}

// Styles
var (
	errorBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF5555")).
		Background(lipgloss.Color("#1a1a1a"))

	errorHeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5555")).
		Background(lipgloss.Color("#1a1a1a")).
		Bold(true)

	okButtonStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Background(lipgloss.Color("#1a1a1a"))

	okButtonFocusedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7C58CB"))
)

// wrapText wraps text to fit within a width.
func wrapText(s string, width int) []string {
	words := strings.Fields(s)
	var lines []string
	var currentLine strings.Builder

	for _, word := range words {
		if currentLine.Len()+len(word)+1 > width {
			if currentLine.Len() > 0 {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
			}
		}
		if currentLine.Len() > 0 {
			currentLine.WriteString(" ")
		}
		currentLine.WriteString(word)
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}
