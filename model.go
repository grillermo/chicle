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

	query []rune // the filter text, matched against every cell
	qpos  int    // cursor position within query

	button int    // focused action, or onList when focus is on the rows
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

	// A pre-filled Filter opens the list exactly as typing it would leave it,
	// cursor at the end: the caller is offering a guess, and the first thing
	// the user may want is to correct it.
	if cfg.Filter != "" {
		m.query = []rune(cfg.Filter)
		m.qpos = len(m.query)
		m = m.queryChanged()
	}
	// Enter ticks under MultiSelect, so focus starts on the rows; a button must
	// be tabbed to before Enter runs it.
	if cfg.MultiSelect {
		m.button = onList
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
		s := msg.String()
		if m.confirming {
			next, done := m.updateConfirm(s)
			if done {
				return next, tea.Quit
			}
			return next, nil
		}
		// Printable text goes to the query, so a bare rune is never a command.
		// Alt+rune is not text: it falls through to the hotkeys, where it can
		// match an Action.Key and is otherwise ignored.
		if msg.Type == tea.KeyRunes && !msg.Alt {
			return m.insertRunes(msg.Runes), nil
		}
		if msg.Type == tea.KeySpace {
			return m.insertRunes([]rune{' '}), nil
		}
		// Hotkeys fire with a query set: an action revealed by the filter
		// would otherwise be unreachable, and it exists to use the query.
		// Plain runes were dealt with above, so only named and alt keys can match.
		if i, ok := m.keyedAction(s); ok {
			m.button = i
			next, done := m.activate()
			if done {
				return next, tea.Quit
			}
			return next, nil
		}
		switch s {
		case "ctrl+c":
			m.quit = true
			return m, tea.Quit
		case "esc":
			// A filter on screen is cleared first: quitting and losing the
			// query to the same keystroke is never what was meant.
			if len(m.query) > 0 {
				return m.clearFilter(), nil
			}
			m.quit = true
			return m, tea.Quit
		case "tab":
			return m.moveButton(1), nil
		case "shift+tab":
			return m.moveButton(-1), nil
		case "enter":
			if m.button == onList {
				return m.toggleCursor(), nil
			}
			next, done := m.activate()
			if done {
				return next, tea.Quit
			}
			return next, nil
		case "ctrl+a":
			if m.cfg.MultiSelect {
				return m.setAll(true), nil
			}
		case "ctrl+x":
			if m.cfg.MultiSelect {
				return m.setAll(false), nil
			}
		case "up":
			return m.move(-1).scrollToCursor(), nil
		case "down":
			return m.move(1).scrollToCursor(), nil
		case "ctrl+e", "ctrl+f", "pgdown":
			return m.page(1), nil
		case "ctrl+b", "pgup":
			return m.page(-1), nil
		default:
			return m.editQuery(s), nil
		}
	}
	return m, nil
}
