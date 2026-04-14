// Package components provides shared TUI components.
package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Table renders aligned column data with selection highlighting.
// It auto-computes column widths from the headers and rows.
type Table struct {
	headers    []string
	rows       [][]string
	selected   int
	width      int // total available width
	showHeader bool
}

// NewTable creates a table with the given column headers.
func NewTable(headers []string) *Table {
	return &Table{
		headers:    headers,
		selected:   -1,
		showHeader: true,
	}
}

// WithRows sets the table data.
func (t *Table) WithRows(rows [][]string) *Table {
	t.rows = rows
	return t
}

// WithSelected sets the selected row index (-1 for no selection).
func (t *Table) WithSelected(idx int) *Table {
	t.selected = idx
	return t
}

// WithWidth sets the total available width.
func (t *Table) WithWidth(w int) *Table {
	t.width = w
	return t
}

// WithHeaderVisible controls whether the header row is shown.
func (t *Table) WithHeaderVisible(v bool) *Table {
	t.showHeader = v
	return t
}

// Styles
var (
	headerCellStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Bold(true)

	normalCellStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCCCCC"))

	selectedCellStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#7C58CB"))

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#333333"))

	// Per-column padding (space between columns)
	colGap = 2
)

// columnWidths computes the width of each column based on the widest
// value in that column (header or data).
func (t *Table) columnWidths() []int {
	n := len(t.headers)
	widths := make([]int, n)

	// Start with header widths
	for i, h := range t.headers {
		widths[i] = ansi.StringWidth(h)
	}

	// Expand for row data
	for _, row := range t.rows {
		for i, cell := range row {
			if i < n {
				w := ansi.StringWidth(cell)
				if w > widths[i] {
					widths[i] = w
				}
			}
		}
	}

	// If total width exceeds available width, truncate the widest columns
	// until everything fits. TotalGap = colGap * (n-1) for inter-column spacing.
	if t.width > 0 {
		totalGap := colGap * (n - 1)
		avail := t.width - 2 - totalGap // 2 for left margin

		// First check if we even need to shrink
		total := 0
		for _, w := range widths {
			total += w
		}

		if total > avail {
			// Shrink from the rightmost columns first, but never below a minimum
			minWidth := 4
			for total > avail {
				shrunk := false
				for i := n - 1; i >= 0; i-- {
					if widths[i] > minWidth {
						widths[i]--
						total--
						shrunk = true
						if total <= avail {
							break
						}
					}
				}
				if !shrunk {
					break // can't shrink anymore
				}
			}
		}
	}

	return widths
}

// padRight pads a string to exactly n visible-width characters.
func padRight(s string, n int) string {
	w := ansi.StringWidth(s)
	if w >= n {
		return s
	}
	return s + strings.Repeat(" ", n-w)
}

// truncateRight truncates a string to at most n visible-width characters.
func truncateRight(s string, n int) string {
	if ansi.StringWidth(s) <= n {
		return s
	}
	// Trim runes until we fit
	runes := []rune(s)
	for len(runes) > 0 && ansi.StringWidth(string(runes)) > n {
		runes = runes[:len(runes)-1]
	}
	return string(runes)
}

// Render produces the table as a string.
func (t *Table) Render() string {
	var b strings.Builder
	widths := t.columnWidths()
	n := len(t.headers)

	// Header row
	if t.showHeader {
		cells := make([]string, n)
		for i, h := range t.headers {
			cells[i] = headerCellStyle.Render(padRight(h, widths[i]))
		}
		b.WriteString("  ")
		b.WriteString(strings.Join(cells, strings.Repeat(" ", colGap)))
		b.WriteString("\n")

		// Divider
		dashes := 0
		for _, w := range widths {
			dashes += w
		}
		dashes += colGap * (n - 1) + 2
		b.WriteString(dividerStyle.Render(strings.Repeat("─", dashes)))
		b.WriteString("\n")
	}

	// Data rows
	for i, row := range t.rows {
		cells := make([]string, n)
		for j := 0; j < n; j++ {
			var val string
			if j < len(row) {
				val = row[j]
			}

			// Truncate if too wide, then pad to column width
			val = truncateRight(val, widths[j])
			val = padRight(val, widths[j])

			if i == t.selected {
				cells[j] = selectedCellStyle.Render(val)
			} else {
				cells[j] = normalCellStyle.Render(val)
			}
		}

		b.WriteString("  ")
		b.WriteString(strings.Join(cells, strings.Repeat(" ", colGap)))
		b.WriteString("\n")
	}

	return b.String()
}

// RowCount returns the number of data rows.
func (t *Table) RowCount() int {
	return len(t.rows)
}