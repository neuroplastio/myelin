package myelinkey

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ParseHotty reads a key named as HOTTY's key names are now (SPEC §10.4),
// its modifiers in any order, and returns it as a [Key]: "Control+a" is
// "ctrl+a", "Shift+Control+a" and "Control+A" are "ctrl+shift+a", "A" is
// "shift+a", "!" is "!", "PageDown" is "page_down", "Meta+Enter" is
// "super+enter", "Control+ " is "ctrl+space". A modifier alone is its
// modifier key: "Shift" is "shift", "Control+Shift" is "ctrl+shift". ok is
// false for a name that does not parse.
//
// HOTTY's names are moving to the Key's own; this reads the names hosts,
// terminal decoders and toolkit adapters write until then.
func ParseHotty(name string) (Key, bool) {
	hmods, value, ok := splitName(name)
	if !ok {
		return "", false
	}
	hmods, value = shiftInChar(hmods, value)
	var mods Mods
	for i, m := range hottyMods {
		if hmods&(1<<i) != 0 {
			mods |= m
		}
	}
	switch {
	case value == " ":
		value = "space"
	case isChar(value):
		if isCapital(value) {
			mods |= Shift
		}
		value = fold(value)
	default:
		if m, ok := hottyModKeys[value]; ok {
			if mods&m != 0 {
				return "", false
			}
			value = modName(m)
		} else if value, ok = physicalOf(value); !ok {
			return "", false
		}
	}
	return join(mods, value), true
}

// HOTTY's modifiers, in the order it writes them, and the modifiers they
// are.
var (
	hottyModNames = [...]string{"Control", "Alt", "Meta", "Shift"}
	hottyMods     = [...]Mods{Ctrl, Alt, Super, Shift}
)

const hottyShift = 1 << 3

// hottyModKeys are HOTTY's lone modifiers, the modifier keys.
var hottyModKeys = map[string]Mods{"Control": Ctrl, "Alt": Alt, "Meta": Super, "Shift": Shift}

// splitName reads a key's name, its modifiers in any order. The value of
// the space bar is " ".
func splitName(name string) (mods int, value string, ok bool) {
	var head string
	switch i := strings.LastIndexByte(name, '+'); {
	case name == "+":
		value = "+"
	case strings.HasSuffix(name, "++") && len(name) > 2:
		head, value = name[:len(name)-2], "+"
	case i >= 0:
		head, value = name[:i], name[i+1:]
	default:
		value = name
	}
	if head != "" {
	next:
		for _, m := range strings.Split(head, "+") {
			for i, known := range hottyModNames {
				if m == known && mods&(1<<i) == 0 {
					mods |= 1 << i
					continue next
				}
			}
			return 0, "", false
		}
	}
	if value == "Space" {
		value = " "
	}
	if !isHottyChar(value) && !isHottyNamed(value) {
		return 0, "", false
	}
	return mods, value, true
}

// shiftInChar shows Shift in a letter where it can, as HOTTY does: Shift
// with a small letter is its capital, and Shift with a capital is the
// capital.
func shiftInChar(mods int, value string) (int, string) {
	if mods&hottyShift == 0 || !isHottyChar(value) {
		return mods, value
	}
	r, n := utf8.DecodeRuneInString(value)
	if n != len(value) {
		return mods, value
	}
	if up := unicode.ToUpper(r); up != r {
		return mods &^ hottyShift, string(up)
	}
	if unicode.ToLower(r) != r {
		return mods &^ hottyShift, value
	}
	return mods, value
}

// isHottyChar reports whether a key's value is a character in HOTTY: one
// grapheme cluster that is not a control, the space bar's " " included.
func isHottyChar(v string) bool {
	if v == "" || clusterLen(v) != len(v) {
		return false
	}
	r, _ := utf8.DecodeRuneInString(v)
	return r != utf8.RuneError && !unicode.IsControl(r)
}

// isHottyNamed reports whether a key's value is a named key's value in
// HOTTY: a capital, then letters and digits ("Enter", "ArrowLeft", "F12").
func isHottyNamed(v string) bool {
	if len(v) < 2 || v[0] < 'A' || v[0] > 'Z' {
		return false
	}
	for i := 1; i < len(v); i++ {
		c := v[i]
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9') {
			return false
		}
	}
	return true
}

// clusterLen is the length in bytes of the grapheme cluster s starts with,
// near enough for a key: a character with the marks, joiners, variation
// selectors, skin tones and tags that follow it, a pair of regional
// indicators, or CR LF.
func clusterLen(s string) int {
	r, n := utf8.DecodeRuneInString(s)
	if r == '\r' && strings.HasPrefix(s[n:], "\n") {
		return 2
	}
	if unicode.IsControl(r) {
		return n
	}
	if isRegional(r) {
		if r2, m := utf8.DecodeRuneInString(s[n:]); isRegional(r2) {
			return n + m
		}
		return n
	}
	for n < len(s) {
		r2, m := utf8.DecodeRuneInString(s[n:])
		switch {
		case r2 == 0x200d:
			n += m
			if n < len(s) {
				_, m2 := utf8.DecodeRuneInString(s[n:])
				n += m2
			}
		case unicode.In(r2, unicode.Mn, unicode.Me, unicode.Mc), r2 >= 0xfe00 && r2 <= 0xfe0f,
			r2 >= 0x1f3fb && r2 <= 0x1f3ff, r2 >= 0xe0020 && r2 <= 0xe007f:
			n += m
		default:
			return n
		}
	}
	return n
}

// isRegional reports whether r is a regional indicator, half of a flag.
func isRegional(r rune) bool { return r >= 0x1f1e6 && r <= 0x1f1ff }
