package chicle

import (
	"strings"
	"testing"
)

// Config.Filter and Action.Show are one feature between them: a list opened on
// a guess, with an action that acts on the guess rather than on a row.

// guessed is a list of three names opened already narrowed to "web", with an
// action that only appears while something is typed.
func guessed(a *acted) Model {
	return New(Config{
		Columns: []Column{{Title: "NAME"}},
		Rows:    rows("web-prod", "web-staging", "db-prod"),
		Filter:  "web",
		Actions: []Action{
			{Label: "Enter", Run: a.run(Outcome{Result: "entered", Done: true})},
			{
				Label:   "Create",
				Key:     "ctrl+n",
				OnEmpty: true,
				Show:    func(s Selection) bool { return s.Filter != "" },
				Run:     a.run(Outcome{Result: "created", Done: true}),
			},
			{Label: "Exit"},
		},
	})
}

func TestConfigFilterNarrowsBeforeTheFirstKey(t *testing.T) {
	eq(t, keys(guessed(&acted{})), []string{"web-prod", "web-staging"})
}

func TestConfigFilterLeavesTheKeyboardInTheQuery(t *testing.T) {
	m := guessed(&acted{})
	if !m.filtering {
		t.Fatal("want the query focused so the guess can be corrected")
	}
	// Typing continues the query rather than hitting a list binding.
	eq(t, keys(press(t, m, "-prod")), []string{"web-prod"})
}

func TestConfigFilterPutsTheQueryCursorAtTheEnd(t *testing.T) {
	// Backspace must shorten the guess. At qpos 0 it is a no-op and the query
	// would still read "web", so this pins the cursor to the end.
	if got := string(press(t, guessed(&acted{}), "backspace").query); got != "we" {
		t.Fatalf("got query %q, want we", got)
	}
}

func TestEnterOnAPrefilledFilterRunsTheFocusedAction(t *testing.T) {
	var a acted
	m := press(t, guessed(&a), "enter")
	if m.Result() != "entered" {
		t.Fatalf("got %q, want entered", m.Result())
	}
	if got := a.calls[0].Cursor.Key; got != "web-prod" {
		t.Fatalf("acted on %q, want the first match", got)
	}
}

func TestActionReceivesTheFilterText(t *testing.T) {
	var a acted
	press(t, guessed(&a), "ctrl+n")
	if got := a.calls[0].Filter; got != "web" {
		t.Fatalf("got filter %q, want web", got)
	}
}

func TestShowHidesAnActionUntilTheFilterIsTyped(t *testing.T) {
	m := resize(t, fixtureWithCreate(&acted{}), 80, 20)
	if strings.Contains(m.View(), "Create") {
		t.Fatal("Create is offered with no filter to create from")
	}
	if !strings.Contains(press(t, m, "/", "new").View(), "Create") {
		t.Fatal("Create stayed hidden once the filter had text")
	}
}

func TestAHiddenActionIgnoresItsHotkey(t *testing.T) {
	var a acted
	m := press(t, fixtureWithCreate(&a), "ctrl+n")
	if len(a.calls) != 0 {
		t.Fatal("a hidden action fired from its Key")
	}
	if m.quit {
		t.Fatal("a hidden action ended the picker")
	}
}

func TestAShownActionFiresItsHotkeyWhileFiltering(t *testing.T) {
	var a acted
	m := press(t, fixtureWithCreate(&a), "/", "new", "ctrl+n")
	if m.Result() != "created" {
		t.Fatalf("got %q, want created", m.Result())
	}
	if got := a.calls[0].Filter; got != "new" {
		t.Fatalf("got filter %q, want new", got)
	}
}

func TestOnEmptyActionRunsWithNoMatchingRows(t *testing.T) {
	var a acted
	// "zzz" matches none of the rows, which is the whole reason to create.
	m := press(t, fixtureWithCreate(&a), "/", "zzz", "ctrl+n")
	if m.Result() != "created" {
		t.Fatalf("got %q, want created", m.Result())
	}
	if got := a.calls[0].Filter; got != "zzz" {
		t.Fatalf("got filter %q, want zzz", got)
	}
	if got := a.calls[0].Cursor.Key; got != "" {
		t.Fatalf("got cursor %q, want no row", got)
	}
}

func TestActionWithoutOnEmptyStaysInertWithNoMatchingRows(t *testing.T) {
	var a acted
	// Enter has no OnEmpty, so tabbing back and firing it must do nothing.
	m := press(t, fixtureWithCreate(&a), "/", "zzz", "tab", "enter")
	if len(a.calls) != 0 || m.Result() != "" {
		t.Fatalf("Enter ran on an empty list: calls=%d result=%q", len(a.calls), m.Result())
	}
}

func TestFocusSkipsHiddenActions(t *testing.T) {
	// With no filter, Create is hidden: one step right from Enter lands on
	// Exit rather than on the button that is not drawn.
	m := press(t, fixtureWithCreate(&acted{}), "right")
	if got := m.cfg.Actions[m.button].Label; got != "Exit" {
		t.Fatalf("focused %q, want Exit", got)
	}
}

func TestFocusReachesAnActionTheFilterReveals(t *testing.T) {
	m := press(t, fixtureWithCreate(&acted{}), "/", "new", "tab", "right")
	if got := m.cfg.Actions[m.button].Label; got != "Create" {
		t.Fatalf("focused %q, want Create", got)
	}
}

func TestFocusSlidesOffAnActionTheFilterHides(t *testing.T) {
	// Focus Create, then clear the query out from under it.
	m := press(t, fixtureWithCreate(&acted{}), "/", "new", "tab", "right")
	m = press(t, m, "esc")
	if got := m.cfg.Actions[m.button].Label; got != "Enter" {
		t.Fatalf("focused %q, want focus to have slid back to Enter", got)
	}
	if strings.Contains(resize(t, m, 80, 20).View(), "Create") {
		t.Fatal("Create is still drawn with the filter cleared")
	}
}

func TestEnterDoesNotFireAnActionTheFilterJustHid(t *testing.T) {
	var a acted
	m := press(t, fixtureWithCreate(&a), "/", "new", "tab", "right")
	m = press(t, m, "esc", "enter")
	for _, c := range a.calls {
		if c.Filter == "" && m.Result() == "created" {
			t.Fatal("Create ran after the filter that revealed it was cleared")
		}
	}
	if m.Result() == "created" {
		t.Fatalf("got %q, want the hidden action not to have run", m.Result())
	}
}

// fixtureWithCreate is the same shape as guessed, but opens unfiltered.
func fixtureWithCreate(a *acted) Model {
	return New(Config{
		Columns: []Column{{Title: "NAME"}},
		Rows:    rows("web-prod", "web-staging", "db-prod"),
		Actions: []Action{
			{Label: "Enter", Run: a.run(Outcome{Result: "entered", Done: true})},
			{
				Label:   "Create",
				Key:     "ctrl+n",
				OnEmpty: true,
				Show:    func(s Selection) bool { return s.Filter != "" },
				Run:     a.run(Outcome{Result: "created", Done: true}),
			},
			{Label: "Exit"},
		},
	})
}
