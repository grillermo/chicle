package chicle

import (
	"fmt"
	"strings"
	"testing"
)

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

// twenty is a list long enough to page through on an 11-line terminal.
func twenty(t *testing.T) Model {
	t.Helper()
	names := make([]string, 20)
	for i := range names {
		names[i] = string(rune('a' + i))
	}
	return resize(t, fixture(names...), 80, 11)
}

func TestCtrlEPagesDownAndCtrlBPagesBack(t *testing.T) {
	m := twenty(t)
	h := m.visibleHeight()

	m = press(t, m, "ctrl+e")
	if m.cursor != h {
		t.Fatalf("cursor after ctrl+e = %d, want %d (one page)", m.cursor, h)
	}
	if start, end := m.window(); m.cursor < start || m.cursor >= end {
		t.Fatalf("cursor %d outside window [%d,%d)", m.cursor, start, end)
	}

	m = press(t, m, "ctrl+b")
	if start, _ := m.window(); m.cursor != 0 || start != 0 {
		t.Fatalf("after ctrl+b cursor = %d, window start = %d; want both 0", m.cursor, start)
	}
}

func TestPagingStopsAtTheEndsInsteadOfWrapping(t *testing.T) {
	m := twenty(t)
	m = press(t, m, "ctrl+b")
	if m.cursor != 0 {
		t.Fatalf("ctrl+b at the top moved the cursor to %d", m.cursor)
	}
	m = press(t, m, "ctrl+e", "ctrl+e", "ctrl+e", "ctrl+e", "ctrl+e")
	if m.cursor != 19 {
		t.Fatalf("cursor after paging past the end = %d, want the last row", m.cursor)
	}
	if _, below := m.offscreen(); below != 0 {
		t.Fatalf("%d rows still below the last page", below)
	}
}

func TestPageKeysAliases(t *testing.T) {
	for _, k := range []string{"ctrl+f", "pgdown"} {
		if m := press(t, twenty(t), k); m.cursor == 0 {
			t.Errorf("%s did not page down", k)
		}
	}
	if m := press(t, twenty(t), "ctrl+e", "pgup"); m.cursor != 0 {
		t.Errorf("pgup did not page up")
	}
}

func TestCtrlEPagesEvenWithAQuery(t *testing.T) {
	names := make([]string, 20)
	for i := range names {
		names[i] = fmt.Sprintf("row%02d", i)
	}
	m := press(t, resize(t, fixture(names...), 80, 11), "r", "ctrl+e")
	if m.qpos != 1 || m.cursor == 0 {
		t.Fatalf("qpos = %d, cursor = %d; ctrl+e must page, not move the query cursor", m.qpos, m.cursor)
	}
}

func TestThePageHintShowsOnlyWhenTheListOverflows(t *testing.T) {
	if h := twenty(t).hint(); !strings.Contains(h, "page") {
		t.Errorf("hint %q does not mention paging on a list that overflows", h)
	}
	if h := resize(t, fixture("a", "b"), 80, tallEnough).hint(); strings.Contains(h, "page") {
		t.Errorf("hint %q offers paging on a list that fits", h)
	}
}
