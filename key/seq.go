package key

import (
	"errors"
	"strings"
)

// ParseSeq reads a sequence of keys in the short spelling, separated by
// ASCII white space: "g g", "ctrl+x d". Other white space, such as a
// no-break space, is a key. "<leader>" is refused, as [Parse] refuses it.
func ParseSeq(s string) ([]Key, error) {
	fields := SplitSeq(s)
	if len(fields) == 0 {
		return nil, errors.New("an empty key sequence")
	}
	out := make([]Key, len(fields))
	for i, f := range fields {
		k, err := Parse(f)
		if err != nil {
			return nil, err
		}
		out[i] = k
	}
	return out, nil
}

// SplitSeq splits a written sequence into its keys, still as written, at
// ASCII white space, as HOTTY splits a data-keys value. A program that gives
// a word of its own a meaning in a sequence, as a keymap gives "<leader>",
// reads the words here and parses the rest with [Parse].
func SplitSeq(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return strings.ContainsRune(" \t\n\f\r", r) })
}

// FormatSeq writes a sequence the way ParseSeq reads it: "g g".
func FormatSeq(seq []Key) string {
	parts := make([]string, len(seq))
	for i, k := range seq {
		parts[i] = Format(k)
	}
	return strings.Join(parts, " ")
}
