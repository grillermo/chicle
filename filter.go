package chicle

import "strings"

// matches reports whether row satisfies every term. Terms are ANDed, and each
// one may land in any cell: typing "web prod" narrows twice rather than
// searching for the literal phrase.
func matches(r Row, terms []string) bool {
	hay := strings.ToLower(strings.Join(r.Cols, " "))
	for _, t := range terms {
		if !strings.Contains(hay, t) {
			return false
		}
	}
	return true
}

// filtered applies the query to rows. An empty or whitespace-only query
// returns rows untouched, sharing the backing array — no copy on the hot path.
func filtered(all []Row, query []rune) []Row {
	terms := strings.Fields(strings.ToLower(string(query)))
	if len(terms) == 0 {
		return all
	}
	out := make([]Row, 0, len(all))
	for _, r := range all {
		if matches(r, terms) {
			out = append(out, r)
		}
	}
	return out
}
