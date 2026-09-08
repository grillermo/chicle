package chicle

import (
	"errors"
	"testing"
)

var errBoom = errors.New("boom")

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

func confirmable(a *acted) Model {
	return withActions(Action{
		Label:   "Delete",
		Confirm: func(s Selection) string { return "Delete " + s.Cursor.Key + "?" },
		Run:     a.run(Outcome{Done: true, Result: "deleted"}),
	})
}

func TestConfirmPromptOpensInsteadOfRunning(t *testing.T) {
	var a acted
	m := press(t, confirmable(&a), "enter")
	if !m.confirming {
		t.Fatal("no confirm prompt opened")
	}
	if len(a.calls) != 0 {
		t.Fatal("the action ran before it was confirmed")
	}
	if m.question != "Delete a?" {
		t.Fatalf("question %q, want %q", m.question, "Delete a?")
	}
}

func TestConfirmDefaultsToNo(t *testing.T) {
	var a acted
	m := press(t, confirmable(&a), "enter", "enter")
	if len(a.calls) != 0 {
		t.Fatal("enter on a fresh prompt ran the action — the default must be No")
	}
	if m.confirming {
		t.Fatal("the prompt stayed open after answering")
	}
}

func TestLeftIsYesAndRightIsNo(t *testing.T) {
	var a acted
	// Left, then enter: Yes.
	m := press(t, confirmable(&a), "enter", "left", "enter")
	if len(a.calls) != 1 {
		t.Fatalf("action ran %d times after left+enter, want 1", len(a.calls))
	}
	if got := m.Result(); got != "deleted" {
		t.Fatalf("result %q, want deleted", got)
	}
}

func TestHoldingLeftDoesNotFlipBackToNo(t *testing.T) {
	// The bug this guards: a toggle would land back on No after two presses,
	// so leaning on the key would silently cancel a confirmed delete.
	var a acted
	m := press(t, confirmable(&a), "enter", "left", "left", "left", "enter")
	if len(a.calls) != 1 {
		t.Fatalf("action ran %d times, want 1 — left must be Yes, not a toggle", len(a.calls))
	}
	_ = m
}

func TestYAndNAnswerThePrompt(t *testing.T) {
	var yes acted
	press(t, confirmable(&yes), "enter", "y")
	if len(yes.calls) != 1 {
		t.Fatalf("y ran the action %d times, want 1", len(yes.calls))
	}

	var no acted
	m := press(t, confirmable(&no), "enter", "n")
	if len(no.calls) != 0 {
		t.Fatal("n ran the action")
	}
	if m.confirming {
		t.Fatal("n left the prompt open")
	}
}

func TestHAndLAnswerThePromptToo(t *testing.T) {
	var a acted
	press(t, confirmable(&a), "enter", "h", "enter")
	if len(a.calls) != 1 {
		t.Fatalf("h+enter ran the action %d times, want 1", len(a.calls))
	}
}

func TestEscCancelsThePrompt(t *testing.T) {
	var a acted
	m := press(t, confirmable(&a), "enter", "left", "esc")
	if m.confirming {
		t.Fatal("esc left the prompt open")
	}
	if len(a.calls) != 0 {
		t.Fatal("esc ran the action")
	}
	if m.quit {
		t.Fatal("esc on a prompt quit the whole picker")
	}
}

func TestAnEmptyConfirmStringSkipsThePrompt(t *testing.T) {
	var a acted
	m := withActions(Action{
		Label:   "Go",
		Confirm: func(Selection) string { return "" },
		Run:     a.run(Outcome{Done: true}),
	})
	m = press(t, m, "enter")
	if m.confirming {
		t.Fatal("a prompt opened for an empty Confirm string")
	}
	if len(a.calls) != 1 {
		t.Fatalf("action ran %d times, want 1", len(a.calls))
	}
}

func TestNavigationIsFrozenWhileConfirming(t *testing.T) {
	var a acted
	m := press(t, confirmable(&a), "enter", "down", "j")
	if got := m.CursorRow().Key; got != "a" {
		t.Fatalf("cursor moved to %q while a prompt was open, want a", got)
	}
}

func TestReloadRefreshesTheRowsAfterAnUnfinishedAction(t *testing.T) {
	var a acted
	m := New(Config{
		Columns: []Column{{Title: "N"}},
		Rows:    rows("a", "b"),
		Actions: []Action{{Label: "Go", Run: a.run(Outcome{Status: "ok"})}},
		Reload:  func() ([]Row, error) { return rows("x", "y", "z"), nil },
	})
	m = press(t, m, "enter")
	eq(t, keys(m), []string{"x", "y", "z"})
}

