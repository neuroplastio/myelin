package key

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ParseHotty reads a key's name as the HOTTY protocol writes it, its
// modifiers in any order ("Shift+Control+a", "Control+ "), and returns its
// canonical name ("Control+A", "Control+Space"): the modifiers in the order
// Control, Alt, Meta, Shift; Shift shown in a letter where it can be; Space
// for a space. ok is false for a name that does not parse.
//
// This is how a key comes in from a HOTTY host, from a terminal's bytes or
// from a UI toolkit's key event, once something has named it.
func ParseHotty(name string) (Key, bool) {
	mods, value, ok := splitName(name)
	if !ok {
		return "", false
	}
	mods, value = canonical(mods, value)
	if value == " " {
		value = "Space"
	}
	return Key(join(mods, value)), true
}

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
			for i, known := range modNames {
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
	if !isChar(value) && !isNamed(value) {
		return 0, "", false
	}
	return mods, value, true
}

// canonical shows Shift in a letter where it can: Shift with a small letter
// is its capital, and Shift with a capital is the capital.
func canonical(mods int, value string) (int, string) {
	if mods&modShift == 0 || !isChar(value) {
		return mods, value
	}
	r, n := utf8.DecodeRuneInString(value)
	if n != len(value) {
		return mods, value
	}
	if up := unicode.ToUpper(r); up != r {
		return mods &^ modShift, string(up)
	}
	if unicode.ToLower(r) != r {
		return mods &^ modShift, value
	}
	return mods, value
}

// isChar reports whether a key's value is a character: one grapheme
// cluster that is not a control.
func isChar(v string) bool {
	if v == "" || clusterLen(v) != len(v) {
		return false
	}
	r, _ := utf8.DecodeRuneInString(v)
	return r != utf8.RuneError && !unicode.IsControl(r)
}

// isNamed reports whether a key's value is a named key's value: a capital,
// then letters and digits ("Enter", "ArrowLeft", "F12").
func isNamed(v string) bool {
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
