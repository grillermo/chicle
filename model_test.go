package chicle

import "testing"

func TestNewStartsOnTheFirstRow(t *testing.T) {
	m := fixture("alpha", "beta", "gamma")
	if got := m.CursorRow().Key; got != "alpha" {
		t.Fatalf("cursor on %q, want alpha", got)
	}
}

func TestVisibleRowsStartsAsEveryRow(t *testing.T) {
	m := fixture("alpha", "beta", "gamma")
	eq(t, keys(m), []string{"alpha", "beta", "gamma"})
}

func TestEmptyListHasNoCursorRow(t *testing.T) {
	m := New(Config{Columns: []Column{{Title: "NAME"}}})
	if got := m.CursorRow().Key; got != "" {
		t.Fatalf("cursor on %q, want empty", got)
	}
}
