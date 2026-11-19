package chicle

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// styles carries the lipgloss styles for one Model. They hang off the Model
// rather than the package because a lipgloss style is bound to the renderer
// that made it, and the default renderer decides how much color it may emit by
// looking at os.Stdout — while chicle draws to /dev/tty. Under `sel=$(picker)`,
// the case that /dev/tty output exists to serve, stdout is a pipe: the default
// renderer sees no terminal, reports no color support, and every style renders
// as plain text. Run rebinds these to the tty so the highlight survives being
// captured.
type styles struct {
	title    lipgloss.Style
	header   lipgloss.Style
	selected lipgloss.Style
	dim      lipgloss.Style
	plain    lipgloss.Style // unstyled, for width-clamping only
}

func newStyles(r *lipgloss.Renderer) styles {
	return styles{
		title:    r.NewStyle().Bold(true),
		header:   r.NewStyle().Bold(true),
		selected: r.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
		dim:      r.NewStyle().Foreground(lipgloss.Color("241")),
		plain:    r.NewStyle(),
	}
}

// withRenderer returns a copy drawing with r. Run calls it with a renderer
// bound to the terminal, which is the only place the real color profile is
// visible.
func (m Model) withRenderer(r *lipgloss.Renderer) Model {
	m.styles = newStyles(r)
	return m
}

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
	return m.styles.plain.MaxWidth(w).Render(s)
}

func (m Model) drawButton(label string, focused bool) string {
	if focused {
		return m.styles.selected.Render("[ " + label + " ]")
	}
	return "[ " + label + " ]"
}

// queryCursor marks the insertion point in the query line.
const queryCursor = "▏"

// queryLine is always on screen, so it is never a mystery why rows are
// missing, and an empty one doubles as the hint that "/" filters. The cursor
// is drawn only while filtering: it marks where typing goes, and outside the
// query typing does not go there.
func (m Model) queryLine() string {
	prompt := m.styles.dim.Render("filter: ")
	if !m.filtering {
		if len(m.query) == 0 {
			return prompt + m.styles.dim.Render("press / to filter")
		}
		return prompt + string(m.query)
	}
	if len(m.query) == 0 {
		return prompt + queryCursor + m.styles.dim.Render("type to filter")
	}
	return prompt + string(m.query[:m.qpos]) + queryCursor + string(m.query[m.qpos:])
}

// keyHints advertises the shown actions that carry a hotkey. They matter
// because a hotkey is the only way to fire an action without tabbing to its
// button first.
func (m Model) keyHints() []string {
	var out []string
	for _, i := range m.shownActions() {
		if a := m.cfg.Actions[i]; a.Key != "" {
			out = append(out, "["+a.Key+"] "+strings.ToLower(a.Label))
		}
	}
	return out
}

// hint is the key legend.
func (m Model) hint() string {
	if m.confirming {
		return "[←] yes   [→] no   [enter] answer   [esc] cancel"
	}
	parts := []string{"[↑↓] move"}
	if above, below := m.offscreen(); above+below > 0 {
		// Only worth the space when there is somewhere to page to.
		parts = append(parts, "[^e ^b] page")
	}
	if !m.filtering {
		parts = append(parts, "[/] filter")
	}
	if m.cfg.MultiSelect {
		if !m.filtering {
			parts = append(parts, "[space] toggle") // space types while filtering
		}
		parts = append(parts, "[^a] all", "[^x] none")
	}
	parts = append(parts, m.keyHints()...)
	if len(m.shownActions()) > 1 {
		parts = append(parts, "[tab] next action")
	}
	switch {
	case m.filtering:
		parts = append(parts, "[esc] done filtering", "[^w] delete word", "[^u] clear")
	case len(m.query) > 0:
		parts = append(parts, "[esc] clear filter")
	default:
		parts = append(parts, "[esc] quit")
	}
	if len(m.cfg.Actions) == 0 || len(m.shownActions()) > 0 {
		parts = append(parts, "[enter] go")
	}
	return strings.Join(parts, "   ")
}

func (m Model) View() string {
	var b strings.Builder

	if m.cfg.Title != "" {
		b.WriteString(m.styles.title.Render(m.cfg.Title) + "\n")
	}
	b.WriteString(m.clamp(m.queryLine()) + "\n")

	switch {
	case len(m.rows) == 0:
		b.WriteString(m.styles.dim.Render("Nothing to show.") + "\n")
	case len(m.visibleRows()) == 0:
		b.WriteString(m.styles.dim.Render("No rows match the filter.") + "\n")
	default:
		b.WriteString(m.styles.header.Render(m.clamp(m.pad(m.columnTitles()))) + "\n")
		b.WriteString(m.body())
	}

	b.WriteString("\n")
	if m.confirming {
		b.WriteString(m.clamp(fmt.Sprintf("%s  %s  %s",
			m.question, m.drawButton("Yes", m.confirmYes), m.drawButton("No", !m.confirmYes))) + "\n")
	} else if shown := m.shownActions(); len(shown) > 0 {
		var row []string
		for _, i := range shown {
			row = append(row, m.drawButton(m.cfg.Actions[i].Label, i == m.button))
		}
		b.WriteString(m.clamp(strings.Join(row, "  ")) + "\n")
	}
	if m.status != "" {
		b.WriteString(m.clamp(m.status) + "\n")
	}
	b.WriteString(m.styles.dim.Render(m.clamp(m.hint())))
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
		b.WriteString(m.styles.dim.Render(fmt.Sprintf("  %d more above", above)) + "\n")
	}

	section := ""
	for i := start; i < end; i++ {
		r := vis[i]
		if withHeadings && (i == start || r.Section != section) {
			if i > start {
				b.WriteString("\n")
			}
			b.WriteString(m.styles.dim.Render("  "+r.Section) + "\n")
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
			line = m.styles.selected.Render("> " + line)
		} else {
			line = "  " + line
		}
		b.WriteString(m.clamp(line) + "\n")
	}

	if below > 0 {
		b.WriteString(m.styles.dim.Render(fmt.Sprintf("  %d more below", below)) + "\n")
	}
	return b.String()
}
