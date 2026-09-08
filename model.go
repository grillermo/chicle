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

	filtering bool   // typing goes to the query instead of the list
	query     []rune // the filter text, matched against every cell
	qpos      int    // cursor position within query

	button int    // focused action
	status string // message under the action row

	confirming bool   // the prompt replaces the action row
	confirmYes bool   // focus sits on No until the user moves it
	question   string // text of the pending prompt
}

// New builds a Model from cfg. Rows are copied, so the caller's slice can be
// reused.
func New(cfg Config) Model {
	m := Model{cfg: cfg}
	m.rows = append([]Row(nil), cfg.Rows...)
	return m.normaliseLocks()
}

func (m Model) Init() tea.Cmd { return waitForUpdate(m.cfg.Updates) }

// Result is the picker's answer: the string an Action returned, or "" if the
// user quit without choosing.
func (m Model) Result() string { return m.result }

// visibleRows is the rows the filter lets through, in display order.
func (m Model) visibleRows() []Row { return filtered(m.rows, m.query) }

// showQuery keeps the query line on screen while it holds a filter, so it is
// never a mystery why rows are missing.
func (m Model) showQuery() bool { return m.filtering || len(m.query) > 0 }

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
	case rowsMsg:
		if msg.closed {
			return m, nil // stop watching; re-issuing would spin on a closed channel
		}
		return m.applyUpdate(msg.rows), waitForUpdate(m.cfg.Updates)
	case tea.KeyMsg:
		s := msg.String()
		if m.confirming {
			next, done := m.updateConfirm(s)
			if done {
				return next, tea.Quit
			}
			return next, nil
		}
		if m.filtering {
			// Printable text goes to the query; everything else is an editing
			// key. Runes are checked first so "q" and "/" type normally here.
			if msg.Type == tea.KeyRunes {
				for _, r := range msg.Runes {
					m = m.insertRune(r)
				}
				return m, nil
			}
			if msg.Type == tea.KeySpace {
				// Space is the tick command even while filtering — a
				// narrowed-down list is exactly when you want to tick the one
				// row you searched for, not type a literal space.
				if m.cfg.MultiSelect {
					return m.toggleCursor(), nil
				}
				return m.insertRune(' '), nil
			}
			next, quit := m.updateFiltering(s)
			if quit {
				return next, tea.Quit
			}
			return next, nil
		}
		if i, ok := m.keyedAction(s); ok {
			m.button = i
			next, done := m.activate()
			if done {
				return next, tea.Quit
			}
			return next, nil
		}
		switch s {
		case "down", "j":
			return m.move(1).scrollToCursor(), nil
		case "up", "k":
			return m.move(-1).scrollToCursor(), nil
		case "/":
			m.filtering = true
			m.qpos = len(m.query)
			return m, nil
		case "esc":
			// A filter on screen is cleared first: quitting and losing the
			// query to the same keystroke is never what was meant.
			if len(m.query) > 0 {
				return m.clearFilter(), nil
			}
			m.quit = true
			return m, tea.Quit
		case "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		case "left":
			return m.moveButton(-1), nil
		case "right":
			return m.moveButton(1), nil
		case "enter":
			next, done := m.activate()
			if done {
				return next, tea.Quit
			}
			return next, nil
		case " ", "space", "x":
			if m.cfg.MultiSelect {
				return m.toggleCursor(), nil
			}
		case "a":
			if m.cfg.MultiSelect {
				return m.setAll(true), nil
			}
		case "n":
			if m.cfg.MultiSelect {
				return m.setAll(false), nil
			}
		}
	}
	return m, nil
}
