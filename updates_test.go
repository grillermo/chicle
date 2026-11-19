package chicle

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// deliver hands the model an updated row set the way waitForUpdate would.
func deliver(t *testing.T, m Model, rs []Row) Model {
	t.Helper()
	next, _ := m.Update(rowsMsg{rows: rs})
	return next.(Model)
}

func withUpdates(ch <-chan []Row) Model {
	return New(Config{
		Columns: []Column{{Title: "N"}},
		Rows:    rows("a", "b", "c"),
		Updates: ch,
	})
}

func TestUpdatesReplaceTheRows(t *testing.T) {
	m := deliver(t, withUpdates(nil), rows("x", "y"))
	eq(t, keys(m), []string{"x", "y"})
}

func TestAnUpdateDoesNotMoveTheCursor(t *testing.T) {
	// The bug this guards: a slow column filling in must not yank the
	// highlight off the row the user had picked out.
	m := press(t, withUpdates(nil), "down") // cursor on "b"
	m = deliver(t, m, []Row{
		{Key: "a", Cols: []string{"a", "merged"}},
		{Key: "b", Cols: []string{"b", "merged"}},
		{Key: "c", Cols: []string{"c", "unmerged"}},
	})
	if got := m.CursorRow().Key; got != "b" {
		t.Fatalf("cursor on %q after an update, want b", got)
	}
}

func TestTheCursorFollowsItsRowWhenTheOrderChanges(t *testing.T) {
	m := press(t, withUpdates(nil), "down", "down") // cursor on "c"
	m = deliver(t, m, rows("c", "a", "b"))
	if got := m.CursorRow().Key; got != "c" {
		t.Fatalf("cursor on %q, want c — it should follow the row, not the index", got)
	}
}

func TestTheCursorFallsBackWhenItsRowDisappears(t *testing.T) {
	m := press(t, withUpdates(nil), "down", "down") // cursor on "c"
	m = deliver(t, m, rows("a", "b"))
	if got := m.CursorRow().Key; got != "b" {
		t.Fatalf("cursor on %q, want b — the last row, since c is gone", got)
	}
}

func TestUpdatesKeepTheFilter(t *testing.T) {
	m := press(t, withUpdates(nil), "/", "a")
	m = deliver(t, m, rows("apple", "beta", "apex"))
	eq(t, keys(m), []string{"apple", "beta", "apex"}) // all contain "a"
	m = press(t, m, "p")
	eq(t, keys(m), []string{"apple", "apex"})
}

func TestUpdatesPreserveTicks(t *testing.T) {
	m := New(Config{
		Columns:     []Column{{Title: "N"}},
		Rows:        rows("a", "b"),
		MultiSelect: true,
	})
	m = press(t, m, "space") // tick "a"
	m = deliver(t, m, rows("a", "b", "c"))
	eq(t, ticked(m), []string{"a"})
}

func TestInitWatchesTheUpdatesChannel(t *testing.T) {
	ch := make(chan []Row, 1)
	ch <- rows("x")
	m := withUpdates(ch)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init returned no command — nothing is watching Updates")
	}
	msg := cmd()
	got, ok := msg.(rowsMsg)
	if !ok {
		t.Fatalf("got %T, want rowsMsg", msg)
	}
	if got.closed {
		t.Fatal("channel reported closed while it had a value")
	}
	eq(t, []string{got.rows[0].Key}, []string{"x"})
}

func TestAClosedUpdatesChannelStopsTheWatch(t *testing.T) {
	ch := make(chan []Row)
	close(ch)
	m := withUpdates(ch)
	msg := m.Init()().(rowsMsg)
	if !msg.closed {
		t.Fatal("a closed channel did not report closed")
	}
	next, cmd := m.Update(msg)
	if cmd != nil {
		t.Fatal("still watching a closed channel — this would spin")
	}
	eq(t, keys(next.(Model)), []string{"a", "b", "c"}) // rows untouched
}

func TestNoUpdatesChannelMeansNoInitCommand(t *testing.T) {
	if cmd := fixture("a").Init(); cmd != nil {
		t.Fatal("Init returned a command with no Updates channel")
	}
}

var _ tea.Model = Model{}
