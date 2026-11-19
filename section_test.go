package chicle

import "testing"

func sectioned() Model {
	return New(Config{
		Columns: []Column{{Title: "NAME"}},
		Rows: []Row{
			{Key: "a", Cols: []string{"a"}, Section: "NEW"},
			{Key: "b", Cols: []string{"b"}, Section: "NEW"},
			{Key: "c", Cols: []string{"c"}, Section: "OLD"},
		},
	})
}

func TestSectionHeadingsAreNotSelectable(t *testing.T) {
	m := press(t, sectioned(), "down")
	if got := m.CursorRow().Key; got != "b" {
		t.Fatalf("cursor on %q, want b — headings must not be landed on", got)
	}
	m = press(t, m, "down")
	if got := m.CursorRow().Key; got != "c" {
		t.Fatalf("cursor on %q, want c", got)
	}
}

func TestHeadingsCostScreenHeight(t *testing.T) {
	// Two headings plus a blank line before the second one: the window must
	// reserve three lines it cannot use for rows.
	if got := sectioned().headingLines(); got != 3 {
		t.Fatalf("headings take %d lines, want 3", got)
	}
}

func TestASingleSectionDrawsNoHeadings(t *testing.T) {
	// When everything is in one group the heading is pure noise.
	m := New(Config{
		Columns: []Column{{Title: "N"}},
		Rows: []Row{
			{Key: "a", Cols: []string{"a"}, Section: "ONLY"},
			{Key: "b", Cols: []string{"b"}, Section: "ONLY"},
		},
	})
	if got := m.headingLines(); got != 0 {
		t.Fatalf("headings take %d lines, want 0 for a single section", got)
	}
}

func TestNoSectionsDrawsNoHeadings(t *testing.T) {
	if got := fixture("a", "b").headingLines(); got != 0 {
		t.Fatalf("headings take %d lines, want 0", got)
	}
}

func TestFilteringAwayASectionDropsItsHeading(t *testing.T) {
	m := press(t, sectioned(), "/", "c")
	if got := m.headingLines(); got != 0 {
		t.Fatalf("headings take %d lines, want 0 — only one section survives the filter", got)
	}
}
