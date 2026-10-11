package myelin

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/neuroplastio/myelin/myelinkey"
)

// winner is the precedence of the best candidate so far.
type winner struct {
	depth, rank, order int
}

// none is no candidate yet.
var none = winner{depth: -1, rank: -1, order: -1}

// beaten reports whether a candidate at w2 wins over w: deeper, then the
// higher layer, then the later bind.
func (w winner) beaten(w2 winner) bool {
	switch {
	case w2.depth != w.depth:
		return w2.depth > w.depth
	case w2.rank != w.rank:
		return w2.rank > w.rank
	}
	return w2.order > w.order
}

// at is a bound's precedence at depth d.
func (b *bound) at(d int) winner { return winner{depth: d, rank: b.rank, order: b.order} }

// floor is the depth a binding of a sequence that starts with first must
// reach on the activation's path: a key that types text is a text node's
// unless a binding at least as deep as it takes the key (the field rule).
func (a *pathAct) floor(first myelinkey.Key) int {
	if a.text >= 0 && myelinkey.TypesText(first) {
		return a.text
	}
	return 0
}

// Match looks up a sequence on a path.
func (km *Keymap) Match(seq []myelinkey.Key, p Path) Match {
	if len(seq) == 0 || !km.Starts(seq[0]) {
		return Match{}
	}
	a := km.act(p)
	defer km.release(a)
	m, _ := km.match(seq, a)
	return m
}

// match is Match on an activation of the path, with the index of the bind
// that won, an unbind included, or -1.
func (km *Keymap) match(seq []myelinkey.Key, a *pathAct) (Match, int) {
	if len(seq) == 0 {
		return Match{}, -1
	}
	floor := a.floor(seq[0])
	best, won := none, -1
	prefix := -1
	for _, i := range km.byFirst[seq[0]] {
		b := &km.binds[i]
		if !hasPrefix(b.b.Seq, seq) {
			continue
		}
		d := a.ctxDepth(b.cid, b.ctx)
		if d < floor || !km.live(b, d, a) {
			continue
		}
		if len(b.b.Seq) > len(seq) {
			if !b.b.Unbind {
				prefix = max(prefix, d)
			}
			continue
		}
		if best.beaten(b.at(d)) {
			best, won = b.at(d), i
		}
	}
	var m Match
	m.Prefix = prefix >= 0 && prefix >= best.depth
	if won >= 0 && !km.binds[won].b.Unbind {
		m.Exact = &km.binds[won].b
	}
	return m, won
}

// live reports whether a binding holding at depth d is not taken away by an
// unbind of its keys, or of a shorter sequence they continue (`unbind "g"`
// unbinds the group), that beats it there. So a group whose bindings were
// all unbound is no prefix.
func (km *Keymap) live(b *bound, d int, a *pathAct) bool {
	if b.b.Unbind {
		return true
	}
	for _, i := range km.unbinds {
		u := &km.binds[i]
		if !hasPrefix(b.b.Seq, u.b.Seq) {
			continue
		}
		if du := a.ctxDepth(u.cid, u.ctx); du >= 0 && b.at(d).beaten(u.at(du)) {
			return false
		}
	}
	return true
}

// Effective is every binding that wins its keys on the path, what typing
// them there runs, in load order. A binding a shorter one runs before, one
// that does not wait for it, is not there: where the leader key itself
// closes a dialog, nothing under the leader is effective in the dialog.
func (km *Keymap) Effective(p Path) []Binding {
	a := km.act(p)
	defer km.release(a)
	var out []Binding
	for i := range km.binds {
		if km.effective(i, a) {
			out = append(out, km.binds[i].b)
		}
	}
	return out
}

// effective reports whether bind i is a binding that wins its keys on the
// activation's path, and typing them gets to it.
func (km *Keymap) effective(i int, a *pathAct) bool {
	b := &km.binds[i]
	if b.b.Unbind || a.ctxDepth(b.cid, b.ctx) < 0 {
		return false
	}
	_, won := km.match(b.b.Seq, a)
	return won == i && km.reached(b.b.Seq, a)
}

