package chicle

import (
	"strings"
	"testing"
)

func multi(rs ...Row) Model {
	return New(Config{
		Columns:     []Column{{Title: "NAME"}},
		Rows:        rs,
		MultiSelect: true,
	})
}

func ticked(m Model) []string {
	var out []string
	for _, r := range m.Selection().Ticked {
		out = append(out, r.Key)
	}
	return out
}

func TestSpaceTicksTheCursorRow(t *testing.T) {
	m := press(t, multi(rows("a", "b", "c")...), "space")
	eq(t, ticked(m), []string{"a"})
}

func TestSpaceUnticksAgain(t *testing.T) {
	m := press(t, multi(rows("a", "b")...), "space", "space")
	if got := len(ticked(m)); got != 0 {
		t.Fatalf("%d rows ticked, want 0", got)
	}
}

func TestTicksAccumulateAcrossRows(t *testing.T) {
	m := press(t, multi(rows("a", "b", "c")...), "space", "down", "down", "space")
	eq(t, ticked(m), []string{"a", "c"})
}

func TestSelectAllAndNone(t *testing.T) {
	m := press(t, multi(rows("a", "b", "c")...), "ctrl+a")
	eq(t, ticked(m), []string{"a", "b", "c"})
	m = press(t, m, "ctrl+x")
	if got := len(ticked(m)); got != 0 {
		t.Fatalf("%d rows ticked after none, want 0", got)
	}
}

func TestLockedRowsStartTickedAndCannotBeUnticked(t *testing.T) {
	m := multi(Row{Key: "a", Cols: []string{"a"}, Locked: true})
	eq(t, ticked(m), []string{"a"})
	m = press(t, m, "space")
	eq(t, ticked(m), []string{"a"}) // still ticked
}

func TestNoneKeepsLockedRowsTicked(t *testing.T) {
	m := multi(
		Row{Key: "a", Cols: []string{"a"}, Locked: true},
		Row{Key: "b", Cols: []string{"b"}},
	)
	m = press(t, m, "ctrl+a", "ctrl+x")
	eq(t, ticked(m), []string{"a"})
}

func TestTickAllKeysDoNotEditTheQuery(t *testing.T) {
	m := press(t, multi(rows("alpha", "beta")...), "al", "ctrl+a")
	if string(m.query) != "al" || m.qpos != 2 {
		t.Fatalf("query %q qpos %d, want al/2 — ctrl+a is no longer home", string(m.query), m.qpos)
	}
	eq(t, ticked(m), []string{"alpha", "beta"})
	m = press(t, m, "ctrl+x")
	if string(m.query) != "al" {
		t.Fatalf("query %q, want al", string(m.query))
	}
	if got := len(ticked(m)); got != 0 {
		t.Fatalf("%d rows ticked after ctrl+x, want 0", got)
	}
}

func TestTogglingWhileFilteredTicksTheRowYouSee(t *testing.T) {
	m := multi(rows("alpha", "beta", "gamma")...)
	m = press(t, m, "gam", "space")
	eq(t, ticked(m), []string{"gamma"})
}

func TestFilteringNeverChangesWhatIsTicked(t *testing.T) {
	// The bug this guards: narrowing the view must not drop ticks for rows the
	// filter is hiding, or a filter silently unselects things.
	m := multi(rows("alpha", "beta", "gamma")...)
	m = press(t, m, "space") // tick alpha
	m = press(t, m, "gam")   // hide it
	eq(t, ticked(m), []string{"alpha"})
	m = press(t, m, "space") // also tick gamma
	m = press(t, m, "esc")   // unfilter
	eq(t, ticked(m), []string{"alpha", "gamma"})
}

func TestSingleSelectHasNoTickKeys(t *testing.T) {
	// Without MultiSelect, ctrl+a and ctrl+x tick nothing, and space types a
	// space instead.
	m := press(t, fixture("alpha", "beta"), "ctrl+a", "ctrl+x", "space")
	if got := len(ticked(m)); got != 1 {
		t.Fatalf("%d rows in Ticked, want 1 (the cursor row)", got)
	}
	if string(m.query) != " " {
		t.Fatalf("query %q, want a single space", string(m.query))
	}
}

func TestSingleSelectSelectionIsJustTheCursorRow(t *testing.T) {
	m := press(t, fixture("alpha", "beta"), "down")
	sel := m.Selection()
	if sel.Cursor.Key != "beta" {
		t.Fatalf("cursor %q, want beta", sel.Cursor.Key)
	}
	if len(sel.Ticked) != 1 || sel.Ticked[0].Key != "beta" {
		t.Fatalf("ticked %v, want [beta]", ticked(m))
	}
}

func TestTickedIsInRowOrderNotClickOrder(t *testing.T) {
	// Callers write this straight to config files, so it must be stable.
	m := multi(rows("a", "b", "c")...)
	m = press(t, m, "down", "down", "space", "up", "up", "space")
	if got := strings.Join(ticked(m), ","); got != "a,c" {
		t.Fatalf("ticked %q, want %q", got, "a,c")
	}
}

func TestSelectionOnAnEmptyListIsEmpty(t *testing.T) {
	m := New(Config{Columns: []Column{{Title: "N"}}, MultiSelect: true})
	sel := m.Selection()
	if sel.Cursor.Key != "" || len(sel.Ticked) != 0 {
		t.Fatalf("got %+v, want an empty selection", sel)
	}
}
