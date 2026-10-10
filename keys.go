package myelin

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	hotty "github.com/neuroplastio/hotty-go"
)

// The short spelling is how keys are written in a keymap: modifiers
// (ctrl+, alt+, meta+, shift+) and then one character or a named key:
// "ctrl+d", "pgdn", "shift+tab", "ctrl+shift+a", "H". A sequence is keys
// separated by spaces: "g g", "<leader> p n".
//
// Every key HOTTY names has exactly one spelling, which FormatKey writes and
// ParseKey reads, and ParseKey reads no other. Modifiers come
// in HOTTY's order; modifiers and short names are lower case. A named key
// with no short name is written as HOTTY names it: "ContextMenu".

// shortNames are the named keys the short spelling writes in its own way,
// and their HOTTY names. The function keys are spelled by rule: f1 is F1,
// f13 is F13 (fkeyName).
var shortNames = map[string]string{
	"up": "ArrowUp", "down": "ArrowDown", "left": "ArrowLeft", "right": "ArrowRight",
	"enter": "Enter", "esc": "Escape", "tab": "Tab", "space": "Space",
	"home": "Home", "end": "End", "pgup": "PageUp", "pgdn": "PageDown",
	"backspace": "Backspace", "delete": "Delete", "insert": "Insert",
	// A modifier alone is that modifier's key.
	"ctrl": "Control", "alt": "Alt", "meta": "Meta", "shift": "Shift",
}

// hottyNames are shortNames turned around.
var hottyNames = func() map[string]string {
	m := make(map[string]string, len(shortNames))
	for short, name := range shortNames {
		m[name] = short
	}
	return m
}()

// The modifiers, in the order HOTTY writes them, and so the
// order FormatKey writes them: their HOTTY names and their short spelling.
var (
	modNames  = [...]string{"Control", "Alt", "Meta", "Shift"}
	modShorts = [...]string{"ctrl", "alt", "meta", "shift"}
)

const (
	modControl = 1 << iota
	modAlt
	modMeta
	modShift
)

// leaderToken is how a sequence writes the leader alias.
const leaderToken = "<leader>"

// nameOf is the HOTTY name of a named key written short: "pgdn" is
// "PageDown", "f13" is "F13".
func nameOf(short string) (string, bool) {
	if name, ok := shortNames[short]; ok {
		return name, true
	}
	if n, ok := fkey(short, 'f'); ok {
		return "F" + n, true
	}
	return "", false
}

// shortOf is how the short spelling writes a HOTTY named key that it
// writes in its own way: "PageDown" is "pgdn", "F13" is "f13".
func shortOf(name string) (string, bool) {
	if short, ok := hottyNames[name]; ok {
		return short, true
	}
	if n, ok := fkey(name, 'F'); ok {
		return "f" + n, true
	}
	return "", false
}

// fkey reads a function key's number after its letter: a decimal from 1,
// with no leading zero.
func fkey(s string, letter byte) (string, bool) {
	if len(s) < 2 || s[0] != letter || s[1] == '0' {
		return "", false
	}
	if _, err := strconv.ParseUint(s[1:], 10, 16); err != nil {
		return "", false
	}
	return s[1:], true
}

