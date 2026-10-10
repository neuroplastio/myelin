package myelin

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

// vectorsFile is HOTTY's conformance vectors, copied from the hotty-go this
// module pins (make vectors; make check fails when they differ).
const vectorsFile = "testdata/conformance/vectors.json"

// The §10.4 key names in HOTTY's conformance vectors: their keys section,
// which canonicalizes names and decodes terminal input into them.
func TestVectorKeyNames(t *testing.T) {
	data, err := os.ReadFile(vectorsFile)
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Keys []struct {
			Name  string   `json:"name"`
			Key   *string  `json:"key"`
			Canon *string  `json:"canon"`
			Keys  []string `json:"keys"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}

	// Every name the vectors canonicalize, ParseHottyKey canonicalizes the
	// same, and refuses the ones they refuse.
	var names []Key
	canon := 0
	for _, x := range v.Keys {
		if x.Key == nil {
			for _, k := range x.Keys {
				if k != "" {
					names = append(names, Key(k))
				}
			}
			continue
		}
		canon++
		got, ok := ParseHottyKey(*x.Key)
		switch {
		case x.Canon == nil && ok:
			t.Errorf("%s: ParseHottyKey(%q) = %q, want it refused", x.Name, *x.Key, got)
		case x.Canon != nil && (!ok || string(got) != *x.Canon):
			t.Errorf("%s: ParseHottyKey(%q) = %q, %v; want %q", x.Name, *x.Key, got, ok, *x.Canon)
		case x.Canon != nil:
			names = append(names, Key(*x.Canon))
		}
	}
	slices.Sort(names)
	names = slices.Compact(names)

	// Each is canonical, and has a spelling: FormatKey writes it and
	// ParseKey reads it back.
	for _, k := range names {
		if c, ok := ParseHottyKey(string(k)); !ok || c != k {
			t.Errorf("ParseHottyKey(%q) = %q, %v; want it unchanged", k, c, ok)
		}
		if back, err := ParseKey(FormatKey(k)); err != nil || back != k {
			t.Errorf("%q written %q reads back as %q, %v", k, FormatKey(k), back, err)
		}
	}
	t.Logf("%d canonicalization vectors, %d distinct valid names", canon, len(names))
}
