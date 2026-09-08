package chicle

// New must force locked rows on. A locked row that starts unticked would be
// unremovable *and* unselected, which is a state no caller wants.
func (m Model) normaliseLocks() Model {
	rs := append([]Row(nil), m.rows...)
	for i := range rs {
		if rs[i].Locked {
			rs[i].Ticked = true
		}
	}
	m.rows = rs
	return m
}

// rowIndex maps the cursor — a position among the *visible* rows — back to an
// index into m.rows, or -1 when nothing is visible. Ticking goes through here
// so that a filter can never tick the wrong row.
func (m Model) rowIndex() int {
	vis := m.visibleRows()
	if len(vis) == 0 || m.cursor >= len(vis) {
		return -1
	}
	key := vis[m.cursor].Key
	for i := range m.rows {
		if m.rows[i].Key == key {
			return i
		}
	}
	return -1
}

// toggleCursor flips the tick under the cursor. The backing array is copied
// first: Bubble Tea passes models by value, so mutating in place would reach
// back into models the caller may still be holding.
func (m Model) toggleCursor() Model {
	i := m.rowIndex()
	if i < 0 || m.rows[i].Locked {
		return m
	}
	rs := append([]Row(nil), m.rows...)
	rs[i].Ticked = !rs[i].Ticked
	m.rows = rs
	return m
}

// setAll ticks or unticks every row. Locked rows stay ticked either way.
func (m Model) setAll(on bool) Model {
	rs := append([]Row(nil), m.rows...)
	for i := range rs {
		rs[i].Ticked = on || rs[i].Locked
	}
	m.rows = rs
	return m
}

// Selection is what the caller acts on. Without MultiSelect there are no
// checkboxes, so the cursor row is the selection — which keeps Action.Run
// identical for both kinds of list.
func (m Model) Selection() Selection {
	cur := m.CursorRow()
	if !m.cfg.MultiSelect {
		sel := Selection{Cursor: cur}
		if cur.Key != "" {
			sel.Ticked = []Row{cur}
		}
		return sel
	}
	sel := Selection{Cursor: cur}
	for _, r := range m.rows {
		if r.Ticked {
			sel.Ticked = append(sel.Ticked, r)
		}
	}
	return sel
}
