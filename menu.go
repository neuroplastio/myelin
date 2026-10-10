package myelin

import (
	"slices"

	"github.com/neuroplastio/myelin/key"
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
func (km *Keymap) Menu(seq []key.Key, p Path, applies func(Binding) bool) Menu {
	a := km.env.act(p)
	return Menu{Seq: slices.Clone(seq), Label: km.label(seq, p, a), Items: km.items(seq, p, a, applies, false)}
}

// Tree is Menu with every level under the sequence: each group's items
// nested in it. After the leader it is a card of every binding under the
// leader.
func (km *Keymap) Tree(seq []key.Key, p Path, applies func(Binding) bool) Menu {
	a := km.env.act(p)
	return Menu{Seq: slices.Clone(seq), Label: km.label(seq, p, a), Items: km.items(seq, p, a, applies, true)}
}

// items are the items after seq; with tree, each group's own nested.
func (km *Keymap) items(seq []key.Key, p Path, a *pathAct, applies func(Binding) bool, tree bool) []Item {
	es := km.next(seq, p, a)
	out := make([]Item, 0, len(es))
	for _, e := range es {
		it := Item{Key: e.key, Label: e.label, Group: e.group, Binding: e.b}
		if e.group {
			under := km.items(append(slices.Clone(seq), e.key), p, a, applies, tree)
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

// label is the group label of a sequence, as the level before it lists the
// group; "" when it is no group there.
func (km *Keymap) label(seq []key.Key, p Path, a *pathAct) string {
	if len(seq) == 0 {
		return ""
	}
	last := seq[len(seq)-1]
	for _, e := range km.next(seq[:len(seq)-1], p, a) {
		if e.key == last && e.group {
			return e.label
		}
	}
	return ""
}
