package key

import (
	_ "embed"
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

// vectorsJSON is HOTTY's conformance vectors, copied from the hotty
// repository at the commit in hotty-commit (make vectors; make check fails
// when the copy differs). They are embedded, so that the tests run where
// there is no file system, as in TinyGo's wasm.
//
//go:embed testdata/conformance/vectors.json
var vectorsJSON []byte

// vectors are the sections of the conformance vectors that name keys.
type vectors struct {
	Keys []struct {
		Name  string   `json:"name"`
		Key   *string  `json:"key"`
		Canon *string  `json:"canon"`
		Keys  []string `json:"keys"`
	} `json:"keys"`
	Keymap []struct {
		Name   string  `json:"name"`
		Parse  *string `json:"parse"`
		Format *string `json:"format"`
	} `json:"keymap"`
}

func readVectors(t *testing.T) vectors {
	t.Helper()
	var v vectors
	if err := json.Unmarshal(vectorsJSON, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

// The keys section: ParseHotty canonicalizes every name the way the vectors
// do and refuses the ones they refuse, and every canonical name in them,
// decoded terminal input included, has a short spelling that reads back.
func TestVectorKeys(t *testing.T) {
	v := readVectors(t)
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
		got, ok := ParseHotty(*x.Key)
		switch {
		case x.Canon == nil && ok:
			t.Errorf("%s: ParseHotty(%q) = %q, want it refused", x.Name, *x.Key, got)
		case x.Canon != nil && (!ok || string(got) != *x.Canon):
			t.Errorf("%s: ParseHotty(%q) = %q, %v; want %q", x.Name, *x.Key, got, ok, *x.Canon)
		case x.Canon != nil:
			names = append(names, Key(*x.Canon))
		}
	}
	slices.Sort(names)
	names = slices.Compact(names)
	for _, k := range names {
		if c, ok := ParseHotty(string(k)); !ok || c != k {
			t.Errorf("ParseHotty(%q) = %q, %v; want it unchanged", k, c, ok)
		}
		if back, err := Parse(Format(k)); err != nil || back != k {
			t.Errorf("%q written %q reads back as %q, %v", k, Format(k), back, err)
		}
	}
	if canon == 0 || len(names) == 0 {
		t.Fatal("the vectors name no keys")
	}
}

// The keymap section's key names: a data-keys value's keys come out
// canonical. Which bindings a host keeps is hotty-go's to test; here each
// key a formatted keymap has is canonical, and is what ParseHotty makes of a
// key in the value it was parsed from.
func TestVectorKeymapKeys(t *testing.T) {
	v := readVectors(t)
	n := 0
	for _, x := range v.Keymap {
		if x.Format == nil {
			continue
		}
		parsed := map[Key]bool{}
		if x.Parse != nil {
			for _, k := range bindingKeys(*x.Parse) {
				if c, ok := ParseHotty(k); ok {
					parsed[c] = true
				}
			}
		}
		for _, k := range bindingKeys(*x.Format) {
			n++
			c, ok := ParseHotty(k)
			if !ok || string(c) != k {
				t.Errorf("%s: %q in the formatted keymap is not canonical (%q, %v)", x.Name, k, c, ok)
			}
			if x.Parse != nil && !parsed[c] {
				t.Errorf("%s: %q is no key of the value it was parsed from", x.Name, k)
			}
		}
	}
	if n == 0 {
		t.Fatal("the vectors' keymaps have no keys")
	}
}

// bindingKeys is the key of each key=action binding in a data-keys value,
// as written.
func bindingKeys(value string) []string {
	var out []string
	for _, b := range SplitSeq(value) {
		if i := strings.LastIndexByte(b, '='); i >= 0 {
			out = append(out, b[:i])
		}
	}
	return out
}
