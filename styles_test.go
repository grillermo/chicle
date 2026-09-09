package chicle

import (
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// colorModel is a Model whose styles come from a renderer that reports full
// color support, standing in for the /dev/tty renderer Run installs. The
// writer is irrelevant: only the profile decides whether styles emit ANSI.
func colorModel(t *testing.T, m Model) Model {
	t.Helper()
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)
	return m.withRenderer(r)
}

// The bug this guards: chicle draws to /dev/tty, but package-level lipgloss
// styles bind to the default renderer, which sniffs os.Stdout. Under
// `sel=$(picker)` — the exact case /dev/tty output exists to serve — stdout is
// a pipe, so the default renderer reported no color and every style rendered
// plain. Focused and unfocused buttons became byte-identical.
func TestFocusedButtonIsStyledWhenTheRendererHasColor(t *testing.T) {
	m := colorModel(t, New(Config{
		Columns: []Column{{Title: "NAME"}},
		Rows:    rows("alpha"),
		Actions: []Action{{Label: "Enter"}, {Label: "Exit"}},
	}))

	v := resize(t, m, 80, tallEnough).View()

	focused := m.styles.selected.Render("[ Enter ]")
	if !strings.Contains(v, focused) {
		t.Fatalf("focused button not styled; view:\n%q", v)
	}
	if !strings.Contains(v, "[ Exit ]") {
		t.Fatalf("unfocused button missing; view:\n%q", v)
	}
}

func TestFocusedAndUnfocusedButtonsDifferOnAColorTerminal(t *testing.T) {
	m := resize(t, colorModel(t, New(Config{
		Columns: []Column{{Title: "NAME"}},
		Rows:    rows("alpha"),
		Actions: []Action{{Label: "Same"}, {Label: "Same"}},
	})), 80, tallEnough)

	// Two identically labelled buttons: only styling tells them apart. Drawn
	// bare and adjacent is precisely what the flattened renderer produced.
	if v := m.View(); strings.Contains(v, "[ Same ]  [ Same ]") {
		t.Fatalf("focused button not distinguishable from unfocused; view:\n%q", v)
	}
}

func TestCursorRowIsStyledWhenTheRendererHasColor(t *testing.T) {
	m := resize(t, colorModel(t, fixture("alpha", "beta")), 80, tallEnough)

	if v := m.View(); !strings.Contains(v, m.styles.selected.Render("> alpha")) {
		t.Fatalf("cursor row not styled; view:\n%q", v)
	}
}

// withRenderer must reach every style, not just the ones View happens to use
// first — a half-rebound Model would show colored rows but plain buttons.
func TestWithRendererRebindsEveryStyle(t *testing.T) {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)
	s := New(Config{Columns: []Column{{Title: "NAME"}}}).withRenderer(r).styles

	for name, style := range map[string]lipgloss.Style{
		"title": s.title, "header": s.header,
		"selected": s.selected, "dim": s.dim, "plain": s.plain,
	} {
		// A style on a colorless renderer drops the foreground silently, so
		// asking for one and finding no escape sequence means it never moved.
		if got := style.Foreground(lipgloss.Color("212")).Render("x"); !strings.Contains(got, "\x1b") {
			t.Fatalf("style %q still on a colorless renderer: %q", name, got)
		}
	}
}
