package chicle

import tea "github.com/charmbracelet/bubbletea"

// Model is the list. It implements tea.Model and holds no terminal state, so
// tests drive it by calling Update directly.
type Model struct {
	cfg  Config
	rows []Row // the live row set; replaced by Reload and Updates

	cursor int // index into the visible rows, never into rows
	top    int // first visible row, for scrolling

	width, height int

	quit   bool
	result string
}

// New builds a Model from cfg. Rows are copied, so the caller's slice can be
// reused.
func New(cfg Config) Model {
	m := Model{cfg: cfg}
	m.rows = append([]Row(nil), cfg.Rows...)
	return m
}

func (m Model) Init() tea.Cmd { return nil }

// Result is the picker's answer: the string an Action returned, or "" if the
// user quit without choosing.
func (m Model) Result() string { return m.result }

// visibleRows is the rows the filter lets through, in display order.
func (m Model) visibleRows() []Row { return m.rows }

// showQuery reports whether the filter line takes up a screen row.
func (m Model) showQuery() bool { return false }

// CursorRow is the highlighted row, or a zero Row when nothing is visible.
func (m Model) CursorRow() Row {
	vis := m.visibleRows()
	if len(vis) == 0 || m.cursor >= len(vis) {
		return Row{}
	}
	return vis[m.cursor]
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.scrollToCursor(), nil
	case tea.KeyMsg:
		switch msg.String() {
		case "down", "j":
			return m.move(1).scrollToCursor(), nil
		case "up", "k":
			return m.move(-1).scrollToCursor(), nil
		case "q", "ctrl+c", "esc":
			m.quit = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string { return "" }
