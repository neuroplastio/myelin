// Package myelinkey names keystrokes: one name for each, whichever way it came
// in, and one way to write it.
//
// A [Key] is a keystroke's canonical name, in Ghostty's vocabulary: its
// modifiers in the order super+, ctrl+, alt+, shift+, then the key, which is
// a physical key, a character or a modifier key:
//
//   - A physical key is a place on the keyboard, named by its W3C UI Events
//     code value in snake case: "escape", "enter", "tab", "space",
//     "page_down", "arrow_up", "f1", "key_a", "digit_1". It is the same key
//     whatever the layout types there.
//   - A character is the one the key types: "a", "?", "ö", "+". It follows
//     the layout. A letter is written small, and Shift with it is written as
//     a modifier ("shift+g", "ctrl+shift+a"). Any other character shows
//     Shift itself: Shift with / types "?", so the key is "?". White space
//     and control characters are keys of their own: "space", "enter",
//     "tab", "backspace", "escape".
//   - A modifier key is a modifier tapped on its own, pressed and released
//     with no other key between, on either side: "shift", "ctrl+shift". Its
//     sides are physical keys: "shift_left", "shift_right".
//
// Two keys are the same keystroke exactly when they are ==. A program sends
// whichever its input gives it, and keys of different kinds never match
// each other: "ctrl+key_w" is the key at the US layout's W, "ctrl+w" is Ctrl
// with the key that types w; "shift_left" is the left Shift, "shift" either.
//
// SPEC.md in this module specifies the names, and conformance/keys.json
// holds its test vectors.
//
// [Parse] reads a key as people write it, the way Ghostty reads a trigger:
// modifiers in any order and case, with the aliases cmd and command for
// super, opt and option for alt, and control for ctrl; a physical key by
// its own name, by its W3C code value ("PageDown", "KeyA") or by another
// keymap's name for it ("pgdn", "esc", "up"); a letter in either case.
// [ParseWith] can read more strictly ([Options]). A sequence is keys
// separated by ASCII white space: "g g" ([ParseSeq], [FormatSeq]).
// [Display] writes a key as a menu shows it, with glyphs for the arrows and
// Tab.
//
// [ParseHotty] reads a key named by HOTTY's key names (SPEC §10.4) as they
// are now: "Control+a", "PageDown", "A".
//
// The package needs nothing but the standard library, and builds with
// TinyGo.
package myelinkey

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Key is a keystroke's canonical name: "a", "shift+g", "ctrl+d",
// "page_down", "ctrl+key_w", "super+ctrl+alt+shift+enter", "?", "ctrl++",
// "shift".
type Key string

// Kind is what a key's value is.
type Kind uint8

// The kinds of key.
const (
	// Character is a key named by the character it types: "a", "?", "+".
	Character Kind = iota
	// Physical is a place on the keyboard: "page_down", "key_a",
	// "shift_left".
	Physical
	// Modifier is a modifier key tapped on its own, either side: "shift",
	// "ctrl".
	Modifier
)

// Mods is a set of modifiers.
type Mods uint8

// The modifiers, in the order a key writes them. Super is the platform's
// command key: Command on macOS, the Windows key or Super elsewhere.
const (
	Super Mods = 1 << iota
	Ctrl
	Alt
	Shift
)

// modNames are the modifiers' names, in the order of their bits.
var modNames = [...]string{"super", "ctrl", "alt", "shift"}

// modAliases are the other names the default reading takes for modifiers,
// Ghostty's.
var modAliases = map[string]Mods{
	"cmd": Super, "command": Super, "opt": Alt, "option": Alt, "control": Ctrl,
}

// Options are how strictly [ParseWith] reads a key. The zero value is the
// default reading, [Parse]'s.
type Options struct {
	// Strict reads only a key's canonical spelling, and refuses any other
	// with it: "Ctrl+D" asks for "ctrl+d", "esc" for "escape",
	// "shift+ctrl+a" for "ctrl+shift+a".
	Strict bool
	// ShiftedCapitals reads a capital letter as Shift with its small
	// letter, as vim does: "G" is "shift+g", "ctrl+A" is "ctrl+shift+a".
	// Without it a letter's case does not matter: "G" is "g".
	ShiftedCapitals bool
}

// Parse reads a key the default way, which is Ghostty's: "ctrl+d",
// "Ctrl+D", "control+d" and "d+ctrl" are all "ctrl+d"; "PageDown", "pgdn"
// and "page_down" are "page_down"; "G" is "g".
func Parse(s string) (Key, error) { return ParseWith(s, Options{}) }

// ParseWith reads a key as [Options] say.
func ParseWith(s string, o Options) (Key, error) {
	if s == "" {
		return "", errors.New("an empty key")
	}
	if isAliasForm(s) {
		return "", fmt.Errorf("%q is an alias, which a keymap resolves", s)
	}
	var (
		mods    Mods
		value   string
		capital bool
		lastMod Mods
	)
	for rest := s; rest != ""; {
		part, after, plus := strings.Cut(rest, "+")
		if plus && after == "" && part != "" {
			return "", fmt.Errorf("%q: no key after the last +", s)
		}
		rest = after
		if m, ok := modOf(part); ok {
			if mods&m != 0 {
				return "", fmt.Errorf("%q: %s twice", s, part)
			}
			mods |= m
			lastMod = m
			continue
		}
		if value != "" {
			return "", fmt.Errorf("%q is more than one key (a sequence's keys are separated by spaces: `g g`)", s)
		}
		switch phys, ok := physicalOf(part); {
		case part == "":
			value = "+"
		case isPhysical[part]:
			value = part
		case isChar(part):
			capital = isCapital(part)
			value = fold(part)
		case ok:
			value = phys
		default:
			return "", fmt.Errorf("%q is not a key%s", s, hint(part))
		}
	}
	if value == "" {
		// Modifiers alone: the last one written is the key, tapped.
		mods &^= lastMod
		value = modName(lastMod)
	}
	if capital && o.ShiftedCapitals {
		mods |= Shift
	}
	k := join(mods, value)
	if o.Strict && s != string(k) && !(o.ShiftedCapitals && s == capitalSpelling(k)) {
		if capital && !o.ShiftedCapitals && mods&Shift == 0 {
			return "", fmt.Errorf("%q: write %s, or %s for Shift with it", s, k, join(mods|Shift, value))
		}
		return "", fmt.Errorf("%q: write %s", s, k)
	}
	return k, nil
}

