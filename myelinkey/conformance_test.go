package myelinkey

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"testing"
)

// keysJSON is the key-name spec's vectors (SPEC.md §8), embedded so that the
// tests run where there is no file system, as in TinyGo's wasm.
//
//go:embed conformance/keys.json
var keysJSON []byte

// readCase is a text and the key a reading makes of it; nil is refused.
type readCase struct {
	In  string  `json:"in"`
	Key *string `json:"key"`
}

type keyVectors struct {
	Version               string     `json:"version"`
	Physical              []string   `json:"physical"`
	Read                  []readCase `json:"read"`
	Strict                []readCase `json:"strict"`
	ShiftedCapitals       []readCase `json:"shifted_capitals"`
	StrictShiftedCapitals []readCase `json:"strict_shifted_capitals"`
	Mod                   []struct {
		readCase
		Mod             string `json:"mod"`
		Strict          bool   `json:"strict"`
		ShiftedCapitals bool   `json:"shifted_capitals"`
	} `json:"mod"`
	Split []struct {
		Key   string   `json:"key"`
		Mods  []string `json:"mods"`
		Value string   `json:"value"`
		Kind  string   `json:"kind"`
	} `json:"split"`
	Sequences []struct {
		In   string   `json:"in"`
		Keys []string `json:"keys"`
	} `json:"sequences"`
	Display []struct {
		Key     string `json:"key"`
		Display string `json:"display"`
	} `json:"display"`
}

// checkRead checks one reading's vector.
func checkRead(t *testing.T, name string, c readCase, o Options) {
	t.Helper()
	k, err := ParseWith(c.In, o)
	switch {
	case c.Key == nil && err == nil:
		t.Errorf("%s %q = %q, want it refused", name, c.In, k)
	case c.Key != nil && (err != nil || string(k) != *c.Key):
		t.Errorf("%s %q = %q, %v; want %q", name, c.In, k, err, *c.Key)
	}
}

// myelinkey conforms to its spec: every vector of every section.
func TestConformance(t *testing.T) {
	var v keyVectors
	if err := json.Unmarshal(keysJSON, &v); err != nil {
		t.Fatal(err)
	}
	if v.Version != "0.1" {
		t.Fatalf("the vectors are version %q", v.Version)
	}
	if !slices.Equal(v.Physical, physicalNames) {
		t.Errorf("physical = %q, want names.go's table", v.Physical)
	}
	for _, s := range []struct {
		name  string
		o     Options
		cases []readCase
	}{
		{"read", Options{}, v.Read},
		{"strict", Options{Strict: true}, v.Strict},
		{"shifted_capitals", Options{ShiftedCapitals: true}, v.ShiftedCapitals},
		{"strict_shifted_capitals", Options{Strict: true, ShiftedCapitals: true}, v.StrictShiftedCapitals},
	} {
		if len(s.cases) == 0 {
			t.Errorf("%s: no vectors", s.name)
		}
		for _, c := range s.cases {
			checkRead(t, s.name, c, s.o)
		}
	}
	if len(v.Mod) == 0 {
		t.Error("mod: no vectors")
	}
	for _, c := range v.Mod {
		o := Options{Strict: c.Strict, ShiftedCapitals: c.ShiftedCapitals, Mod: map[string]Mods{"super": Super, "ctrl": Ctrl}[c.Mod]}
		if o.Mod == 0 {
			t.Errorf("mod %q: mod is %q", c.In, c.Mod)
		}
		checkRead(t, fmt.Sprintf("mod %s %+v", c.Mod, o), c.readCase, o)
	}
	kinds := map[Kind]string{Character: "character", Physical: "physical", Modifier: "modifier"}
	for _, c := range v.Split {
		mods, value, kind := Split(Key(c.Key))
		var names []string
		for i, n := range modNames {
			if mods&(1<<i) != 0 {
				names = append(names, n)
			}
		}
		if !slices.Equal(names, c.Mods) && len(names)+len(c.Mods) > 0 || value != c.Value || kinds[kind] != c.Kind {
			t.Errorf("split %q = %q, %q, %s; want %q, %q, %s", c.Key, names, value, kinds[kind], c.Mods, c.Value, c.Kind)
		}
	}
	for _, c := range v.Sequences {
		seq, err := ParseSeq(c.In)
		got := make([]string, len(seq))
		for i, k := range seq {
			got[i] = string(k)
		}
		switch {
		case c.Keys == nil && err == nil:
			t.Errorf("sequence %q = %q, want it refused", c.In, got)
		case c.Keys != nil && (err != nil || !slices.Equal(got, c.Keys)):
			t.Errorf("sequence %q = %q, %v; want %q", c.In, got, err, c.Keys)
		}
	}
	for _, c := range v.Display {
		if got := Display(Key(c.Key)); got != c.Display {
			t.Errorf("display %q = %q, want %q", c.Key, got, c.Display)
		}
	}
}
