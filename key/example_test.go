package key_test

import (
	"fmt"

	"github.com/neuroplastio/myelin/key"
)

// Every key has one spelling, and any other is refused with it.
func ExampleParse() {
	for _, s := range []string{"ctrl+d", "pgdn", "H", "ctrl+shift+a", "shift+h", "Escape", "alt+ctrl+x"} {
		k, err := key.Parse(s)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("%s is %s\n", s, k)
	}
	// Output:
	// ctrl+d is Control+d
	// pgdn is PageDown
	// H is H
	// ctrl+shift+a is Control+A
	// "shift+h": write H
	// "Escape": write esc
	// "alt+ctrl+x": write ctrl+alt+x
}

// A key that comes in named by a HOTTY host, a terminal decoder or a
// toolkit's adapter is made canonical, and then compares with ==.
func ExampleParseHotty() {
	for _, name := range []string{"Shift+Control+a", "Control+ ", "Alt+Shift+1", "Hyper+x"} {
		k, ok := key.ParseHotty(name)
		if !ok {
			fmt.Printf("%q is not a key\n", name)
			continue
		}
		fmt.Printf("%q is %s, written %s\n", name, k, key.Format(k))
	}
	// Output:
	// "Shift+Control+a" is Control+A, written ctrl+shift+a
	// "Control+ " is Control+Space, written ctrl+space
	// "Alt+Shift+1" is Alt+Shift+1, written alt+shift+1
	// "Hyper+x" is not a key
}

// Menus show keys with glyphs for the arrows and Tab.
func ExampleDisplay() {
	for _, s := range []string{"ctrl+left", "shift+tab", "pgdn", "G"} {
		k, _ := key.Parse(s)
		fmt.Printf("%s shows as %s\n", s, key.Display(k))
	}
	// Output:
	// ctrl+left shows as ctrl+←
	// shift+tab shows as ⇧⇥
	// pgdn shows as pgdn
	// G shows as G
}
