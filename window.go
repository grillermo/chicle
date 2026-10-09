package chicle

// chrome is the number of screen lines the list does not get: the title, the
// query line, the column header, the blank line, the action row, and the
// status line. It is
// deliberately generous — one row too few is invisible, one row too many
// scrolls the terminal and tears the frame.
func (m Model) chrome() int {
	n := 5
	if m.cfg.Title != "" {
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
	h := m.height - m.chrome() - m.headingLines()
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

// page moves the cursor and the window together by one screenful, dir +1 for
// down and -1 for up, so the rows keep their place on screen as in vim's
// ctrl+f/ctrl+b. It stops at either end rather than wrapping: a page key held
// down should come to rest on the first or last row, not cycle forever.
func (m Model) page(dir int) Model {
	n := len(m.visibleRows())
	if n == 0 {
		return m
	}
	h := m.visibleHeight()
	m.cursor = min(max(m.cursor+dir*h, 0), n-1)
	m.top = min(max(m.top+dir*h, 0), max(n-h, 0))
	return m.scrollToCursor()
}

// offscreen is how many rows fell off each end, for the "N more above" and
// "N more below" hints.
func (m Model) offscreen() (above, below int) {
	start, end := m.window()
	return start, len(m.visibleRows()) - end
}

// sections is the distinct Section values among the visible rows, in display
// order.
func (m Model) sections() []string {
	var out []string
	prev := ""
	for i, r := range m.visibleRows() {
		if i == 0 || r.Section != prev {
			out = append(out, r.Section)
			prev = r.Section
		}
	}
	return out
}

// headingLines is the screen height the section headings take: one line each,
// plus a blank separator before every heading but the first. A list with only
// one section draws no headings at all — a single heading tells you nothing
// you cannot see.
func (m Model) headingLines() int {
	secs := m.sections()
	if len(secs) < 2 {
		return 0
	}
	if len(secs) == 1 && secs[0] == "" {
		return 0
	}
	return len(secs)*2 - 1
}
