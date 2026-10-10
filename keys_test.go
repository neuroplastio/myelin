package myelin

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

func TestParseKey(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Key
	}{
		{"a", "a"},
		{"H", "H"},
		{"ctrl+d", "Control+d"},
		{"ctrl+shift+d", "Control+D"},
		{"ctrl+alt+x", "Control+Alt+x"},
		{"meta+a", "Meta+a"},
		{"ctrl+alt+meta+shift+enter", "Control+Alt+Meta+Shift+Enter"},
		{"esc", "Escape"},
		{"enter", "Enter"},
		{"pgdn", "PageDown"},
		{"pgup", "PageUp"},
		{"f5", "F5"},
		{"f12", "F12"},
		{"f13", "F13"},
		{"f24", "F24"},
		{"ContextMenu", "ContextMenu"},
		{"ctrl+PrintScreen", "Control+PrintScreen"},
		{"shift+1", "Shift+1"},
		{"ctrl+shift+1", "Control+Shift+1"},
		{"shift+!", "Shift+!"},
		{"shift", "Shift"},
		{"ctrl", "Control"},
		{"ctrl+shift", "Control+Shift"},
		{"shift+ctrl", "Shift+Control"},
		{"insert", "Insert"},
		{"shift+tab", "Shift+Tab"},
		{"ctrl+left", "Control+ArrowLeft"},
		{"space", "Space"},
		{"ctrl+space", "Control+Space"},
		{"shift+space", "Shift+Space"},
		{"+", "+"},
		{"ctrl++", "Control++"},
		{"alt+=", "Alt+="},
		{"!", "!"},
		{"?", "?"},
		{"<", "<"},
		{"é", "é"},
		{"É", "É"},
		{"中", "中"},
		{"👩‍💻", "👩‍💻"},
	} {
		got, err := ParseKey(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("ParseKey(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
}

func TestParseKeyRefuses(t *testing.T) {
	for _, tc := range []struct{ in, says string }{
		{"", "empty"},
		{"ctrl+", "modifiers are"},
		{"ctrl+ctrl+a", "ctrl+ twice"},
		{"hyper+a", "modifiers are"},
		{"control+a", "modifiers are"},
		{"Ctrl+d", "lower case: ctrl+"},
		{"SHIFT+h", "lower case: shift+"},
		{"escape", "written short"},
		{"return", "written short"},
		{"pagedown", "written short"},
		{"gg", "separated by spaces"},
		// One spelling per key: the others ask for it.
		{"ctrl+D", "write ctrl+shift+d"},
		{"shift+h", "write H"},
		{"shift+é", "write É"},
		{"shift+ctrl+d", "write ctrl+shift+d"},
		{"alt+ctrl+x", "write ctrl+alt+x"},
		{"shift+meta+alt+ctrl+enter", "write ctrl+alt+meta+shift+enter"},
		{"shift+ǅ", "write Ǆ"},
		{"ctrl+shift+ǅ", "write ctrl+shift+ǆ"},
		{"alt+H", "write alt+shift+h"},
		{"shift+H", "write H"},
		{"shift+ϴ", "write ϴ"},
		{"ctrl+shift+ϴ", "write ctrl+ϴ"},
		{"ctrl+É", "write ctrl+shift+é"},
		{"Escape", "write esc"},
		{"ctrl+PageDown", "write ctrl+pgdn"},
		{"ArrowLeft", "write left"},
		{"F12", "write f12"},
		{"Shift", "write shift"},
		{"Enter", "write enter"},
		{"ESC", "write esc"},
		{"shift+Tab", "write shift+tab"},
		{"<leader>", "keymap's to resolve"},
		{"\x01", "not a key"},
		{"ab+", "modifiers are"},
		{"f0", "written short"},
		{" ", "write space"},
		{"f01", "written short"},
	} {
		got, err := ParseKey(tc.in)
		if err == nil {
			t.Errorf("ParseKey(%q) = %q, want an error", tc.in, got)
			continue
		}
		if !strings.Contains(err.Error(), tc.says) {
			t.Errorf("ParseKey(%q): %v; want it to say %q", tc.in, err, tc.says)
		}
	}
}

// Every named key round-trips, alone and with each modifier: the short
// spelling to its HOTTY name and back.
func TestNamedKeysRoundTrip(t *testing.T) {
	named := maps.Clone(shortNames)
	for _, n := range []string{"1", "12", "13", "24", "35"} {
		named["f"+n] = "F" + n
	}
	for short, name := range named {
		for _, mod := range []string{"", "ctrl+", "alt+", "meta+", "shift+", "ctrl+alt+meta+shift+"} {
			k, err := ParseKey(mod + short)
			if err != nil {
				t.Errorf("ParseKey(%q): %v", mod+short, err)
				continue
			}
			if !strings.HasSuffix(string(k), name) {
				t.Errorf("ParseKey(%q) = %q, want a name ending %q", mod+short, k, name)
			}
			if got := FormatKey(k); got != mod+short {
				t.Errorf("FormatKey(%q) = %q, want %q", k, got, mod+short)
			}
			if got, ok := ParseHottyKey(string(k)); !ok || got != k {
				t.Errorf("ParseHottyKey(%q) = %q, %v; want it unchanged", k, got, ok)
			}
		}
	}
}

func TestFormatKey(t *testing.T) {
	for _, tc := range []struct {
		k    Key
		want string
	}{
		{"Control+d", "ctrl+d"},
		{"Control+A", "ctrl+shift+a"},
		{"Alt+É", "alt+shift+é"},
		{"A", "A"},
		{"Control+Alt+x", "ctrl+alt+x"},
		{"H", "H"},
		{"Space", "space"},
		{"Control++", "ctrl++"},
		{"Shift+Tab", "shift+tab"},
		{"PageDown", "pgdn"},
		{"Shift+1", "shift+1"},
		{"Control+Shift+1", "ctrl+shift+1"},
		{"Shift", "shift"},
		{"Control+Shift", "ctrl+shift"},
		{"F13", "f13"},
		// A named key with no short name is written as HOTTY names it.
		{"ContextMenu", "ContextMenu"},
		{"Control+F0", "ctrl+F0"},
	} {
		if got := FormatKey(tc.k); got != tc.want {
			t.Errorf("FormatKey(%q) = %q, want %q", tc.k, got, tc.want)
		}
	}
}

func TestDisplayKey(t *testing.T) {
	for _, tc := range []struct {
		k    Key
		want string
	}{
		{"ArrowLeft", "←"},
		{"ArrowRight", "→"},
		{"ArrowUp", "↑"},
		{"ArrowDown", "↓"},
		{"Control+ArrowLeft", "ctrl+←"},
		{"Shift+ArrowUp", "shift+↑"},
		{"Tab", "⇥"},
		{"Shift+Tab", "⇧⇥"},
		{"Control+Shift+Tab", "ctrl+⇧⇥"},
		{"Control+A", "ctrl+shift+a"},
		{"Shift+1", "shift+1"},
		{"Control+b", "ctrl+b"},
		{"Escape", "esc"},
		{"Shift+Enter", "shift+enter"},
		{"Space", "space"},
		{"G", "G"},
		{"Control+Alt+Meta+x", "ctrl+alt+meta+x"},
	} {
		if got := DisplayKey(tc.k); got != tc.want {
			t.Errorf("DisplayKey(%q) = %q, want %q", tc.k, got, tc.want)
		}
	}
}

func TestTypesText(t *testing.T) {
	for _, tc := range []struct {
		k    Key
		want bool
	}{
		{"a", true},
		{"A", true},
		{"!", true},
		{"é", true},
		{"Space", true},
		{"Shift+Space", true},
		{"Control+a", false},
		{"Alt+a", false},
		{"Meta+a", false},
		{"Control+Space", false},
		{"Enter", false},
		{"Tab", false},
		{"Shift+Tab", false},
		{"F1", false},
		{"ArrowLeft", false},
		{"Escape", false},
		{"Shift", false},
		{"ContextMenu", false},
	} {
		if got := TypesText(tc.k); got != tc.want {
			t.Errorf("TypesText(%q) = %v, want %v", tc.k, got, tc.want)
		}
	}
}

func TestSeq(t *testing.T) {
	seq, err := ParseSeq("  g  g ")
	if err != nil || !slices.Equal(seq, []Key{"g", "g"}) {
		t.Errorf("ParseSeq(g g) = %q, %v", seq, err)
	}
	if seq, err = ParseSeq("g \u00a0"); err != nil || !slices.Equal(seq, []Key{"g", "\u00a0"}) {
		t.Errorf("a no-break space is a key: %q, %v", seq, err)
	}
	seq, err = ParseSeq("ctrl+x D pgdn")
	if want := []Key{"Control+x", "D", "PageDown"}; err != nil || !slices.Equal(seq, want) {
		t.Errorf("ParseSeq = %q, %v; want %q", seq, err, want)
	}
	if got := FormatSeq(seq); got != "ctrl+x D pgdn" {
		t.Errorf("FormatSeq = %q", got)
	}
	for _, bad := range []string{"", "   ", "<leader> n", "g escape", "g shift+g"} {
		if seq, err := ParseSeq(bad); err == nil {
			t.Errorf("ParseSeq(%q) = %q, want an error", bad, seq)
		}
	}
	// CheckSeq reads a Bind's sequence: <leader> is a key there.
	for s, want := range map[string]string{
		"<leader> p n": "", "g g": "", "<leader>": "",
		"": "an empty key sequence", "g Escape": "write esc", "<leader> shift+g": "write G", "<Leader> n": "not a key",
	} {
		err := CheckSeq(s)
		if (err == nil) != (want == "") || (err != nil && !strings.Contains(err.Error(), want)) {
			t.Errorf("CheckSeq(%q) = %v, want %q", s, err, want)
		}
	}
}

func FuzzParseKey(f *testing.F) {
	for _, s := range []string{"a", "ctrl+d", "shift+h", "ctrl+shift+a", "alt+ctrl+x", "pgdn", "ctrl++", "+",
		"shift+1", "escape", "<leader>", "é", "👩‍💻", "meta+F12", "ctrl+", "Ctrl+Shift+Tab"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		k, err := ParseKey(s)
		// One spelling per key: what ParseKey reads is exactly what FormatKey
		// writes for that key.
		if w := FormatKey(k); err == nil && s != w {
			t.Fatalf("ParseKey(%q) = %q, but that key is written %q", s, k, w)
		}
		if err != nil {
			return
		}
		// What ParseKey returns is a canonical HOTTY name...
		if c, ok := ParseHottyKey(string(k)); !ok || c != k {
			t.Fatalf("ParseKey(%q) = %q, which HOTTY reads as %q, %v", s, k, c, ok)
		}
		// ...that the short spelling writes and reads back.
		short := FormatKey(k)
		if back, err := ParseKey(short); err != nil || back != k {
			t.Fatalf("ParseKey(%q) = %q, written %q, read back as %q, %v", s, k, short, back, err)
		}
		if FormatKey(k) != short {
			t.Fatalf("FormatKey(%q) is not stable", k)
		}
	})
}

// Every name HOTTY reads has a spelling: FormatKey writes it and ParseKey
// reads it back. The exception is a name HOTTY reads but no host sends: a
// case variant of a short name ("Esc", "Up", "Ctrl"), which ParseKey refuses
// in favour of the short name.
func FuzzHottyName(f *testing.F) {
	for _, s := range []string{"a", "A", "Control+A", "Shift+1", "Control+Shift+1", "Shift", "Control+Shift",
		"F13", "ContextMenu", "Shift+Control+a", "Alt+É", "Control+ǅ", "Control+İ", "Meta+Space", "Control++",
		"Esc", "Up", "Ctrl"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, name string) {
		k, ok := ParseHottyKey(name)
		if !ok {
			return
		}
		short := FormatKey(k)
		back, err := ParseKey(short)
		if _, value := split(k); err != nil && looksNamed(value) && caseVariant(value) {
			return
		}
		if err != nil || back != k {
			t.Fatalf("%q is %q, written %q, read back as %q, %v", name, k, short, back, err)
		}
	})
}

// caseVariant reports whether a HOTTY named value is a short name in
// another case, and not the name the short spelling has for that key.
func caseVariant(value string) bool {
	_, short := shortOf(value)
	_, lower := nameOf(strings.ToLower(value))
	return lower && !short
}
