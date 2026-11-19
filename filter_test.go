package chicle

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

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

func TestTypingNarrowsTheListStraightAway(t *testing.T) {
	m := press(t, table(), "/", "web")
	if string(m.query) != "web" {
		t.Fatalf("query %q, want web", string(m.query))
	}
	eq(t, keys(m), []string{"1", "2"})
}

func TestTypingDoesNothingUntilSlash(t *testing.T) {
	m := press(t, table(), "w", "e", "b")
	if len(m.query) != 0 || m.filtering {
		t.Fatalf("query %q filtering %v, want typing ignored before /", string(m.query), m.filtering)
	}
}

func TestListKeysTypeWhileFiltering(t *testing.T) {
	// j, k, q and space are commands on the list, and plain text after /.
	m := press(t, table(), "/", "j", "k", "q", "space")
	if string(m.query) != "jkq " {
		t.Fatalf("query %q, want %q", string(m.query), "jkq ")
	}
	if m.cursor != 0 || m.quit {
		t.Fatalf("cursor %d, quit %v — list keys must not act while filtering", m.cursor, m.quit)
	}
}

func TestJAndKMoveAndQQuitsOnTheList(t *testing.T) {
	m := press(t, table(), "j")
	if m.cursor != 1 {
		t.Fatalf("cursor %d after j, want 1", m.cursor)
	}
	m = press(t, m, "k")
	if m.cursor != 0 {
		t.Fatalf("cursor %d after k, want 0", m.cursor)
	}
	if m = press(t, m, "q"); !m.quit {
		t.Fatal("q did not quit")
	}
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

func TestHomeGoesToTheStartAndEndToTheEnd(t *testing.T) {
	m := press(t, table(), "/", "web", "home")
	if m.qpos != 0 {
		t.Fatalf("qpos %d after home, want 0", m.qpos)
	}
	m = press(t, m, "end")
	if m.qpos != 3 {
		t.Fatalf("qpos %d after end, want 3", m.qpos)
	}
}

func TestTypingInsertsAtTheQueryCursor(t *testing.T) {
	m := press(t, table(), "/", "wb", "home", "right", "e")
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
		t.Fatalf("cursor %d — left/right belong to the query", m.cursor)
	}
}

func TestUpDownStillMoveTheList(t *testing.T) {
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

func TestEscLeavesFilteringThenClearsThenQuits(t *testing.T) {
	// Each esc takes one step back: out of the query (keeping the filter), then
	// the filter, then the picker. Losing the query and the session to the
	// same keystroke is never what was meant.
	m := press(t, table(), "/", "web", "esc")
	if m.quit || m.filtering {
		t.Fatalf("quit %v filtering %v after the first esc, want back on the list", m.quit, m.filtering)
	}
	eq(t, keys(m), []string{"1", "2"})

	m = press(t, m, "esc")
	if m.quit || len(m.query) != 0 || m.qpos != 0 {
		t.Fatalf("quit %v query %q qpos %d after the second esc, want the filter cleared", m.quit, string(m.query), m.qpos)
	}
	eq(t, keys(m), []string{"1", "2", "3"})

	if m = press(t, m, "esc"); !m.quit {
		t.Fatal("did not quit on the third esc")
	}
}

func TestSlashResumesEditingAtTheEnd(t *testing.T) {
	m := press(t, table(), "/", "web", "esc", "/", "space", "prod")
	if string(m.query) != "web prod" || !m.filtering {
		t.Fatalf("query %q filtering %v, want %q still filtering", string(m.query), m.filtering, "web prod")
	}
}

func TestCtrlCQuits(t *testing.T) {
	m := press(t, table(), "/", "web", "ctrl+c")
	if !m.quit {
		t.Fatal("ctrl+c did not quit")
	}
}

func TestMultibyteInputEditsByRune(t *testing.T) {
	for _, word := range []string{"café", "日本語"} {
		n := len([]rune(word))
		m := press(t, table(), "/", word)
		if string(m.query) != word || m.qpos != n {
			t.Fatalf("%q: query %q qpos %d, want qpos %d", word, string(m.query), m.qpos, n)
		}
		m = press(t, m, "left", "backspace")
		want := string([]rune(word)[:n-2]) + string([]rune(word)[n-1:])
		if string(m.query) != want || m.qpos != n-2 {
			t.Fatalf("%q: after left+backspace query %q qpos %d, want %q/%d", word, string(m.query), m.qpos, want, n-2)
		}
	}
}

func send(t *testing.T, m Model, msg tea.KeyMsg) Model {
	t.Helper()
	next, _ := m.Update(msg)
	return next.(Model)
}

func TestPastedTextInsertsAtOnceAndDropsControlRunes(t *testing.T) {
	m := press(t, table(), "/", "wb", "left")
	m = send(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e\n\tx\r")})
	if string(m.query) != "wexb" || m.qpos != 3 {
		t.Fatalf("query %q qpos %d, want wexb/3", string(m.query), m.qpos)
	}
}

func TestAltRunesAreNotInserted(t *testing.T) {
	m := press(t, table(), "/")
	m = send(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x"), Alt: true})
	if len(m.query) != 0 || m.quit {
		t.Fatalf("query %q quit %v, want alt+x ignored", string(m.query), m.quit)
	}
}
