package myelinkey

import (
	_ "embed"
	"encoding/json"
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

// The keys section: ParseHotty refuses the names the vectors refuse, reads
// a name and its canonical form as the same Key, and makes every name in
// them, decoded terminal input included, a canonical Key.
func TestVectorKeys(t *testing.T) {
	v := readVectors(t)
	n := 0
	for _, x := range v.Keys {
		if x.Key == nil {
			for _, name := range x.Keys {
				if name == "" {
					continue
				}
				n++
				k, ok := ParseHotty(name)
				if !ok {
					t.Errorf("%s: ParseHotty(%q) refused a decoded key", x.Name, name)
					continue
				}
				canonical(t, name, k)
			}
			continue
		}
		n++
		got, ok := ParseHotty(*x.Key)
		switch {
		case x.Canon == nil && ok:
			t.Errorf("%s: ParseHotty(%q) = %q, want it refused", x.Name, *x.Key, got)
		case x.Canon == nil:
		case !ok:
			t.Errorf("%s: ParseHotty(%q) refused, want %q's key", x.Name, *x.Key, *x.Canon)
		default:
			if want, _ := ParseHotty(*x.Canon); got != want {
				t.Errorf("%s: ParseHotty(%q) = %q, and its canonical %q is %q", x.Name, *x.Key, got, *x.Canon, want)
			}
			canonical(t, *x.Key, got)
		}
	}
	if n == 0 {
		t.Fatal("the vectors name no keys")
	}
}

// The keymap section's key names: each key of a formatted keymap reads, and
// is a key of the value it was parsed from. Which bindings a host keeps is
// hotty-go's to test.
func TestVectorKeymapKeys(t *testing.T) {
	v := readVectors(t)
	n := 0
	for _, x := range v.Keymap {
		if x.Format == nil {
			continue
		}
		parsed := map[Key]bool{}
		if x.Parse != nil {
			for _, name := range bindingKeys(*x.Parse) {
				if k, ok := ParseHotty(name); ok {
					parsed[k] = true
				}
			}
		}
		for _, name := range bindingKeys(*x.Format) {
			n++
			k, ok := ParseHotty(name)
			if !ok {
				t.Errorf("%s: %q in the formatted keymap does not read", x.Name, name)
				continue
			}
			canonical(t, name, k)
			if x.Parse != nil && !parsed[k] {
				t.Errorf("%s: %q is no key of the value it was parsed from", x.Name, name)
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
