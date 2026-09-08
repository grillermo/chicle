package chicle

import "testing"

// table is a two-column fixture, so we can prove matching spans all cells.
func table() Model {
	return New(Config{
		Columns: []Column{{Title: "NAME", Width: 10}, {Title: "HOST"}},
		Rows: []Row{
			{Key: "1", Cols: []string{"web", "prod-eu"}},
			{Key: "2", Cols: []string{"web", "staging"}},
			{Key: "3", Cols: []string{"db", "prod-us"}},
		},
	})
}

func TestNoQueryShowsEveryRow(t *testing.T) {
	eq(t, keys(table()), []string{"1", "2", "3"})
}

func TestFilterMatchesASingleTerm(t *testing.T) {
	m := table()
	m.query = []rune("web")
	eq(t, keys(m), []string{"1", "2"})
}

func TestFilterRequiresEveryTerm(t *testing.T) {
	m := table()
	m.query = []rune("web prod")
	eq(t, keys(m), []string{"1"})
}

func TestFilterSpansAllColumns(t *testing.T) {
	m := table()
	m.query = []rune("prod")
	eq(t, keys(m), []string{"1", "3"})
}

func TestFilterIsCaseInsensitive(t *testing.T) {
	m := table()
	m.query = []rune("WEB Prod")
	eq(t, keys(m), []string{"1"})
}

func TestFilterMatchingNothingShowsNothing(t *testing.T) {
	m := table()
	m.query = []rune("nonesuch")
	if got := len(m.visibleRows()); got != 0 {
		t.Fatalf("%d rows visible, want 0", got)
	}
}

func TestWhitespaceOnlyQueryShowsEveryRow(t *testing.T) {
	m := table()
	m.query = []rune("   ")
	eq(t, keys(m), []string{"1", "2", "3"})
}

func TestSlashEntersFilterModeAndTypingNarrows(t *testing.T) {
	m := press(t, table(), "/", "web")
	if !m.filtering {
		t.Fatal("not in filter mode after /")
	}
	eq(t, keys(m), []string{"1", "2"})
}

func TestBackspaceWidensAgain(t *testing.T) {
	m := press(t, table(), "/", "web", "backspace")
	if string(m.query) != "we" {
		t.Fatalf("query %q, want %q", string(m.query), "we")
	}
	eq(t, keys(m), []string{"1", "2"})
}

func TestCtrlWDeletesThePreviousWord(t *testing.T) {
	m := press(t, table(), "/", "web prod", "ctrl+w")
	if string(m.query) != "web " {
		t.Fatalf("query %q, want %q", string(m.query), "web ")
	}
}

func TestCtrlWEatsTrailingSpacesThenTheWord(t *testing.T) {
	m := press(t, table(), "/", "web prod   ", "ctrl+w")
	if string(m.query) != "web " {
		t.Fatalf("query %q, want %q", string(m.query), "web ")
	}
}

func TestCtrlUWipesTheWholeQuery(t *testing.T) {
	m := press(t, table(), "/", "web prod", "ctrl+u")
	if string(m.query) != "" {
		t.Fatalf("query %q, want empty", string(m.query))
	}
	eq(t, keys(m), []string{"1", "2", "3"})
}

func TestCtrlAGoesToTheStartAndCtrlEToTheEnd(t *testing.T) {
	// readline order: ctrl+a is the start of the line, ctrl+e the end.
	m := press(t, table(), "/", "web", "ctrl+a")
	if m.qpos != 0 {
		t.Fatalf("qpos %d after ctrl+a, want 0", m.qpos)
	}
	m = press(t, m, "ctrl+e")
	if m.qpos != 3 {
		t.Fatalf("qpos %d after ctrl+e, want 3", m.qpos)
	}
}

func TestTypingInsertsAtTheQueryCursor(t *testing.T) {
	m := press(t, table(), "/", "wb", "ctrl+a", "right", "e")
	if string(m.query) != "web" {
		t.Fatalf("query %q, want %q", string(m.query), "web")
	}
}

func TestArrowsMoveTheQueryCursorNotTheList(t *testing.T) {
	m := press(t, table(), "/", "e", "left")
	if m.qpos != 0 {
		t.Fatalf("qpos %d, want 0", m.qpos)
	}
	if m.cursor != 0 {
		t.Fatalf("cursor %d — left/right must not move the list while filtering", m.cursor)
	}
}

func TestUpDownStillMoveTheListWhileFiltering(t *testing.T) {
	m := press(t, table(), "/", "web", "down")
	if got := m.CursorRow().Key; got != "2" {
		t.Fatalf("cursor on %q, want 2", got)
	}
}

func TestEditingTheQueryResetsTheCursorToTheTop(t *testing.T) {
	m := press(t, table(), "/", "web", "down")
	m = press(t, m, "backspace")
	if m.cursor != 0 {
		t.Fatalf("cursor %d after editing the query, want 0", m.cursor)
	}
}

func TestEscLeavesFilterModeAndClearsTheQuery(t *testing.T) {
	m := press(t, table(), "/", "web", "esc")
	if m.filtering {
		t.Fatal("still in filter mode after esc")
	}
	if string(m.query) != "" {
		t.Fatalf("query %q, want empty", string(m.query))
	}
	eq(t, keys(m), []string{"1", "2", "3"})
}

func TestTabLeavesFilterModeButKeepsTheFilter(t *testing.T) {
	m := press(t, table(), "/", "web", "tab")
	if m.filtering {
		t.Fatal("still in filter mode after tab")
	}
	eq(t, keys(m), []string{"1", "2"})
}

func TestEscInTheListClearsTheFilterBeforeItQuits(t *testing.T) {
	// A filtered list must not exit on the first esc — that would throw away
	// the query and the session in one keystroke.
	m := press(t, table(), "/", "web", "tab", "esc")
	if m.quit {
		t.Fatal("quit on the first esc — should have cleared the filter instead")
	}
	eq(t, keys(m), []string{"1", "2", "3"})

	m = press(t, m, "esc")
	if !m.quit {
		t.Fatal("did not quit on the second esc")
	}
}

func TestQIsATextCharacterWhileFiltering(t *testing.T) {
	m := press(t, table(), "/", "q")
	if m.quit {
		t.Fatal("q quit the picker while filtering — it must type instead")
	}
	if string(m.query) != "q" {
		t.Fatalf("query %q, want %q", string(m.query), "q")
	}
}

func TestCtrlCQuitsEvenWhileFiltering(t *testing.T) {
	m := press(t, table(), "/", "web", "ctrl+c")
	if !m.quit {
		t.Fatal("ctrl+c did not quit")
	}
}
