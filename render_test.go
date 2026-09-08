package chicle

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func view(t *testing.T, m Model) string {
	t.Helper()
	return m.View()
}

func TestViewShowsColumnTitles(t *testing.T) {
	v := view(t, resize(t, table(), 80, tallEnough))
	if !strings.Contains(v, "NAME") || !strings.Contains(v, "HOST") {
		t.Fatalf("column titles missing from:\n%s", v)
	}
}

func TestViewShowsEveryVisibleRow(t *testing.T) {
	v := view(t, resize(t, fixture("alpha", "beta"), 80, tallEnough))
	for _, want := range []string{"alpha", "beta"} {
		if !strings.Contains(v, want) {
			t.Fatalf("row %q missing from:\n%s", want, v)
		}
	}
}

func TestViewHidesFilteredOutRows(t *testing.T) {
	m := press(t, resize(t, fixture("alpha", "beta"), 80, tallEnough), "/", "alph")
	v := view(t, m)
	if strings.Contains(v, "beta") {
		t.Fatalf("filtered-out row still drawn:\n%s", v)
	}
}

func TestViewShowsTheQueryLine(t *testing.T) {
	m := press(t, resize(t, fixture("alpha"), 80, tallEnough), "/", "alp")
	if v := view(t, m); !strings.Contains(v, "alp") {
		t.Fatalf("query not drawn:\n%s", v)
	}
}

func TestTheQueryLineStaysAfterLeavingFilterMode(t *testing.T) {
	// Otherwise it is a mystery why rows are missing.
	m := press(t, resize(t, fixture("alpha", "beta"), 80, tallEnough), "/", "alp", "tab")
	if v := view(t, m); !strings.Contains(v, "alp") {
		t.Fatalf("query line vanished after tab:\n%s", v)
	}
}

func TestViewShowsActionLabels(t *testing.T) {
	m := resize(t, withActions(
		Action{Label: "Open", Run: (&acted{}).run(Outcome{})},
		Action{Label: "Delete", Run: (&acted{}).run(Outcome{})},
	), 80, tallEnough)
	v := view(t, m)
	if !strings.Contains(v, "Open") || !strings.Contains(v, "Delete") {
		t.Fatalf("action labels missing from:\n%s", v)
	}
}

func TestViewShowsTheConfirmQuestion(t *testing.T) {
	m := press(t, resize(t, confirmable(&acted{}), 80, tallEnough), "enter")
	v := view(t, m)
	if !strings.Contains(v, "Delete a?") {
		t.Fatalf("question missing from:\n%s", v)
	}
	if !strings.Contains(v, "Yes") || !strings.Contains(v, "No") {
		t.Fatalf("yes/no missing from:\n%s", v)
	}
}

func TestTheConfirmPromptReplacesTheActionRow(t *testing.T) {
	m := press(t, resize(t, confirmable(&acted{}), 80, tallEnough), "enter")
	if v := view(t, m); strings.Contains(v, "[ Delete ]") {
		t.Fatalf("action row still drawn under a prompt:\n%s", v)
	}
}

func TestViewShowsTheStatusLine(t *testing.T) {
	m := withActions(Action{Label: "Go", Run: (&acted{}).run(Outcome{Status: "did a thing"})})
	m = press(t, resize(t, m, 80, tallEnough), "enter")
	if v := view(t, m); !strings.Contains(v, "did a thing") {
		t.Fatalf("status missing from:\n%s", v)
	}
}

func TestViewShowsCheckboxesOnlyForMultiSelect(t *testing.T) {
	multiV := view(t, resize(t, multi(rows("a")...), 80, tallEnough))
	if !strings.Contains(multiV, "[ ]") {
		t.Fatalf("no checkbox in a multi-select list:\n%s", multiV)
	}
	singleV := view(t, resize(t, fixture("a"), 80, tallEnough))
	if strings.Contains(singleV, "[ ]") {
		t.Fatalf("checkbox drawn in a single-select list:\n%s", singleV)
	}
}

