package chicle

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// key turns a key name into the message Bubble Tea would deliver. Anything not
// named here is treated as literal text to type.
func key(s string) tea.KeyMsg {
	named := map[string]tea.KeyType{
		"up": tea.KeyUp, "down": tea.KeyDown,
		"left": tea.KeyLeft, "right": tea.KeyRight,
		"enter": tea.KeyEnter, "esc": tea.KeyEsc, "tab": tea.KeyTab,
		"space": tea.KeySpace, "backspace": tea.KeyBackspace,
		"ctrl+c": tea.KeyCtrlC, "ctrl+w": tea.KeyCtrlW,
		"ctrl+a": tea.KeyCtrlA, "ctrl+e": tea.KeyCtrlE,
		"ctrl+u": tea.KeyCtrlU,
		"f2":     tea.KeyF2, "f9": tea.KeyF9,
	}
	if t, ok := named[s]; ok {
		return tea.KeyMsg{Type: t}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// press feeds keys to the model in order and returns the model afterwards.
// Multi-character arguments that are not key names are typed one rune at a
// time, so press(m, "/", "foo") types f, o, o.
func press(t *testing.T, m Model, keys ...string) Model {
	t.Helper()
	for _, k := range keys {
		msgs := []tea.KeyMsg{key(k)}
		if _, named := map[string]bool{
			"up": true, "down": true, "left": true, "right": true,
			"enter": true, "esc": true, "tab": true, "space": true,
			"backspace": true, "ctrl+c": true, "ctrl+w": true,
			"ctrl+a": true, "ctrl+e": true, "ctrl+u": true,
			"f2": true, "f9": true,
		}[k]; !named && len([]rune(k)) > 1 {
			msgs = nil
			for _, r := range k {
				msgs = append(msgs, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			}
		}
		for _, msg := range msgs {
			next, _ := m.Update(msg)
			m = next.(Model)
		}
	}
	return m
}

// resize delivers a terminal size, which the windowing code needs before it
// can decide what fits.
func resize(t *testing.T, m Model, w, h int) Model {
	t.Helper()
	next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return next.(Model)
}

// rows builds a Row per name, with the name as both Key and only cell.
func rows(names ...string) []Row {
	out := make([]Row, 0, len(names))
	for _, n := range names {
		out = append(out, Row{Key: n, Cols: []string{n}})
	}
	return out
}

// fixture is a plain single-select list over the given names.
func fixture(names ...string) Model {
	return New(Config{
		Columns: []Column{{Title: "NAME"}},
		Rows:    rows(names...),
	})
}

// keys is the Key of every currently visible row, for comparing against a
// wanted list.
func keys(m Model) []string {
	var out []string
	for _, r := range m.visibleRows() {
		out = append(out, r.Key)
	}
	return out
}

func eq(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
}