// ParseKey reads one key in the short spelling and returns its canonical
// HOTTY name: "ctrl+d" is "Control+d", "pgdn" is "PageDown", "H" is "H",
// "ctrl+shift+a" is "Control+A", "shift+1" is "Shift+1".
//
// It reads only the spelling [FormatKey] writes. Any other way of writing a
// key is refused with that spelling: "shift+h" asks for "H", "ctrl+D" for
// "ctrl+shift+d", "alt+ctrl+x" for "ctrl+alt+x", "Escape" for "esc".
// "<leader>" is an error here: it is the keymap's to resolve
// ([Keymap.ParseSeq], [CheckSeq]).
func ParseKey(s string) (Key, error) {
	var mods int
	rest := s
	for {
		i := strings.IndexByte(rest, '+')
		if i <= 0 || i == len(rest)-1 {
			break
		}
		m := modIndex(rest[:i])
		if m < 0 {
			if m = modIndexFold(rest[:i]); m >= 0 {
				return "", fmt.Errorf("%q: modifiers are written in lower case: %s+", s, modShorts[m])
			}
			break
		}
		if mods&(1<<m) != 0 {
			return "", fmt.Errorf("%q: %s+ twice", s, modShorts[m])
		}
		mods |= 1 << m
		rest = rest[i+1:]
	}
	var value string
	switch name, short := nameOf(rest); {
	case rest == "":
		return "", errors.New("an empty key")
	case rest == leaderToken:
		return "", fmt.Errorf("%q: %s is the keymap's to resolve", s, leaderToken)
	case short:
		value = name
	case strings.Contains(rest, "+") && rest != "+":
		return "", fmt.Errorf("%q is not a key (modifiers are ctrl+, alt+, meta+ and shift+)", s)
	case looksNamed(rest):
		// A short name in another case ("ESC", "Up") is HOTTY's to read as a
		// name of its own, which no host sends: ask for the short name.
		if _, ok := nameOf(strings.ToLower(rest)); ok {
			return "", fmt.Errorf("%q: write %s", s, s[:len(s)-len(rest)]+strings.ToLower(rest))
		}
		if _, ok := hotty.ParseKey(rest); !ok {
			return "", fmt.Errorf("%q is not a key (named keys are written short: esc, enter, pgdn, f1; a sequence's keys are separated by spaces: `g g`)", s)
		}
		value = rest
	default:
		if _, ok := hotty.ParseKey(rest); !ok {
			return "", fmt.Errorf("%q is not a key", s)
		}
		value = rest
	}
	k, ok := hotty.ParseKey(hottyName(mods, value))
	if !ok {
		return "", fmt.Errorf("%q is not a key", s)
	}
	if w := FormatKey(Key(k)); w != s {
		return "", fmt.Errorf("%q: write %s", s, w)
	}
	return Key(k), nil
}

// hottyName writes modifiers and a value as HOTTY does.
func hottyName(mods int, value string) string {
	var b strings.Builder
	for i, m := range modNames {
		if mods&(1<<i) != 0 {
			b.WriteString(m)
			b.WriteByte('+')
		}
	}
	b.WriteString(value)
	return b.String()
}

// modIndex is the index of a modifier written short, or -1.
func modIndex(s string) int {
	for i, m := range modShorts {
		if s == m {
			return i
		}
	}
	return -1
}

// modIndexFold is modIndex in any case, for the error that asks for lower
// case.
func modIndexFold(s string) int {
	for i, m := range modShorts {
		if strings.EqualFold(s, m) {
			return i
		}
	}
	return -1
}

