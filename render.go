package chicle

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	headerStyle   = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// truncate cuts s to width display cells. It counts runes rather than bytes:
// slicing a multi-byte string mid-rune emits invalid UTF-8, which terminals
// draw as a replacement character.
func truncate(s string, width int) string {
	r := []rune(s)
	if width < 0 || len(r) <= width {
		return s
	}
	return string(r[:width])
}

// pad lays cells out in the configured columns, two spaces apart. A zero-width
// column takes whatever is left.
func (m Model) pad(cells []string) string {
	parts := make([]string, 0, len(m.cfg.Columns))
	for i, c := range m.cfg.Columns {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		if c.Width > 0 {
			cell = fmt.Sprintf("%-*s", c.Width, truncate(cell, c.Width))
		}
		parts = append(parts, cell)
	}
	return strings.Join(parts, "  ")
}

// clamp keeps a line inside the terminal. MaxWidth rather than a rune slice:
// it measures display width, so a name made of wide characters cannot overflow
// its column and wrap, and it is ANSI-aware so it will not eat a style reset.
func (m Model) clamp(s string) string {
	w := m.width
	if w <= 0 {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(w).Render(s)
}

func button(label string, focused bool) string {
	if focused {
		return selectedStyle.Render("[ " + label + " ]")
	}
	return "[ " + label + " ]"
}

func (m Model) queryLine() string {
	return dimStyle.Render("/ ") + string(m.query)
}

// hint is the key legend. It changes with the mode: there is no point offering
// [a]/[n] while runes are going into the query.
func (m Model) hint() string {
	if m.confirming {
		return "[←] yes   [→] no   [enter] answer   [esc] cancel"
	}
	if m.filtering {
		return "[esc] clear filter   [ctrl+w] delete word   [tab] back to list   [enter] go"
	}
	parts := []string{"[↑↓] move"}
	if m.cfg.MultiSelect {
		parts = append(parts, "[space] toggle", "[a] all", "[n] none")
	}
	parts = append(parts, "[/] filter", "[enter] go", "[q] quit")
	return strings.Join(parts, "   ")
}

func (m Model) View() string {
	var b strings.Builder

	if m.cfg.Title != "" {
		b.WriteString(titleStyle.Render(m.cfg.Title) + "\n")
	}
	if m.showQuery() {
		b.WriteString(m.clamp(m.queryLine()) + "\n")
	}

	switch {
	case len(m.rows) == 0:
		b.WriteString(dimStyle.Render("Nothing to show.") + "\n")
	case len(m.visibleRows()) == 0:
		b.WriteString(dimStyle.Render("No rows match the filter.") + "\n")
	default:
		b.WriteString(headerStyle.Render(m.clamp(m.pad(m.columnTitles()))) + "\n")
		b.WriteString(m.body())
	}

	b.WriteString("\n")
	if m.confirming {
		b.WriteString(m.clamp(fmt.Sprintf("%s  %s  %s",
			m.question, button("Yes", m.confirmYes), button("No", !m.confirmYes))) + "\n")
	} else if len(m.cfg.Actions) > 0 {
		var row []string
		for i, a := range m.cfg.Actions {
			row = append(row, button(a.Label, i == m.button))
		}
		b.WriteString(m.clamp(strings.Join(row, "  ")) + "\n")
	}
	if m.status != "" {
		b.WriteString(m.clamp(m.status) + "\n")
	}
	b.WriteString(dimStyle.Render(m.clamp(m.hint())))
	return b.String()
}

func (m Model) columnTitles() []string {
	out := make([]string, len(m.cfg.Columns))
	for i, c := range m.cfg.Columns {
		out[i] = c.Title
	}
	return out
}

// body draws the windowed rows, their section headings, and the scrolled-off
// counters.
func (m Model) body() string {
	var b strings.Builder
	vis := m.visibleRows()
	start, end := m.window()
	above, below := m.offscreen()
	withHeadings := m.headingLines() > 0

	if above > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d more above", above)) + "\n")
	}

	section := ""
	for i := start; i < end; i++ {
		r := vis[i]
		if withHeadings && (i == start || r.Section != section) {
			if i > start {
				b.WriteString("\n")
			}
			b.WriteString(dimStyle.Render("  "+r.Section) + "\n")
		}
		section = r.Section

		line := m.pad(r.Cols)
		if m.cfg.MultiSelect {
			box := "[ ]"
			if r.Ticked {
				box = "[x]"
			}
			line = box + " " + line
		}
		if i == m.cursor {
			line = selectedStyle.Render("> " + line)
		} else {
			line = "  " + line
		}
		b.WriteString(m.clamp(line) + "\n")
	}

	if below > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d more below", below)) + "\n")
	}
	return b.String()
}
