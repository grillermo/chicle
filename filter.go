package chicle

import (
	"strings"
	"unicode"
)

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
// the old position points at a row that may no longer be there. The button row
// is re-clamped too, since an Action.Show may read the query.
func (m Model) queryChanged() Model {
	m.cursor = 0
	m.top = 0
	return m.clampCursor().clampButton()
}

func (m Model) clearFilter() Model {
	m.query = nil
	m.qpos = 0
	return m.queryChanged()
}

// insertRunes types rs at the query cursor and re-filters once, so a paste
// costs one pass over the rows rather than one per rune. Control runes are
// dropped: a pasted newline or tab has no place in a single-line query.
func (m Model) insertRunes(rs []rune) Model {
	typed := make([]rune, 0, len(rs))
	for _, r := range rs {
		if !unicode.IsControl(r) {
			typed = append(typed, r)
		}
	}
	if len(typed) == 0 {
		return m
	}
	q := append(append([]rune(nil), m.query[:m.qpos]...), typed...)
	m.query = append(q, m.query[m.qpos:]...)
	m.qpos += len(typed)
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

// editQuery handles the keys that edit or move within the query. Anything
// else is ignored.
func (m Model) editQuery(k string) Model {
	switch k {
	case "left":
		if m.qpos > 0 {
			m.qpos--
		}
	case "right":
		if m.qpos < len(m.query) {
			m.qpos++
		}
	case "home":
		m.qpos = 0
	case "end":
		m.qpos = len(m.query)
	case "backspace":
		return m.deleteBack()
	case "ctrl+w":
		return m.deleteWord()
	case "ctrl+u":
		m.query = nil
		m.qpos = 0
		return m.queryChanged()
	}
	return m
}
