package chicle

import "testing"

func TestArrowsAndJKMoveTheCursor(t *testing.T) {
	m := press(t, fixture("a", "b", "c"), "down")
	if got := m.CursorRow().Key; got != "b" {
		t.Fatalf("after down, cursor on %q, want b", got)
	}
	m = press(t, m, "j")
	if got := m.CursorRow().Key; got != "c" {
		t.Fatalf("after j, cursor on %q, want c", got)
	}
	m = press(t, m, "up", "k")
	if got := m.CursorRow().Key; got != "a" {
		t.Fatalf("after up+k, cursor on %q, want a", got)
	}
}

func TestCursorCyclesPastTheEnd(t *testing.T) {
	m := press(t, fixture("a", "b", "c"), "down", "down", "down")
	if got := m.CursorRow().Key; got != "a" {
		t.Fatalf("cursor on %q, want a — should wrap past the end", got)
	}
}

func TestCursorCyclesPastTheStart(t *testing.T) {
	m := press(t, fixture("a", "b", "c"), "up")
	if got := m.CursorRow().Key; got != "c" {
		t.Fatalf("cursor on %q, want c — should wrap past the start", got)
	}
}

func TestNoCycleClampsAtBothEnds(t *testing.T) {
	cfg := Config{Columns: []Column{{Title: "N"}}, Rows: rows("a", "b"), NoCycle: true}

	m := press(t, New(cfg), "up")
	if got := m.CursorRow().Key; got != "a" {
		t.Fatalf("cursor on %q, want a — NoCycle must stop at the top", got)
	}

	m = press(t, New(cfg), "down", "down", "down")
	if got := m.CursorRow().Key; got != "b" {
		t.Fatalf("cursor on %q, want b — NoCycle must stop at the bottom", got)
	}
}

func TestMovingAnEmptyListIsInert(t *testing.T) {
	m := New(Config{Columns: []Column{{Title: "N"}}})
	m = press(t, m, "down", "up", "j", "k") // must not panic or divide by zero
	if got := m.CursorRow().Key; got != "" {
		t.Fatalf("cursor on %q, want empty", got)
	}
}
