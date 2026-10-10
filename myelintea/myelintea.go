// Package myelintea reads Bubble Tea v2's keys as myelin's: a program built
// on Bubble Tea feeds its key presses to a myelin keymap through Key.
//
//	case tea.KeyPressMsg:
//		if k, ok := myelintea.Key(msg.Key()); ok {
//			st, res = myelin.Feed(km, st, k, path, time.Now())
//		}
//
// It is a module of its own, so that Bubble Tea stays out of myelin's.
package myelintea

import (
	tea "charm.land/bubbletea/v2"

	"github.com/neuroplastio/hotty-go/hottytea"
	"github.com/neuroplastio/myelin"
)

// Key is a Bubble Tea key as myelin compares it: its HOTTY name, as
// hottytea.KeyName gives it, so the key is the one a HOTTY host or a
// terminal's bytes give for the same keystroke. ok is false for a key with
// no name there.
func Key(k tea.Key) (myelin.Key, bool) {
	name := hottytea.KeyName(k)
	if name == "" {
		return "", false
	}
	return myelin.ParseHottyKey(name)
}
