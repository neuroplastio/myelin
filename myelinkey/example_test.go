package myelinkey_test

import (
	"fmt"

	"github.com/neuroplastio/myelin/myelinkey"
)

// A key is read the way Ghostty reads one: modifiers in any order, aliases
// and W3C code values, a letter in either case.
func ExampleParse() {
	for _, s := range []string{"ctrl+d", "Shift+Ctrl+A", "cmd+s", "PageDown", "pgdn", "G", "ctrl+key_w", "?"} {
		k, err := myelinkey.Parse(s)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("%s is %s\n", s, k)
	}
	// Output:
	// ctrl+d is ctrl+d
	// Shift+Ctrl+A is ctrl+shift+a
	// cmd+s is super+s
	// PageDown is page_down
	// pgdn is page_down
	// G is g
	// ctrl+key_w is ctrl+key_w
	// ? is ?
}

// A program can read more strictly: only the canonical spelling, or a
// capital as Shift with its letter, as vim writes keys.
func ExampleParseWith() {
	strict := myelinkey.Options{Strict: true}
	vim := myelinkey.Options{ShiftedCapitals: true}
	for _, s := range []string{"Ctrl+d", "esc", "G"} {
		if _, err := myelinkey.ParseWith(s, strict); err != nil {
			fmt.Println(err)
		}
	}
	k, _ := myelinkey.ParseWith("G", vim)
	fmt.Println("with ShiftedCapitals, G is", k)
	// Output:
	// "Ctrl+d": write ctrl+d
	// "esc": write escape
	// "G": write g, or shift+g for Shift with it
	// with ShiftedCapitals, G is shift+g
}

// A key that comes in named as HOTTY names keys now is read into a Key, and
// then compares with ==.
func ExampleParseHotty() {
	for _, name := range []string{"Shift+Control+a", "A", "Control+ ", "Meta+PageDown", "Hyper+x"} {
		k, ok := myelinkey.ParseHotty(name)
		if !ok {
			fmt.Printf("%q is not a key\n", name)
			continue
		}
		fmt.Printf("%q is %s\n", name, k)
	}
	// Output:
	// "Shift+Control+a" is ctrl+shift+a
	// "A" is shift+a
	// "Control+ " is ctrl+space
	// "Meta+PageDown" is super+page_down
	// "Hyper+x" is not a key
}

// Menus show keys with glyphs for the arrows and Tab.
func ExampleDisplay() {
	for _, s := range []string{"ctrl+left", "shift+tab", "pgdn", "shift+g"} {
		k, _ := myelinkey.Parse(s)
		fmt.Printf("%s shows as %s\n", s, myelinkey.Display(k))
	}
	// Output:
	// ctrl+left shows as ctrl+←
	// shift+tab shows as ⇧⇥
	// pgdn shows as page_down
	// shift+g shows as shift+g
}