// modName is the name of one modifier.
func modName(m Mods) string {
	for i, n := range modNames {
		if m == 1<<i {
			return n
		}
	}
	return ""
}

// modOf reads a modifier's name, or one of its aliases, in any case.
func modOf(s string) (Mods, bool) {
	lower := strings.ToLower(s)
	for i, n := range modNames {
		if lower == n {
			return 1 << i, true
		}
	}
	m, ok := modAliases[lower]
	return m, ok
}

// hint says how to write a key that the reading does not know, where it
// can tell.
func hint(part string) string {
	switch {
	case strings.EqualFold(part, "meta"):
		return " (meta is no modifier: write super+ for the command or Windows key, alt+ for Emacs's Meta)"
	case looksWord(part):
		return " (a physical key is named by its W3C code value, page_down or PageDown; a sequence's keys are separated by spaces: `g g`)"
	}
	return ""
}

// join writes a key: its modifiers in order, then its value.
func join(mods Mods, value string) Key {
	var b strings.Builder
	for i, n := range modNames {
		if mods&(1<<i) != 0 {
			b.WriteString(n)
			b.WriteByte('+')
		}
	}
	b.WriteString(value)
	return Key(b.String())
}

// capitalSpelling is how ShiftedCapitals lets a key with Shift and a
// letter be written: "ctrl+shift+a" as "ctrl+A". It is "" for any other
// key.
func capitalSpelling(k Key) string {
	mods, value, kind := Split(k)
	if mods&Shift == 0 || kind != Character {
		return ""
	}
	up := strings.ToUpper(value)
	if !isCapital(up) {
		return ""
	}
	return string(join(mods&^Shift, up))
}

// Split reads a key into its modifiers, its value and the value's kind: a
// character ("?", "a", "+"), a physical key ("page_down", "key_a") or a
// modifier key ("shift").
func Split(k Key) (mods Mods, value string, kind Kind) {
	s := string(k)
	for {
		part, rest, ok := strings.Cut(s, "+")
		m, isMod := Mods(0), false
		for i, n := range modNames {
			if part == n {
				m, isMod = 1<<i, true
			}
		}
		if !ok || rest == "" || !isMod {
			return mods, s, kindOf(s)
		}
		mods |= m
		s = rest
	}
}

// kindOf is the kind of a canonical key's value.
func kindOf(value string) Kind {
	if isPhysical[value] {
		return Physical
	}
	for _, n := range modNames {
		if value == n {
			return Modifier
		}
	}
	return Character
}

// isAliasForm reports whether s is written as an alias: a name between
// < and >, as "<leader>".
func isAliasForm(s string) bool {
	return len(s) > 2 && s[0] == '<' && s[len(s)-1] == '>'
}

// looksWord reports whether s is two or more ASCII letters and digits:
// no character, so a name, or keys run together ("gg").
func looksWord(s string) bool {
	if len(s) < 2 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9') {
			return false
		}
	}
	return true
}

// isChar reports whether s is a character a key can type: one grapheme
// cluster, not white space or a control.
func isChar(s string) bool {
	if s == "" || clusterLen(s) != len(s) {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s)
	return r != utf8.RuneError && r != ' ' && !unicode.IsControl(r)
}

// isCapital reports whether a character is the capital of a small letter,
// which Shift types: "D", "É", not "ǅ" (a title case) or "1".
func isCapital(v string) bool {
	r, n := utf8.DecodeRuneInString(v)
	if n != len(v) {
		return false
	}
	small := unicode.ToLower(r)
	return small != r && unicode.ToUpper(small) == r
}

// fold writes a character's letters small, so that a letter compares in
// either case.
func fold(v string) string { return strings.Map(unicode.ToLower, v) }

// glyphs are how menus show some physical keys.
var glyphs = map[string]string{
	"arrow_left": "←", "arrow_right": "→", "arrow_up": "↑", "arrow_down": "↓", "tab": "⇥",
}

// Display writes a key as a menu shows it: its name, with the arrows and
// Tab as glyphs ("ctrl+←", "⇥"). Shift+Tab is "⇧⇥", one glyph pair a reader
// knows, where "shift+⇥" reads as a chord.
func Display(k Key) string {
	mods, value, _ := Split(k)
	g, ok := glyphs[value]
	if !ok {
		return string(k)
	}
	if value == "tab" && mods&Shift != 0 {
		return string(join(mods&^Shift, "⇧⇥"))
	}
	return string(join(mods, g))
}

// Named lists the physical keys, section by section of the W3C UI Events
// code values: "backquote", "backslash", …, "key_a", …, "page_down", ….
func Named() []Key {
	out := make([]Key, len(physicalNames))
	for i, n := range physicalNames {
		out[i] = Key(n)
	}
	return out
}

// TypesText reports whether a key types text into a field: a character or
// space, with no super, ctrl or alt.
func TypesText(k Key) bool {
	mods, value, kind := Split(k)
	return mods&(Super|Ctrl|Alt) == 0 && (kind == Character || value == "space")
}