// reached reports whether typing seq on the path gets to its last key: each
// shorter part of it waits for more.
func (km *Keymap) reached(seq []myelinkey.Key, a *pathAct) bool {
	for n := 1; n < len(seq); n++ {
		if m, _ := km.match(seq[:n], a); !m.Prefix {
			return false
		}
	}
	return true
}

// KeysFor is every sequence that runs action with exactly args on the path,
// in load order: what a hint shows for it there. nil args are the action's
// bindings with no values. The sequences are the keymap's own: read them,
// don't change them.
func (km *Keymap) KeysFor(action string, args []string, p Path) [][]myelinkey.Key {
	a := km.act(p)
	defer km.release(a)
	var out [][]myelinkey.Key
	for _, i := range km.byAction[action] {
		if b := &km.binds[i].b; slices.Equal(b.Args, args) && km.effective(i, a) {
			out = append(out, b.Seq)
		}
	}
	return out
}

// KeyFor is the binding a palette's key column shows for action on the
// path, or nil: of the bindings that run it there, one that runs it as it
// is before one that gives it values, then the shorter, then the first. It
// is the keymap's own, as a Match's is: read it, don't change it.
func (km *Keymap) KeyFor(action string, p Path) *Binding {
	a := km.act(p)
	defer km.release(a)
	var best *Binding
	for _, i := range km.byAction[action] {
		b := &km.binds[i].b
		switch {
		case best == nil:
		case (len(best.Args) == 0) != (len(b.Args) == 0):
			if len(b.Args) > 0 {
				continue
			}
		case len(b.Seq) >= len(best.Seq):
			continue
		}
		if km.effective(i, a) {
			best = b
		}
	}
	return best
}

// Explain is what a sequence does on a path, and why: every binding of the
// sequence or longer, the bindings of the sequence first, ranked by
// precedence with the winner first, then the longer ones, then those whose
// context does not hold.
func (km *Keymap) Explain(seq []myelinkey.Key, p Path) []Candidate {
	a := km.act(p)
	defer km.release(a)
	_, won := km.match(seq, a)
	type cand struct {
		i, depth int
	}
	var exact, longer, off []cand
	for i := range km.binds {
		b := &km.binds[i]
		if !hasPrefix(b.b.Seq, seq) {
			continue
		}
		c := cand{i, a.ctxDepth(b.cid, b.ctx)}
		switch {
		case c.depth < 0:
			off = append(off, c)
		case len(b.b.Seq) == len(seq):
			exact = append(exact, c)
		default:
			longer = append(longer, c)
		}
	}
	sort.SliceStable(exact, func(i, j int) bool {
		x, y := &km.binds[exact[i].i], &km.binds[exact[j].i]
		return y.at(exact[j].depth).beaten(x.at(exact[i].depth))
	})
	var out []Candidate
	for _, cs := range [][]cand{exact, longer, off} {
		for _, c := range cs {
			out = append(out, Candidate{Binding: km.binds[c.i].b, Holds: c.depth >= 0, Depth: c.depth, Wins: c.i == won})
		}
	}
	return out
}