func TestReloadKeepsTheFilter(t *testing.T) {
	m := New(Config{
		Columns: []Column{{Title: "N"}},
		Rows:    rows("alpha", "beta"),
		Actions: []Action{{Label: "Go", Run: (&acted{}).run(Outcome{Status: "ok"})}},
		Reload:  func() ([]Row, error) { return rows("alpha", "beta", "alpine"), nil },
	})
	m = press(t, m, "/", "alp", "enter")
	eq(t, keys(m), []string{"alpha", "alpine"})
}

func TestReloadFailureIsAppendedToTheStatus(t *testing.T) {
	m := New(Config{
		Columns: []Column{{Title: "N"}},
		Rows:    rows("a"),
		Actions: []Action{{Label: "Go", Run: (&acted{}).run(Outcome{Status: "removed a"})}},
		Reload:  func() ([]Row, error) { return nil, errBoom },
	})
	m = press(t, m, "enter")
	if m.status != "removed a; reloading failed: boom" {
		t.Fatalf("status %q, want both the action's message and the failure", m.status)
	}
}

func TestReloadingToNothingEndsThePicker(t *testing.T) {
	// Acting on the last row leaves nothing to act on, so staying up would
	// strand the user on an empty screen.
	m := New(Config{
		Columns: []Column{{Title: "N"}},
		Rows:    rows("a"),
		Actions: []Action{{Label: "Go", Run: (&acted{}).run(Outcome{Status: "removed"})}},
		Reload:  func() ([]Row, error) { return nil, nil },
	})
	m = press(t, m, "enter")
	if !m.quit {
		t.Fatal("picker stayed up with no rows left")
	}
}

func TestNoReloadHookLeavesTheRowsAlone(t *testing.T) {
	m := withActions(Action{Label: "Go", Run: (&acted{}).run(Outcome{Status: "ok"})})
	m = press(t, m, "enter")
	eq(t, keys(m), []string{"a", "b", "c"})
}

func TestKeyedActionFiresWithoutFocus(t *testing.T) {
	var open, del acted
	m := withActions(
		Action{Label: "Open", Run: open.run(Outcome{Done: true})},
		Action{Label: "Delete", Key: "f2", Run: del.run(Outcome{Done: true})},
	)
	// The focused button is still "Open" (button 0); f2 must fire Delete
	// anyway, with no left/right needed first.
	m = press(t, m, "f2")
	if len(del.calls) != 1 || len(open.calls) != 0 {
		t.Fatalf("ran open=%d delete=%d, want open=0 delete=1", len(open.calls), len(del.calls))
	}
	if got := m.Result(); got != "" {
		t.Fatalf("result %q, want empty (Done was set but Run returned no Result)", got)
	}
	if !m.quit {
		t.Fatal("the keyed action did not finish the picker")
	}
}

func TestKeyedActionStillGoesThroughConfirm(t *testing.T) {
	var a acted
	m := withActions(Action{
		Label:   "Delete",
		Key:     "f2",
		Confirm: func(s Selection) string { return "Delete " + s.Cursor.Key + "?" },
		Run:     a.run(Outcome{Done: true, Result: "deleted"}),
	})
	m = press(t, m, "f2")
	if !m.confirming {
		t.Fatal("no confirm prompt opened for the keyed action")
	}
	if len(a.calls) != 0 {
		t.Fatal("the action ran before it was confirmed")
	}
	m = press(t, m, "left", "enter")
	if len(a.calls) != 1 {
		t.Fatalf("action ran %d times after confirming, want 1", len(a.calls))
	}
}

func TestUnmappedKeyFallsThroughUnchanged(t *testing.T) {
	var a acted
	m := withActions(Action{Label: "Go", Key: "f2", Run: a.run(Outcome{Done: true})})
	// f9 matches no action's Key and is not otherwise bound; it must not be
	// swallowed into firing the only action or otherwise doing something odd.
	m = press(t, m, "f9")
	if len(a.calls) != 0 {
		t.Fatal("an unmapped key fired an action")
	}
	if m.quit {
		t.Fatal("an unmapped key quit the picker")
	}
}

// Two actions sharing a Key is a caller mistake chicle does not validate
// anywhere else (Config has no other checks in New or Run), so this is
// documented as last-wins rather than a panic, to stay consistent.
func TestTwoActionsSharingAKeyIsLastWins(t *testing.T) {
	var first, second acted
	m := withActions(
		Action{Label: "First", Key: "f2", Run: first.run(Outcome{Done: true})},
		Action{Label: "Second", Key: "f2", Run: second.run(Outcome{Done: true})},
	)
	m = press(t, m, "f2")
	if len(first.calls) != 0 || len(second.calls) != 1 {
		t.Fatalf("ran first=%d second=%d, want first=0 second=1 (last Key match wins)", len(first.calls), len(second.calls))
	}
}
