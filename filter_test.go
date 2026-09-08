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