// Shadowed is the bindings that a deeper binding of a lower layer shadows on
// any of the paths given: where both hold, the lower one's context reaches
// deeper, so the higher one never runs there. A program passes the shapes
// its path takes, such as a list, a panel and each of them under a dialog.
// Each problem is a warning at the shadowed binding's Source.
func (km *Keymap) Shadowed(paths ...Path) []Problem {
	acts := make([]*pathAct, len(paths))
	for i, p := range paths {
		acts[i] = km.act(p)
	}
	defer func() {
		for _, a := range acts {
			km.release(a)
		}
	}()
	var out []Problem
	for i := range km.binds {
		u := &km.binds[i]
		if u.b.Unbind {
			continue
		}
		for j := range km.binds {
			d := &km.binds[j]
			if d.b.Unbind || d.rank >= u.rank || !slices.Equal(d.b.Seq, u.b.Seq) {
				continue
			}
			for k, a := range acts {
				du, dd := a.ctxDepth(u.cid, u.ctx), a.ctxDepth(d.cid, d.ctx)
				if du >= 0 && dd > du {
					out = append(out, Problem{Source: u.b.Source, Warning: true, Text: fmt.Sprintf(
						"%s → %s is shadowed where %s: %s's %s → %s (%s) has a deeper context there",
						written(u.b.Keys), describe(&u.b), FormatPath(paths[k]), d.b.Layer,
						written(d.b.Keys), describe(&d.b), d.b.Source)})
					break
				}
			}
		}
	}
	return out
}

// describe is a binding's action and values.
func describe(b *Binding) string {
	if b.Unbind {
		return "(unbind)"
	}
	return strings.TrimSpace(b.Action + " " + strings.Join(b.Args, " "))
}

// entry is one key that can follow a sequence: a group (more keys follow;
// label is its group's) or a leaf (b wins there). best is the key's own
// winning binding's precedence, deeper the depth of its deepest longer one.
type entry struct {
	key    myelinkey.Key
	group  bool
	label  string
	b      *Binding
	best   winner
	deeper int
}

// next lists what can follow a sequence on the path, one entry per next
// key, in the order the keymap first binds each. With no keys typed it is
// every first key. A key only lists that key's entry alone, if any.
func (km *Keymap) next(seq []myelinkey.Key, only myelinkey.Key, a *pathAct) []entry {
	var out []entry
	at := a.keys
	clear(at)
	for i := range km.binds {
		b := &km.binds[i]
		if len(b.b.Seq) <= len(seq) || !hasPrefix(b.b.Seq, seq) {
			continue
		}
		k := b.b.Seq[len(seq)]
		if only != "" && k != only {
			continue
		}
		d := a.ctxDepth(b.cid, b.ctx)
		if d < a.floor(b.b.Seq[0]) || !km.live(b, d, a) || (b.b.Unbind && len(b.b.Seq) > len(seq)+1) {
			continue
		}
		j, seen := at[k]
		if !seen {
			j = len(out)
			at[k] = j
			out = append(out, entry{key: k, best: none, deeper: -1})
		}
		e := &out[j]
		if len(b.b.Seq) > len(seq)+1 {
			e.deeper = max(e.deeper, d)
			continue
		}
		if e.best.beaten(b.at(d)) {
			e.best, e.b = b.at(d), &b.b
		}
	}
	kept := out[:0]
	for _, e := range out {
		// A group as Match has a prefix: only where its longer bindings are
		// at least as deep as the key's own.
		e.group = e.deeper >= 0 && e.deeper >= e.best.depth
		switch {
		case e.group:
			e.b = nil
			e.label = km.groupLabel(seq, e.key, a)
		case e.b == nil || e.b.Unbind:
			continue // unbound here
		default:
			e.label = e.b.Label
		}
		kept = append(kept, e)
	}
	return kept
}

// groupLabel is the winning label of the group at exactly seq and then
// key, or "".
func (km *Keymap) groupLabel(seq []myelinkey.Key, key myelinkey.Key, a *pathAct) string {
	best := none
	label := ""
	for i := range km.groups {
		g := &km.groups[i]
		if len(g.keys) != len(seq)+1 || g.keys[len(seq)] != key || !hasPrefix(g.keys, seq) {
			continue
		}
		d := a.ctxDepth(g.cid, g.ctx)
		if d < 0 {
			continue
		}
		if w := (winner{depth: d, rank: g.rank, order: g.order}); best.beaten(w) {
			best, label = w, g.g.Label
		}
	}
	return label
}
