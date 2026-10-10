package myelintea

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/neuroplastio/myelin"
)

// Bubble Tea's keys come out as the keys a keymap writes short.
func TestKey(t *testing.T) {
	for _, tc := range []struct {
		name  string
		k     tea.Key
		short string
	}{
		{"a character", tea.Key{Code: 'j', Text: "j"}, "j"},
		{"shift with a letter is its capital", tea.Key{Code: 'h', ShiftedCode: 'H', Mod: tea.ModShift, Text: "H"}, "H"},
		{"ctrl", tea.Key{Code: 'd', Mod: tea.ModCtrl}, "ctrl+d"},
		{"ctrl+shift with a letter", tea.Key{Code: 'a', Mod: tea.ModCtrl | tea.ModShift}, "ctrl+shift+a"},
		{"alt+ctrl in HOTTY's order", tea.Key{Code: 'x', Mod: tea.ModAlt | tea.ModCtrl}, "ctrl+alt+x"},
		{"super is meta", tea.Key{Code: 'k', Mod: tea.ModSuper}, "meta+k"},
		{"a shifted character", tea.Key{Code: '1', ShiftedCode: '!', Mod: tea.ModShift, Text: "!"}, "!"},
		{"shift with no shifted key", tea.Key{Code: '1', Mod: tea.ModShift}, "shift+1"},
		{"the space bar", tea.Key{Code: tea.KeySpace, Text: " "}, "space"},
		{"ctrl+space", tea.Key{Code: tea.KeySpace, Mod: tea.ModCtrl}, "ctrl+space"},
		{"enter", tea.Key{Code: tea.KeyEnter}, "enter"},
		{"esc", tea.Key{Code: tea.KeyEscape}, "esc"},
		{"shift+tab", tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}, "shift+tab"},
		{"an arrow", tea.Key{Code: tea.KeyLeft, Mod: tea.ModCtrl}, "ctrl+left"},
		{"page down", tea.Key{Code: tea.KeyPgDown}, "pgdn"},
		{"a function key", tea.Key{Code: tea.KeyF5}, "f5"},
		{"a modifier alone", tea.Key{Code: tea.KeyLeftShift}, "shift"},
	} {
		got, ok := Key(tc.k)
		if !ok {
			t.Errorf("%s: Key(%+v) has no name", tc.name, tc.k)
			continue
		}
		want, err := myelin.ParseKey(tc.short)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got != want {
			t.Errorf("%s: Key(%+v) = %q, want %q (%s)", tc.name, tc.k, got, want, tc.short)
		}
	}
}

// A key Bubble Tea has no SPEC §10.4 name for is no key.
func TestKeyWithNoName(t *testing.T) {
	if k, ok := Key(tea.Key{Code: tea.KeyCapsLock}); ok {
		t.Errorf("caps lock is %q", k)
	}
}
