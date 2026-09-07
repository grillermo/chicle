package chicle

// move walks the cursor by delta. It wraps around both ends unless NoCycle is
// set: on a short list wrapping is how you get from the last row back to the
// first without holding a key down.
func (m Model) move(delta int) Model {
	n := len(m.visibleRows())
	if n == 0 {
		return m
	}
	next := m.cursor + delta
	if m.cfg.NoCycle {
		if next < 0 {
			next = 0
		}
		if next >= n {
			next = n - 1
		}
		m.cursor = next
		return m
	}
	// Go's % keeps the sign of the dividend, so a negative index needs the
	// extra +n before the second reduction.
	m.cursor = ((next % n) + n) % n
	return m
}

// clampCursor keeps the cursor inside the visible rows after the row set or
// the filter changed underneath it.
func (m Model) clampCursor() Model {
	n := len(m.visibleRows())
	if m.cursor >= n {
		m.cursor = n - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	return m
}
