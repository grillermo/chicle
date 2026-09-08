package chicle

import "testing"

// acted records what an action was handed, so tests can assert on it.
type acted struct {
	calls []Selection
}

func (a *acted) run(out Outcome) func(Selection) Outcome {
	return func(s Selection) Outcome {
		a.calls = append(a.calls, s)
		return out
	}
}

func withActions(as ...Action) Model {
	return New(Config{
		Columns: []Column{{Title: "N"}},
		Rows:    rows("a", "b", "c"),
		Actions: as,
	})
}

func TestNoActionsMeansEnterReturnsTheCursorKey(t *testing.T) {
	m := press(t, fixture("a", "b"), "down", "enter")
	if !m.quit {
		t.Fatal("enter did not finish the picker")
	}
	if got := m.Result(); got != "b" {
		t.Fatalf("result %q, want b", got)
	}
}

func TestEnterRunsTheFocusedAction(t *testing.T) {
	var a acted
	m := withActions(
		Action{Label: "Open", Run: a.run(Outcome{Result: "opened", Done: true})},
	)
	m = press(t, m, "enter")
	if len(a.calls) != 1 {
		t.Fatalf("action ran %d times, want 1", len(a.calls))
	}
	if got := m.Result(); got != "opened" {
		t.Fatalf("result %q, want opened", got)
	}
}

func TestLeftRightMoveBetweenActions(t *testing.T) {
	var open, del acted
	m := withActions(
		Action{Label: "Open", Run: open.run(Outcome{Done: true})},
		Action{Label: "Delete", Run: del.run(Outcome{Done: true})},
	)
	m = press(t, m, "right", "enter")
	if len(del.calls) != 1 || len(open.calls) != 0 {
		t.Fatalf("ran open=%d delete=%d, want open=0 delete=1", len(open.calls), len(del.calls))
	}
}

func TestActionFocusClampsAtBothEnds(t *testing.T) {
	m := withActions(
		Action{Label: "One", Run: (&acted{}).run(Outcome{})},
		Action{Label: "Two", Run: (&acted{}).run(Outcome{})},
	)
	m = press(t, m, "left")
	if m.button != 0 {
		t.Fatalf("button %d after left at the start, want 0 — actions must not cycle", m.button)
	}
	m = press(t, m, "right", "right", "right")
	if m.button != 1 {
		t.Fatalf("button %d, want 1 — actions must clamp at the end", m.button)
	}
}

func TestActionReceivesTheCursorRow(t *testing.T) {
	var a acted
	m := withActions(Action{Label: "Go", Run: a.run(Outcome{Done: true})})
	press(t, m, "down", "enter")
	if got := a.calls[0].Cursor.Key; got != "b" {
		t.Fatalf("action got cursor %q, want b", got)
	}
}

func TestActionReceivesTheFilteredCursorRow(t *testing.T) {
	var a acted
	m := withActions(Action{Label: "Go", Run: a.run(Outcome{Done: true})})
	press(t, m, "/", "c", "enter")
	if got := a.calls[0].Cursor.Key; got != "c" {
		t.Fatalf("action got cursor %q, want c", got)
	}
}

func TestNilRunEndsThePickerWithNoResult(t *testing.T) {
	m := withActions(
		Action{Label: "Go", Run: (&acted{}).run(Outcome{Done: true, Result: "x"})},
		Action{Label: "Exit"}, // no Run
	)
	m = press(t, m, "right", "enter")
	if !m.quit {
		t.Fatal("the bare Exit button did not finish the picker")
	}
	if got := m.Result(); got != "" {
		t.Fatalf("result %q, want empty", got)
	}
}

func TestActionsAreInertWhenNothingMatchesTheFilter(t *testing.T) {
	var a acted
	m := withActions(Action{Label: "Go", Run: a.run(Outcome{Done: true})})
	m = press(t, m, "/", "nonesuch", "enter")
	if len(a.calls) != 0 {
		t.Fatal("action ran with no row under the cursor")
	}
	if m.quit {
		t.Fatal("picker quit with nothing selected")
	}
}

func TestAnUnfinishedActionShowsItsStatusAndKeepsTheListUp(t *testing.T) {
	var a acted
	m := withActions(Action{Label: "Go", Run: a.run(Outcome{Status: "did a thing"})})
	m = press(t, m, "enter")
	if m.quit {
		t.Fatal("picker quit on an Outcome with Done unset")
	}
	if m.status != "did a thing" {
		t.Fatalf("status %q, want %q", m.status, "did a thing")
	}
}
