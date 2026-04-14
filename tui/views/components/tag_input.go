package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TagInputModel is a Bubble Tea model for tag input.
type TagInputModel struct {
	label     string
	tags      []string
	input     string
	focused   bool
	cursor    int
	selected  int // which tag is selected (-1 = none)
	width     int
	err       string
	viewportY int
}

// NewTagInputModel creates a new tag input model.
func NewTagInputModel(label string, tags []string) TagInputModel {
	return TagInputModel{
		label:    label,
		tags:     tags,
		selected: -1,
	}
}

// Value returns the current tags.
func (m TagInputModel) Value() []string {
	return m.tags
}

// SetValue sets the tags.
func (m *TagInputModel) SetValue(tags []string) {
	m.tags = tags
	m.selected = -1
}

// SetError sets an error message.
func (m *TagInputModel) SetError(err string) {
	m.err = err
}

// SetWidth sets the component width.
func (m *TagInputModel) SetWidth(w int) {
	m.width = w
}

// IsFocused returns whether the input is focused.
func (m TagInputModel) IsFocused() bool {
	return m.focused
}

// Focus focuses the component.
func (m *TagInputModel) Focus() {
	m.focused = true
}

// Blur blurs the component.
func (m *TagInputModel) Blur() {
	m.focused = false
	m.selected = -1
}

// Init initializes the model.
func (m TagInputModel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m TagInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}

		switch msg.String() {
		case "enter":
			newTag := strings.TrimSpace(m.input)
			if newTag != "" {
				m.tags = append(m.tags, newTag)
				m.input = ""
				m.selected = -1
				m.err = ""
			}

		case "backspace":
			if m.input == "" && len(m.tags) > 0 {
				if m.selected >= 0 {
					m.tags = append(m.tags[:m.selected], m.tags[m.selected+1:]...)
					if m.selected >= len(m.tags) {
						m.selected = -1
					}
				} else {
					m.tags = m.tags[:len(m.tags)-1]
				}
			}

		case "left":
			if m.selected == -1 && len(m.tags) > 0 {
				m.selected = len(m.tags) - 1
			} else if m.selected > 0 {
				m.selected--
			}

		case "right":
			if m.selected >= 0 && m.selected < len(m.tags)-1 {
				m.selected++
			} else {
				m.selected = -1
			}

		case "delete", "x":
			if m.selected >= 0 && m.selected < len(m.tags) {
				m.tags = append(m.tags[:m.selected], m.tags[m.selected+1:]...)
				if m.selected >= len(m.tags) {
					m.selected = -1
				}
			}

		default:
			// Type into input
			m.selected = -1
			m.input += msg.String()
		}
	}
	return m, nil
}

// View renders the tag input.
func (m TagInputModel) View() string {
	width := m.width
	if width < 30 {
		width = 50
	}

	border := lipgloss.NormalBorder()
	borderColor := lipgloss.Color("#555555")
	if m.focused {
		border = lipgloss.RoundedBorder()
		borderColor = lipgloss.Color("#7C58CB")
	}

	// Build tags row
	var tagStrs []string
	for i, tag := range m.tags {
		if i == m.selected {
			tagStrs = append(tagStrs, tagSelectedStyle.Render(fmt.Sprintf("[%s ×]", tag)))
		} else {
			tagStrs = append(tagStrs, tagStyle.Render(fmt.Sprintf("[%s ×]", tag)))
		}
	}

	tagsRow := lipgloss.JoinHorizontal(lipgloss.Top, tagStrs...)

	// Assemble
	var b strings.Builder
	b.WriteString(labelStyle.Render(m.label))
	b.WriteString("\n")

	boxStyle := lipgloss.NewStyle().
		Border(border).
		BorderForeground(borderColor).
		Width(width).
		Padding(0, 1)

	if m.selected >= 0 || len(m.tags) > 0 {
		b.WriteString(boxStyle.Render(tagsRow))
		b.WriteString("\n")
	}

	// Input field
	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Width(width - 2)

	if m.focused {
		cursor := " "
		if len(m.input) > 0 {
			cursor = "_"
		}
		b.WriteString(boxStyle.Render(inputStyle.Render(m.input + cursor)))
	} else {
		b.WriteString(boxStyle.Render(inputStyle.Render(m.input)))
	}

	// Help text
	if m.focused {
		b.WriteString(helpStyle.Render(" Enter: add  •  Backspace: remove last  •  ←→: select tag  •  Del: remove selected"))
	}

	// Error
	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(errStyle.Render(m.err))
	}

	return b.String()
}

// Styles
var (
	labelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Bold(true)

	tagStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#555555")).
		Padding(0, 1)

	tagSelectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7C58CB")).
		Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Padding(1, 0)

	errStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5555"))
)
