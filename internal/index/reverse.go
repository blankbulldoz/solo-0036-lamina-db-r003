package index

import (
	"lamina/internal/datum"
	"sort"
)

// Reverse maps a referencing column value token to the historical set of
// child primary keys that held it. Like Tree it is a candidate superset:
// entries are never removed between rebuilds, so callers must re-interpret
// each candidate against the version store at their validation watermark.
type Reverse struct{ sets map[string]map[string]bool }

func NewReverse() *Reverse { return &Reverse{sets: map[string]map[string]bool{}} }
func (r *Reverse) Add(value datum.Value, key string) {
	if value.Null() {
		return
	}
	token := value.Key()
	if r.sets[token] == nil {
		r.sets[token] = map[string]bool{}
	}
	r.sets[token][key] = true
}
func (r *Reverse) Candidates(token string) []string {
	out := make([]string, 0, len(r.sets[token]))
	for key := range r.sets[token] {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
