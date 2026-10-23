package myelin

import (
	"slices"

	"github.com/neuroplastio/myelin/myelinkey"
)

// Menu is the which-key level after a sequence on the path: one item per
// key that can follow it, in the order the keymap first binds each, a group
// (more keys follow) or the binding that wins there. With no keys typed it
// is every first key; after the leader, the leader's menu.
//
// applies is the program's verdict on a binding: whether it would do
// anything now. nil means every binding applies. A group is enabled when
// anything under it is. The program draws the menu, dimming what is not
// enabled, and words a binding with no label.
//
// A group's props (Group.Props) come back on its item and on the menu
// after it, decided as its label is, so a program can ask each level of a
// sequence what it does, such as whether it stays open after a binding
// runs while the leader's modifier is held.
func (km *Keymap) Menu(seq []myelinkey.Key, p Path, applies func(Binding) bool) Menu {
	return km.menu(seq, p, applies, false)
}

// Tree is Menu with every level under the sequence: each group's items
// nested in it. After the leader it is a card of every binding under the
// leader.
func (km *Keymap) Tree(seq []myelinkey.Key, p Path, applies func(Binding) bool) Menu {
	return km.menu(seq, p, applies, true)
}

func (km *Keymap) menu(seq []myelinkey.Key, p Path, applies func(Binding) bool, tree bool) Menu {
	a := km.act(p)
	defer km.release(a)
	// own is the sequence with room for the levels under it, which items
	// writes past its end.
	own := append(make([]myelinkey.Key, 0, len(seq)+4), seq...)
	label, props := km.heading(seq, a)
	return Menu{Seq: slices.Clone(seq), Label: label, Props: props, Items: km.items(own, a, applies, tree)}
}

// items are the items after seq; with tree, each group's own nested. The
// levels under seq are listed after it in its own backing array, so seq is
// the caller's to write past its end.
func (km *Keymap) items(seq []myelinkey.Key, a *pathAct, applies func(Binding) bool, tree bool) []Item {
	es := km.next(seq, "", a)
	out := make([]Item, 0, len(es))
	for _, e := range es {
		it := Item{Key: e.key, Label: e.label, Group: e.group, Props: e.props, Binding: e.b}
		if e.group {
			under := km.items(append(seq, e.key), a, applies, tree)
			it.Enabled = slices.ContainsFunc(under, func(c Item) bool { return c.Enabled })
			if tree {
				it.Items = under
			}
		} else {
			it.Enabled = applies == nil || applies(*e.b)
		}
		out = append(out, it)
	}
	return out
}

// heading is the group label and props of a sequence, as the level before
// it lists the group; none when it is no group there.
func (km *Keymap) heading(seq []myelinkey.Key, a *pathAct) (string, map[string]string) {
	if len(seq) == 0 {
		return "", nil
	}
	for _, e := range km.next(seq[:len(seq)-1], seq[len(seq)-1], a) {
		if e.group {
			return e.label, e.props
		}
	}
	return "", nil
}
