package chicle

// chrome is the number of screen lines the list does not get: the title, the
// column header, the blank line, the action row, and the status line. It is
// deliberately generous — one row too few is invisible, one row too many
// scrolls the terminal and tears the frame.
func (m Model) chrome() int {
	n := 4
	if m.cfg.Title != "" {
		n++
	}
	if m.showQuery() {
		n++
	}
	return n
}

// visibleHeight is how many rows fit on screen. A zero height means we have
// not been told the terminal size yet, in which case every row is drawn and
// the terminal does its own scrolling.
func (m Model) visibleHeight() int {
	if m.height <= 0 {
		return len(m.visibleRows())
	}
	h := m.height - m.chrome()
	if h < 1 {
		return 1
	}
	return h
}

// window is the half-open range of visible rows to draw.
func (m Model) window() (start, end int) {
	rows := m.visibleRows()
	h := m.visibleHeight()
	if h >= len(rows) {
		return 0, len(rows)
	}
	start = m.top
	end = start + h
	if end > len(rows) {
		end = len(rows)
		start = end - h
	}
	if start < 0 {
		start = 0
	}
	return start, end
}

// scrollToCursor slides the window just far enough to keep the cursor on
// screen. Called after anything that moves the cursor or changes the rows.
func (m Model) scrollToCursor() Model {
	h := m.visibleHeight()
	if m.cursor < m.top {
		m.top = m.cursor
	}
	if m.cursor >= m.top+h {
		m.top = m.cursor - h + 1
	}
	if m.top < 0 {
		m.top = 0
	}
	return m
}

// offscreen is how many rows fell off each end, for the "N more above" and
// "N more below" hints.
func (m Model) offscreen() (above, below int) {
	start, end := m.window()
	return start, len(m.visibleRows()) - end
}
