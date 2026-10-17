package myelinkey

import (
	"slices"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Key
	}{
		{"a", "a"},
		{"ctrl+d", "ctrl+d"},
		{"ctrl+alt+x", "ctrl+alt+x"},
		{"super+ctrl+alt+shift+enter", "super+ctrl+alt+shift+enter"},
		// Modifiers in any order and case, and Ghostty's aliases.
		{"shift+ctrl+d", "ctrl+shift+d"},
		{"d+ctrl", "ctrl+d"},
		{"Ctrl+D", "ctrl+d"},
		{"control+d", "ctrl+d"},
		{"CONTROL+d", "ctrl+d"},
		{"cmd+s", "super+s"},
		{"command+s", "super+s"},
		{"opt+b", "alt+b"},
		{"option+b", "alt+b"},
		// A letter in either case.
		{"G", "g"},
		{"shift+G", "shift+g"},
		{"ctrl+É", "ctrl+é"},
		{"ǅ", "ǆ"},
		// Physical keys: their names, W3C code values, other keymaps' names.
		{"escape", "escape"},
		{"Escape", "escape"},
		{"ESCAPE", "escape"},
		{"esc", "escape"},
		{"page_down", "page_down"},
		{"PageDown", "page_down"},
		{"pagedown", "page_down"},
		{"pgdn", "page_down"},
		{"pgup", "page_up"},
		{"up", "arrow_up"},
		{"ArrowLeft", "arrow_left"},
		{"ctrl+left", "ctrl+arrow_left"},
		{"KeyA", "key_a"},
		{"ctrl+key_w", "ctrl+key_w"},
		{"Digit1", "digit_1"},
		{"F12", "f12"},
		{"f25", "f25"},
		{"Numpad0", "numpad_0"},
		{"NumpadEnter", "numpad_enter"},
		{"ContextMenu", "context_menu"},
		{"LaunchApp1", "launch_app_1"},
		{"ShiftLeft", "shift_left"},
		{"space", "space"},
		{"Space", "space"},
		{"ctrl+space", "ctrl+space"},
		{"shift+tab", "shift+tab"},
		{"shift+key_a", "shift+key_a"},
		{"comma", "comma"},
		{"ctrl+comma", "ctrl+comma"},
		// Characters.
		{"?", "?"},
		{"!", "!"},
		{",", ","},
		{"<", "<"},
		{">", ">"},
		{"1", "1"},
		{"shift+1", "shift+1"},
		{"+", "+"},
		{"ctrl++", "ctrl++"},
		{"alt+=", "alt+="},
		{"é", "é"},
		{"é", "é"},
		{"中", "中"},
		{"👩‍💻", "👩‍💻"},
		{" ", " "},
		// A modifier alone is its key, tapped; with others, the last one
		// written is the key.
		{"shift", "shift"},
		{"Shift", "shift"},
		{"ctrl+shift", "ctrl+shift"},
		{"shift+ctrl", "shift+ctrl"},
		{"cmd", "super"},
		{"control", "ctrl"},
		{"alt+super+ctrl", "super+alt+ctrl"},
		{"shift_left", "shift_left"},
	} {
		got, err := Parse(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("Parse(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
}

func TestParseRefuses(t *testing.T) {
	for _, tc := range []struct{ in, says string }{
		{"", "empty"},
		{"ctrl+", "no key after the last +"},
		{"a+", "no key after the last +"},
		{"shift+shift", "shift twice"},
		{"ctrl+ctrl+a", "ctrl twice"},
		{"ctrl+control+a", "control twice"},
		{"meta+a", "write super+ for the command or Windows key"},
		{"hyper+a", "not a key"},
		{"gg", "separated by spaces"},
		{"g+g", "more than one key"},
		{"++", "more than one key"},
		{"return", "named by its W3C code value"},
		{"f0", "not a key"},
		{"f26", "not a key"},
		{"key_aa", "not a key"},
		{"<leader>", "an alias, which a keymap resolves"},
		{"<my_key>", "an alias"},
		{"<>", "not a key"},
		{" ", "not a key"},
		{"\x01", "not a key"},
		{"\t", "not a key"},
	} {
		got, err := Parse(tc.in)
		if err == nil {
			t.Errorf("Parse(%q) = %q, want an error", tc.in, got)
			continue
		}
		if !strings.Contains(err.Error(), tc.says) {
			t.Errorf("Parse(%q): %v; want it to say %q", tc.in, err, tc.says)
		}
	}
}

// Strict reads the canonical spelling and asks for it in place of any
// other.
func TestParseStrict(t *testing.T) {
	strict := Options{Strict: true}
	for _, s := range []string{"a", "ctrl+d", "shift+g", "super+ctrl+alt+shift+enter", "page_down", "ctrl++", "+",
		"?", "shift+1", "key_a", "ctrl+arrow_left", "é", "shift", "ctrl+shift", "shift+ctrl"} {
		if k, err := ParseWith(s, strict); err != nil || string(k) != s {
			t.Errorf("strict %q = %q, %v", s, k, err)
		}
	}
	for _, tc := range []struct{ in, says string }{
		{"Ctrl+d", "write ctrl+d"},
		{"control+d", "write ctrl+d"},
		{"shift+ctrl+d", "write ctrl+shift+d"},
		{"d+ctrl", "write ctrl+d"},
		{"cmd+s", "write super+s"},
		{"esc", "write escape"},
		{"PageDown", "write page_down"},
		{"up", "write arrow_up"},
		{"F1", "write f1"},
		{"G", "write g, or shift+g for Shift with it"},
		{"ctrl+É", "write ctrl+é, or ctrl+shift+é for Shift with it"},
		{"shift+G", `"shift+G": write shift+g`},
		{"Shift", "write shift"},
		{"control", "write ctrl"},
	} {
		got, err := ParseWith(tc.in, strict)
		if err == nil {
			t.Errorf("strict %q = %q, want an error", tc.in, got)
			continue
		}
		if !strings.Contains(err.Error(), tc.says) {
			t.Errorf("strict %q: %v; want it to say %q", tc.in, err, tc.says)
		}
	}
}

// ShiftedCapitals reads a capital as Shift with its letter, also when
// strict, where both spellings are the key's.
func TestShiftedCapitals(t *testing.T) {
	for _, o := range []Options{{ShiftedCapitals: true}, {ShiftedCapitals: true, Strict: true}} {
		for _, tc := range []struct {
			in   string
			want Key
		}{
			{"G", "shift+g"},
			{"shift+g", "shift+g"},
			{"ctrl+A", "ctrl+shift+a"},
			{"ctrl+shift+a", "ctrl+shift+a"},
			{"É", "shift+é"},
			{"g", "g"},
			{"?", "?"},
			{"1", "1"},
			{"ǅ", "ǆ"}, // a title case is no capital
		} {
			if o.Strict && tc.in == "ǅ" {
				continue
			}
			got, err := ParseWith(tc.in, o)
			if err != nil || got != tc.want {
				t.Errorf("%+v: %q = %q, %v; want %q", o, tc.in, got, err, tc.want)
			}
		}
	}
	if k, err := ParseWith("shift+G", Options{ShiftedCapitals: true}); err != nil || k != "shift+g" {
		t.Errorf("shift+G = %q, %v", k, err)
	}
}

// Every physical key reads as itself and as its W3C code value, alone and
// with modifiers, and the strict reading takes its name.
func TestPhysicalKeys(t *testing.T) {
	for _, n := range physicalNames {
		for _, mod := range []string{"", "ctrl+", "shift+", "super+ctrl+alt+shift+"} {
			k, err := ParseWith(mod+n, Options{Strict: true})
			if err != nil || string(k) != mod+n {
				t.Errorf("strict %q = %q, %v", mod+n, k, err)
			}
			mods, value, kind := Split(k)
			if value != n || kind != Physical || join(mods, value) != k {
				t.Errorf("Split(%q) = %v, %q, %v", k, mods, value, kind)
			}
		}
		if k, err := Parse(w3c(n)); err != nil || string(k) != n {
			t.Errorf("Parse(%q) = %q, %v; want %q", w3c(n), k, err, n)
		}
	}
}

// w3c is a physical key's W3C code value: "page_down" is "PageDown", "f1"
// is "F1".
func w3c(n string) string {
	var b strings.Builder
	for _, w := range strings.Split(n, "_") {
		b.WriteString(strings.ToUpper(w[:1]) + w[1:])
	}
	return b.String()
}

func TestSplit(t *testing.T) {
	for _, tc := range []struct {
		k     Key
		mods  Mods
		value string
		kind  Kind
	}{
		{"a", 0, "a", Character},
		{"ctrl+shift+a", Ctrl | Shift, "a", Character},
		{"+", 0, "+", Character},
		{"ctrl++", Ctrl, "+", Character},
		{"super+ctrl+alt+shift+enter", Super | Ctrl | Alt | Shift, "enter", Physical},
		{"page_down", 0, "page_down", Physical},
		{"ctrl+key_w", Ctrl, "key_w", Physical},
		{"shift_left", 0, "shift_left", Physical},
		{"shift", 0, "shift", Modifier},
		{"ctrl+shift", Ctrl, "shift", Modifier},
		{"shift+ctrl", Shift, "ctrl", Modifier},
	} {
		mods, value, kind := Split(tc.k)
		if mods != tc.mods || value != tc.value || kind != tc.kind {
			t.Errorf("Split(%q) = %v, %q, %v; want %v, %q, %v", tc.k, mods, value, kind, tc.mods, tc.value, tc.kind)
		}
	}
}

func TestDisplay(t *testing.T) {
	for _, tc := range []struct {
		k    Key
		want string
	}{
		{"arrow_left", "←"},
		{"arrow_right", "→"},
		{"arrow_up", "↑"},
		{"arrow_down", "↓"},
		{"ctrl+arrow_left", "ctrl+←"},
		{"shift+arrow_up", "shift+↑"},
		{"tab", "⇥"},
		{"shift+tab", "⇧⇥"},
		{"ctrl+shift+tab", "ctrl+⇧⇥"},
		{"ctrl+shift+a", "ctrl+shift+a"},
		{"escape", "escape"},
		{"page_down", "page_down"},
		{"shift+g", "shift+g"},
		{"super+x", "super+x"},
		{"+", "+"},
	} {
		if got := Display(tc.k); got != tc.want {
			t.Errorf("Display(%q) = %q, want %q", tc.k, got, tc.want)
		}
	}
}

func TestTypesText(t *testing.T) {
	for _, tc := range []struct {
		k    Key
		want bool
	}{
		{"a", true},
		{"shift+a", true},
		{"!", true},
		{"é", true},
		{"space", true},
		{"shift+space", true},
		{"ctrl+a", false},
		{"alt+a", false},
		{"super+a", false},
		{"ctrl+space", false},
		{"enter", false},
		{"tab", false},
		{"shift+tab", false},
		{"f1", false},
		{"arrow_left", false},
		{"escape", false},
		{"key_a", false},
		{"shift_left", false},
		{"shift", false},
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
	if seq, err = ParseSeq("g  "); err != nil || !slices.Equal(seq, []Key{"g", " "}) {
		t.Errorf("a no-break space is a key: %q, %v", seq, err)
	}
	seq, err = ParseSeq("Ctrl+X d pgdn")
	if want := []Key{"ctrl+x", "d", "page_down"}; err != nil || !slices.Equal(seq, want) {
		t.Errorf("ParseSeq = %q, %v; want %q", seq, err, want)
	}
	if got := FormatSeq(seq); got != "ctrl+x d page_down" {
		t.Errorf("FormatSeq = %q", got)
	}
	for _, bad := range []string{"", "   ", "<leader> n", "g meta+g"} {
		if seq, err := ParseSeq(bad); err == nil {
			t.Errorf("ParseSeq(%q) = %q, want an error", bad, seq)
		}
	}
	// SplitSeq leaves the words as written, for a keymap's own ones.
	if got := SplitSeq(" <leader>\tp  n\n"); !slices.Equal(got, []string{"<leader>", "p ", "n"}) {
		t.Errorf("SplitSeq = %q", got)
	}
}

// Named is every physical key once, in Ghostty's order.
func TestNamed(t *testing.T) {
	named := Named()
	if len(named) != len(physicalNames) || len(isPhysical) != len(physicalNames) {
		t.Fatalf("Named() has %d keys, the set %d; want %d", len(named), len(isPhysical), len(physicalNames))
	}
	for i, k := range named {
		if string(k) != physicalNames[i] {
			t.Errorf("Named()[%d] = %q", i, k)
		}
	}
	for other, n := range otherNames {
		if !isPhysical[n] || isPhysical[other] {
			t.Errorf("other name %q for %q", other, n)
		}
	}
}

func TestParseHotty(t *testing.T) {
	for _, tc := range []struct {
		name string
		want Key
	}{
		{"a", "a"},
		{"A", "shift+a"},
		{"Shift+a", "shift+a"},
		{"Control+a", "ctrl+a"},
		{"Control+A", "ctrl+shift+a"},
		{"Shift+Control+a", "ctrl+shift+a"},
		{"Alt+É", "alt+shift+é"},
		{"!", "!"},
		{"Shift+1", "shift+1"},
		{"Control+Shift+1", "ctrl+shift+1"},
		{"Meta+Enter", "super+enter"},
		{"Control+Alt+Meta+Shift+Enter", "super+ctrl+alt+shift+enter"},
		{"PageDown", "page_down"},
		{"ArrowLeft", "arrow_left"},
		{"Shift+Tab", "shift+tab"},
		{"F12", "f12"},
		{"ContextMenu", "context_menu"},
		{"Space", "space"},
		{" ", "space"},
		{"Control+ ", "ctrl+space"},
		{"Control++", "ctrl++"},
		{"+", "+"},
		{"Alt+<", "alt+<"},
		{"Shift", "shift"},
		{"Control+Shift", "ctrl+shift"},
		{"Meta", "super"},
		{"Shift+Control", "shift+ctrl"},
	} {
		got, ok := ParseHotty(tc.name)
		if !ok || got != tc.want {
			t.Errorf("ParseHotty(%q) = %q, %v; want %q", tc.name, got, ok, tc.want)
		}
	}
	for _, bad := range []string{"", "Hyper+x", "ab", "enter", "Control+Control+a", "\x01", "Shift+", "Shift+Shift", "Mod+s"} {
		if got, ok := ParseHotty(bad); ok {
			t.Errorf("ParseHotty(%q) = %q, want it refused", bad, got)
		}
	}
}

// canonical checks that k is a canonical key: the strict reading takes it
// as itself, and Split and join take it apart and back.
func canonical(t *testing.T, from string, k Key) {
	t.Helper()
	if back, err := ParseWith(string(k), Options{Strict: true}); err != nil || back != k {
		t.Fatalf("%q is %q, which reads back as %q, %v", from, k, back, err)
	}
	if mods, value, _ := Split(k); join(mods, value) != k {
		t.Fatalf("%q is %q, which Split takes apart into %v, %q", from, k, mods, value)
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{"a", "ctrl+d", "shift+h", "Ctrl+Shift+A", "d+ctrl", "pgdn", "ctrl++", "+",
		"shift+1", "Escape", "<leader>", "é", "👩‍💻", "cmd+F12", "ctrl+", "KeyA", "G",
		"mod+s", "Mod+Shift+ArrowLeft", "shift+mod", "mod+ctrl+s"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		for _, o := range []Options{{}, {ShiftedCapitals: true}, {Mod: Ctrl}, {Mod: Super, ShiftedCapitals: true}} {
			k, err := ParseWith(s, o)
			if err != nil {
				continue
			}
			canonical(t, s, k)
			// The strict reading takes s exactly when it is k's spelling.
			strict, err := ParseWith(s, Options{Strict: true, ShiftedCapitals: o.ShiftedCapitals, Mod: o.Mod})
			if err == nil && strict != k {
				t.Fatalf("%q is %q, and strictly %q", s, k, strict)
			}
			// mod only adds spellings: what reads without it reads the same.
			if o.Mod != 0 {
				if plain, err := ParseWith(s, Options{ShiftedCapitals: o.ShiftedCapitals}); err == nil && plain != k {
					t.Fatalf("%q is %q, and with mod %q", s, plain, k)
				}
			}
		}
	})
}

func FuzzHottyName(f *testing.F) {
	for _, s := range []string{"a", "A", "Control+A", "Shift+1", "Control+Shift+1", "Shift", "Control+Shift",
		"F13", "ContextMenu", "Shift+Control+a", "Alt+É", "Control+ǅ", "Control+İ", "Meta+Space", "Control++"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, name string) {
		if k, ok := ParseHotty(name); ok {
			canonical(t, name, k)
		}
	})
}