// looksNamed reports whether s is two or more ASCII letters and digits. No
// character is, so it can only be a key's name: HOTTY's ("PageDown") or the
// short spelling's ("pgdn").
func looksNamed(s string) bool {
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

// ParseHottyKey reads a key's name as the HOTTY protocol writes it, its
// modifiers in any order, from a HOTTY host, a terminal (hotty-go's
// DecodeKeys) or Bubble Tea (hottytea.KeyName), and returns it canonical:
// "Shift+Control+a" is "Control+A". ok is false for a name that does not
// parse.
func ParseHottyKey(name string) (Key, bool) {
	k, ok := hotty.ParseKey(name)
	return Key(k), ok
}

// split reads a canonical key into its modifiers and its value, as HOTTY
// writes them: "Control++" is Control and "+".
func split(k Key) (mods int, value string) {
	s := string(k)
	for {
		i := strings.IndexByte(s, '+')
		if i <= 0 || i == len(s)-1 {
			return mods, s
		}
		m := -1
		for j, name := range modNames {
			if s[:i] == name {
				m = j
			}
		}
		if m < 0 {
			return mods, s
		}
		mods |= 1 << m
		s = s[i+1:]
	}
}

// spell reads a key as the short spelling writes it: its modifiers, with
// shift written out for a capital that has others ("Control+A" is ctrl,
// shift and "a"), and its value written short.
func spell(k Key) (mods int, value string) {
	mods, value = split(k)
	if mods != 0 && isCapital(value) {
		mods |= modShift
		value = string(unicode.ToLower([]rune(value)[0]))
	}
	if short, ok := shortOf(value); ok {
		value = short
	}
	return mods, value
}

// FormatKey writes a key in the short spelling, which ParseKey reads back:
// "Control+d" is "ctrl+d", "PageDown" is "pgdn", "Control+A" is
// "ctrl+shift+a", "Shift+1" is "shift+1", "ContextMenu" is itself.
func FormatKey(k Key) string {
	mods, value := spell(k)
	var b strings.Builder
	for i, m := range modShorts {
		if mods&(1<<i) != 0 {
			b.WriteString(m)
			b.WriteByte('+')
		}
	}
	b.WriteString(value)
	return b.String()
}

// arrowGlyphs are how menus show the arrows, by their short names.
var arrowGlyphs = map[string]string{"left": "←", "right": "→", "up": "↑", "down": "↓"}

// DisplayKey writes a key as a menu shows it: the short spelling, with the
// arrows and Tab as glyphs ("ctrl+←", "⇥"). Shift+Tab is "⇧⇥", one glyph
// pair a reader knows, where "shift+⇥" reads as a chord.
func DisplayKey(k Key) string {
	mods, value := spell(k)
	var b strings.Builder
	for i, m := range modShorts[:3] {
		if mods&(1<<i) != 0 {
			b.WriteString(m)
			b.WriteByte('+')
		}
	}
	shift := mods&modShift != 0
	switch g, arrow := arrowGlyphs[value]; {
	case value == "tab" && shift:
		b.WriteString("⇧⇥")
	case value == "tab":
		b.WriteString("⇥")
	case arrow:
		if shift {
			b.WriteString("shift+")
		}
		b.WriteString(g)
	default:
		if shift {
			b.WriteString("shift+")
		}
		b.WriteString(value)
	}
	return b.String()
}

// ParseSeq reads a sequence of keys in the short spelling, separated by
// ASCII white space: "g g", "ctrl+x d". Other white space, such as a
// no-break space, is a key. It has no <leader>: that is the keymap's to
// resolve.
func ParseSeq(s string) ([]Key, error) {
	fields := seqFields(s)
	if len(fields) == 0 {
		return nil, errors.New("an empty key sequence")
	}
	out := make([]Key, len(fields))
	for i, f := range fields {
		k, err := ParseKey(f)
		if err != nil {
			return nil, err
		}
		out[i] = k
	}
	return out, nil
}

// CheckSeq checks a sequence as a Bind writes it, <leader> included, with
// no keymap: for a program that reports every mistake in its file in one
// pass, before it builds. Build reads the sequence the same way.
func CheckSeq(s string) error {
	fields := seqFields(s)
	if len(fields) == 0 {
		return errors.New("an empty key sequence")
	}
	for _, f := range fields {
		if f == leaderToken {
			continue
		}
		if _, err := ParseKey(f); err != nil {
			return err
		}
	}
	return nil
}

// seqFields splits a sequence at ASCII white space, as HOTTY splits a
// data-keys value (SPEC §10.2).
func seqFields(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return strings.ContainsRune(" \t\n\f\r", r) })
}

// FormatSeq writes a sequence the way ParseSeq reads it: "g g".
func FormatSeq(seq []Key) string {
	parts := make([]string, len(seq))
	for i, k := range seq {
		parts[i] = FormatKey(k)
	}
	return strings.Join(parts, " ")
}

// TypesText reports whether a key types text into a field: a character or
// Space, with no ctrl, alt or meta. Such a key goes to a node that takes
// text ([NodeType].Text) unless a binding at least as deep takes it.
func TypesText(k Key) bool {
	mods, value := split(k)
	if mods&(modControl|modAlt|modMeta) != 0 {
		return false
	}
	return value == "Space" || value != "" && !looksNamed(value)
}
