package myelin

import (
	"maps"
	"slices"
)

// Mount is a document for a widget another program embeds, such as a
// pager, put under the node its host shows it at: a copy of d with every
// bind's and group's context joined to when, as "(when) && (its own)". A
// node with no context of its own gets when. An empty when changes nothing.
func Mount(d Doc, when string) Doc {
	out := d
	out.Drops = slices.Clone(d.Drops)
	out.Binds = slices.Clone(d.Binds)
	for i := range out.Binds {
		b := &out.Binds[i]
		b.When = joinWhen(when, b.When)
		b.Args = slices.Clone(b.Args)
		b.Props = maps.Clone(b.Props)
	}
	out.Groups = slices.Clone(d.Groups)
	for i := range out.Groups {
		out.Groups[i].When = joinWhen(when, out.Groups[i].When)
	}
	return out
}

// joinWhen is a context that holds where both outer and inner hold.
func joinWhen(outer, inner string) string {
	switch {
	case outer == "":
		return inner
	case inner == "":
		return outer
	}
	return "(" + outer + ") && (" + inner + ")"
}
