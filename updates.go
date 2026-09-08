package chicle

import tea "github.com/charmbracelet/bubbletea"

// rowsMsg carries a replacement row set from Config.Updates. closed says the
// channel is finished and should not be watched again.
type rowsMsg struct {
	rows   []Row
	closed bool
}

// waitForUpdate blocks on the Updates channel off the event loop, so keys stay
// responsive while a slow column is still being computed.
func waitForUpdate(ch <-chan []Row) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		rows, open := <-ch
		if !open {
			return rowsMsg{closed: true}
		}
		return rowsMsg{rows: rows}
	}
}

// applyUpdate swaps in a new row set without disturbing the user. The cursor
// re-anchors on the Key it was sitting on: rows arriving is not a user action,
// and a highlight that jumps because a column filled itself in is maddening.
func (m Model) applyUpdate(rs []Row) Model {
	was := m.CursorRow().Key
	ticks := map[string]bool{}
	for _, r := range m.rows {
		if r.Ticked {
			ticks[r.Key] = true
		}
	}

	next := append([]Row(nil), rs...)
	for i := range next {
		if ticks[next[i].Key] {
			next[i].Ticked = true
		}
	}
	m.rows = next
	m = m.normaliseLocks()

	// Re-find the row the cursor was on. If it is gone, clampCursor leaves the
	// cursor as close to where it was as the new list allows.
	for i, r := range m.visibleRows() {
		if r.Key == was {
			m.cursor = i
			return m.scrollToCursor()
		}
	}
	return m.clampCursor().scrollToCursor()
}