func TestATickedRowIsMarked(t *testing.T) {
	m := press(t, resize(t, multi(rows("a")...), 80, tallEnough), "space")
	if v := view(t, m); !strings.Contains(v, "[x]") {
		t.Fatalf("ticked row not marked:\n%s", v)
	}
}

func TestViewShowsSectionHeadings(t *testing.T) {
	v := view(t, resize(t, sectioned(), 80, tallEnough))
	if !strings.Contains(v, "NEW") || !strings.Contains(v, "OLD") {
		t.Fatalf("section headings missing from:\n%s", v)
	}
}

func TestViewShowsOffscreenCounts(t *testing.T) {
	m := resize(t, fixture("a", "b", "c", "d", "e", "f", "g", "h"), 80, 11)
	for range 7 {
		m = press(t, m, "down")
	}
	if v := view(t, m); !strings.Contains(v, "more above") {
		t.Fatalf("no scrolled-off hint:\n%s", v)
	}
}

func TestEmptyListSaysSoDistinctly(t *testing.T) {
	m := resize(t, New(Config{Columns: []Column{{Title: "N"}}}), 80, tallEnough)
	if v := view(t, m); !strings.Contains(v, "Nothing to show") {
		t.Fatalf("no empty-list message:\n%s", v)
	}
}

func TestAFilterMatchingNothingSaysSomethingElse(t *testing.T) {
	// "nothing here" and "nothing matches" are different problems with
	// different fixes, so they must not share a message.
	m := press(t, resize(t, fixture("a", "b"), 80, tallEnough), "/", "zzz")
	v := view(t, m)
	if !strings.Contains(v, "No rows match") {
		t.Fatalf("no no-match message:\n%s", v)
	}
	if strings.Contains(v, "Nothing to show") {
		t.Fatalf("empty-list message shown for a non-matching filter:\n%s", v)
	}
}

func TestTheHelpLegendChangesWithTheMode(t *testing.T) {
	base := view(t, resize(t, multi(rows("a")...), 80, tallEnough))
	if !strings.Contains(base, "filter") {
		t.Fatalf("no filter hint in the list legend:\n%s", base)
	}
	filtering := view(t, press(t, resize(t, multi(rows("a")...), 80, tallEnough), "/"))
	if !strings.Contains(filtering, "clear filter") {
		t.Fatalf("legend did not change for filter mode:\n%s", filtering)
	}
}

func TestWideCharactersDoNotBreakTheLayout(t *testing.T) {
	// Padding by rune count lets CJK overflow its column and wrap, which
	// shears every row below it. Every rendered line must fit the width.
	m := New(Config{
		Columns: []Column{{Title: "NAME", Width: 10}, {Title: "X"}},
		Rows: []Row{
			{Key: "1", Cols: []string{"日本語テキストです", "ok"}},
			{Key: "2", Cols: []string{"ascii", "ok"}},
		},
	})
	m = resize(t, m, 40, tallEnough)
	for _, line := range strings.Split(view(t, m), "\n") {
		if w := lipglossWidth(line); w > 40 {
			t.Fatalf("line is %d wide, want <=40: %q", w, line)
		}
	}
}

func TestATinyTerminalDoesNotPanic(t *testing.T) {
	m := resize(t, withActions(Action{Label: "Go", Run: (&acted{}).run(Outcome{})}), 4, 3)
	_ = view(t, m) // must not panic or produce a negative-width slice
}

func TestTruncateIsRuneSafe(t *testing.T) {
	// Slicing bytes mid-rune emits invalid UTF-8, which terminals draw as a
	// replacement character.
	if got := truncate("日本語です", 3); got != "日本語" {
		t.Fatalf("truncate gave %q, want %q", got, "日本語")
	}
}

func lipglossWidth(s string) int { return lipgloss.Width(s) }
