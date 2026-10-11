//go:build !race

package myelin

import (
	"testing"

	"github.com/neuroplastio/myelin/myelinkey"
)

// What a key, a match and a menu allocate stays where the benchmarks put
// it: a key no binding starts with is its Result's slice alone, a prefix
// costs nothing, and a menu allocates its items, not the evaluation of each
// binding. CEL's own allocations evaluating a context count, so the bounds
// on those leave room. (The race detector drops pooled values, so the test
// does not run under it.)
func TestAllocs(t *testing.T) {
	km, p := benchKeymap(t)
	for _, tc := range []struct {
		name string
		max  float64
		f    func()
	}{
		{"unbound key", 1, func() { Feed(km, Pending{}, "z", p, now) }},
		{"prefix", 0, func() { km.Match([]myelinkey.Key{leader, "k"}, p) }},
		{"contextual match", 8, func() { km.Match([]myelinkey.Key{"ctrl+k"}, p) }},
		{"key column", 8, func() { km.KeyFor("panel.k", p) }},
		{"leader menu", 100, func() { km.Menu([]myelinkey.Key{leader}, p, nil) }},
		{"effective", 50, func() { km.Effective(deepPath) }},
	} {
		if n := testing.AllocsPerRun(100, tc.f); n > tc.max {
			t.Errorf("%s: %v allocations, want at most %v", tc.name, n, tc.max)
		}
	}
}
