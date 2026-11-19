package chicle

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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

	filtering bool   // "/" was pressed: printable keys type into the query
	query     []rune // the filter text, matched against every cell
	qpos      int    // cursor position within query

	button int    // focused action, an index into cfg.Actions
	status string // message under the action row

	confirming bool   // the prompt replaces the action row
	confirmYes bool   // focus sits on No until the user moves it
	question   string // text of the pending prompt

	styles styles // rebound to the terminal by Run; see render.go
}

// New builds a Model from cfg. Rows are copied, so the caller's slice can be
// reused.
//
// Styles start on lipgloss's default renderer, which judges color support from
// os.Stdout. Run replaces them with a renderer bound to the terminal it draws
// to, so a picker whose stdout is captured still gets its highlights.
func New(cfg Config) Model {
	m := Model{cfg: cfg, styles: newStyles(lipgloss.DefaultRenderer())}
	m.rows = append([]Row(nil), cfg.Rows...)
	m = m.normaliseLocks()

	// A pre-filled Filter opens the list exactly as "/" and typing it would
	// leave it, cursor at the end: the caller is offering a guess, and the
	// first thing the user may want is to correct it.
	if cfg.Filter != "" {
		m.filtering = true
		m.query = []rune(cfg.Filter)
		m.qpos = len(m.query)
		m = m.queryChanged()
	}
	return m.clampButton()
}

func (m Model) Init() tea.Cmd { return waitForUpdate(m.cfg.Updates) }

// Result is the picker's answer: the string an Action returned, or "" if the
// user quit without choosing.
func (m Model) Result() string { return m.result }

// visibleRows is the rows the filter lets through, in display order.
func (m Model) visibleRows() []Row { return filtered(m.rows, m.query) }

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
		next, cmd := m.updateKey(msg)
		return next, cmd
	}
	return m, nil
}

// updateKey routes a keystroke: the confirm prompt has the keyboard while it
// is up, then the query while filtering, then the list.
func (m Model) updateKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	// A fast typist or a paste can deliver several runes in one message. On
	// the list each one is its own command, and once one of them is "/" the
	// rest is the start of the query.
	if !m.confirming && !m.filtering && msg.Type == tea.KeyRunes && !msg.Alt && len(msg.Runes) > 1 {
		for i, r := range msg.Runes {
			if m.filtering {
				return m.insertRunes(msg.Runes[i:]), nil
			}
			var cmd tea.Cmd
			if m, cmd = m.updateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}); cmd != nil || m.confirming {
				return m, cmd // quit, or a prompt that the rest must not answer
			}
		}
		return m, nil
	}
	s := msg.String()
	if m.confirming {
		return quitIf(m.updateConfirm(s))
	}
	if m.filtering {
		// Printable text, space included, goes to the query: terms are
		// separated by spaces, so a space is part of what is being typed.
		// Alt+rune is not text: it falls through to the hotkeys below.
		if msg.Type == tea.KeyRunes && !msg.Alt {
			return m.insertRunes(msg.Runes), nil
		}
		if msg.Type == tea.KeySpace {
			return m.insertRunes([]rune{' '}), nil
		}
	}
	// Hotkeys fire while filtering too: an action revealed by the filter would
	// otherwise be unreachable, and it exists to use the query. Plain runes
	// were taken above while filtering, so only named and alt keys match there.
	if i, ok := m.keyedAction(s); ok {
		m.button = i
		return quitIf(m.activate())
	}
	switch s {
	case "ctrl+c":
		m.quit = true
		return m, tea.Quit
	case "tab":
		return m.moveButton(1), nil
	case "shift+tab":
		return m.moveButton(-1), nil
	case "enter":
		return quitIf(m.activate())
	case "up":
		return m.move(-1).scrollToCursor(), nil
	case "down":
		return m.move(1).scrollToCursor(), nil
	case "ctrl+e", "ctrl+f", "pgdown":
		return m.page(1), nil
	case "ctrl+b", "pgup":
		return m.page(-1), nil
	case "ctrl+a":
		if m.cfg.MultiSelect {
			return m.setAll(true), nil
		}
		return m, nil
	case "ctrl+x":
		if m.cfg.MultiSelect {
			return m.setAll(false), nil
		}
		return m, nil
	}
	if m.filtering {
		if s == "esc" {
			// Leave the query, keep the filter: the narrowed list is what
			// the user searched for, and now wants to tick or act on.
			m.filtering = false
			return m, nil
		}
		return m.editQuery(s), nil
	}
	switch s {
	case "/":
		m.filtering = true
		m.qpos = len(m.query)
	case "j":
		return m.move(1).scrollToCursor(), nil
	case "k":
		return m.move(-1).scrollToCursor(), nil
	case " ", "space":
		if m.cfg.MultiSelect {
			return m.toggleCursor(), nil
		}
	case "esc":
		// A filter on screen is cleared first: quitting and losing the
		// query to the same keystroke is never what was meant.
		if len(m.query) > 0 {
			return m.clearFilter(), nil
		}
		m.quit = true
		return m, tea.Quit
	case "q":
		m.quit = true
		return m, tea.Quit
	}
	return m, nil
}

// quitIf turns an action's "done" into Bubble Tea's quit command.
func quitIf(m Model, done bool) (Model, tea.Cmd) {
	if done {
		return m, tea.Quit
	}
	return m, nil
}
