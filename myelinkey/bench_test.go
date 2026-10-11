package myelinkey

import "testing"

// Reading a key is the path of every live key a program names as text, and
// of every key in a keymap at load. A canonical name costs nothing to read.

var sink Key

func benchParse(b *testing.B, s string, o Options) {
	b.ReportAllocs()
	for b.Loop() {
		k, err := ParseWith(s, o)
		if err != nil {
			b.Fatal(err)
		}
		sink = k
	}
}

func BenchmarkParse(b *testing.B) {
	for _, tc := range []struct {
		name, in string
		o        Options
	}{
		{"char", "a", Options{}},
		{"canonical", "ctrl+shift+a", Options{}},
		{"physical", "ctrl+page_down", Options{}},
		{"lenient", "Shift+Ctrl+A", Options{}},
		{"w3c", "ctrl+PageDown", Options{}},
		{"strict", "super+ctrl+alt+shift+enter", Options{Strict: true}},
		{"shifted_capital", "G", Options{ShiftedCapitals: true}},
	} {
		b.Run(tc.name, func(b *testing.B) { benchParse(b, tc.in, tc.o) })
	}
}

func BenchmarkParseHotty(b *testing.B) {
	for _, name := range []string{"a", "Control+a", "Shift+Control+a", "PageDown", "Meta+Enter"} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				k, ok := ParseHotty(name)
				if !ok {
					b.Fatal(name)
				}
				sink = k
			}
		})
	}
}

func BenchmarkSplit(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, v, _ := Split("super+ctrl+alt+shift+page_down"); v != "page_down" {
			b.Fatal(v)
		}
	}
}

func BenchmarkDisplay(b *testing.B) {
	for _, k := range []Key{"ctrl+shift+a", "ctrl+arrow_left", "shift+tab"} {
		b.Run(string(k), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = Display(k)
			}
		})
	}
}

func BenchmarkParseSeq(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := ParseSeq("ctrl+x shift+g page_down"); err != nil {
			b.Fatal(err)
		}
	}
}
