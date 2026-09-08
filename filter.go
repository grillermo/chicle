package chicle

import "strings"

// matches reports whether row satisfies every term. Terms are ANDed, and each
// one may land in any cell: typing "web prod" narrows twice rather than
// searching for the literal phrase.
func matches(r Row, terms []string) bool {
	hay := strings.ToLower(strings.Join(r.Cols, " "))
	for _, t := range terms {
		if !strings.Contains(hay, t) {
			return false
		}
	}
	return true
}

// filtered applies the query to rows. An empty or whitespace-only query
// returns rows untouched, sharing the backing array — no copy on the hot path.
func filtered(all []Row, query []rune) []Row {
	terms := strings.Fields(strings.ToLower(string(query)))
	if len(terms) == 0 {
		return all
	}
	out := make([]Row, 0, len(all))
	for _, r := range all {
		if matches(r, terms) {
			out = append(out, r)
		}
	}
	return out
}

// queryChanged re-applies the filter and starts the cursor over: after an edit
// the old position points at a row that may no longer be there.
func (m Model) queryChanged() Model {
	m.cursor = 0
	m.top = 0
	return m.clampCursor()
}

func (m Model) clearFilter() Model {
	m.filtering = false
	m.query = nil
	m.qpos = 0
	return m.queryChanged()
}

// insertRune types r at the query cursor.
func (m Model) insertRune(r rune) Model {
	tail := append([]rune{r}, m.query[m.qpos:]...)
	m.query = append(append([]rune(nil), m.query[:m.qpos]...), tail...)
	m.qpos++
	return m.queryChanged()
}

// deleteBack removes the rune before the query cursor.
func (m Model) deleteBack() Model {
	if m.qpos == 0 {
		return m
	}
	m.query = append(append([]rune(nil), m.query[:m.qpos-1]...), m.query[m.qpos:]...)
	m.qpos--
	return m.queryChanged()
}

// deleteWord drops the run of spaces before the cursor and the word before
// that, the way ctrl-w does on a shell line.
func (m Model) deleteWord() Model {
	i := m.qpos
	for i > 0 && m.query[i-1] == ' ' {
		i--
	}
	for i > 0 && m.query[i-1] != ' ' {
		i--
	}
	if i == m.qpos {
		return m
	}
	m.query = append(append([]rune(nil), m.query[:i]...), m.query[m.qpos:]...)
	m.qpos = i
	return m.queryChanged()
}

// updateFiltering handles keys while the query line has the keyboard. The list
// still scrolls with up/down, because left/right belong to the query here.
func (m Model) updateFiltering(k string) (Model, bool) {
	switch k {
	case "ctrl+c":
		m.quit = true
		return m, true
	case "esc":
		return m.clearFilter(), false
	case "tab":
		m.filtering = false
		return m, false
	case "up":
		return m.move(-1).scrollToCursor(), false
	case "down":
		return m.move(1).scrollToCursor(), false
	case "left":
		if m.qpos > 0 {
			m.qpos--
		}
		return m, false
	case "right":
		if m.qpos < len(m.query) {
			m.qpos++
		}
		return m, false
	case "ctrl+a":
		m.qpos = 0
		return m, false
	case "ctrl+e":
		m.qpos = len(m.query)
		return m, false
	case "backspace":
		return m.deleteBack(), false
	case "ctrl+w":
		return m.deleteWord(), false
	case "ctrl+u":
		m.query = nil
		m.qpos = 0
		return m.queryChanged(), false
	case "enter":
		next, _ := m.activate()
		return next, next.quit
	}
	return m, false
}
