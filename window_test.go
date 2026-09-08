package chicle

import "testing"

// tallEnough is a terminal with room for every row in these fixtures.
const tallEnough = 40

func TestWindowShowsEverythingWhenItFits(t *testing.T) {
	m := resize(t, fixture("a", "b", "c"), 80, tallEnough)
	start, end := m.window()
	if start != 0 || end != 3 {
		t.Fatalf("window [%d,%d), want [0,3)", start, end)
	}
}

func TestWindowScrollsToFollowTheCursorDown(t *testing.T) {
	m := resize(t, fixture("a", "b", "c", "d", "e", "f", "g", "h"), 80, 11)
	before, _ := m.window()
	if before != 0 {
		t.Fatalf("window starts at %d, want 0", before)
	}
	// Walk to the last row; the window must have followed.
	for range 7 {
		m = press(t, m, "down")
	}
	start, end := m.window()
	if m.cursor < start || m.cursor >= end {
		t.Fatalf("cursor %d outside window [%d,%d)", m.cursor, start, end)
	}
}

func TestWindowScrollsBackUp(t *testing.T) {
	m := resize(t, fixture("a", "b", "c", "d", "e", "f", "g", "h"), 80, 11)
	for range 7 {
		m = press(t, m, "down")
	}
	for range 7 {
		m = press(t, m, "up")
	}
	start, _ := m.window()
	if start != 0 {
		t.Fatalf("window starts at %d after returning to the top, want 0", start)
	}
}

func TestScrolledOffRowsAreCounted(t *testing.T) {
	m := resize(t, fixture("a", "b", "c", "d", "e", "f", "g", "h"), 80, 11)
	for range 7 {
		m = press(t, m, "down")
	}
	above, below := m.offscreen()
	if above == 0 {
		t.Fatalf("no rows counted above, want some — cursor is at the bottom")
	}
	if below != 0 {
		t.Fatalf("%d rows counted below, want 0 — cursor is on the last row", below)
	}
}

func TestUnsizedModelStillWindowsEveryRow(t *testing.T) {
	// Before the first WindowSizeMsg the height is 0. Showing nothing would
	// look like a broken picker, so show everything.
	m := fixture("a", "b", "c")
	start, end := m.window()
	if start != 0 || end != 3 {
		t.Fatalf("window [%d,%d), want [0,3) when the size is unknown", start, end)
	}
}
