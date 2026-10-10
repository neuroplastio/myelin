package myelin

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/neuroplastio/myelin/key"
)

// What wins, for bindings of the same keys whose contexts hold on the path:
//
//  1. the deepest context (Context.Depth: how far into the path it reaches);
//  2. at equal depth, the higher layer (the schema's order);
//  3. within a layer, the later bind (load order).
//
// A binding that is also a prefix of a longer one waits only for longer
// bindings at least as deep as itself (Match.Prefix), and runs when the
// sequence times out. With a node that takes text on the path, a sequence
// that starts with a key that types text is the field's unless a binding at
// least as deep as that node takes it (the field rule).

// Keymap is a built keymap: every layer's bindings, the leader resolved, in
// load order. It is immutable and safe for concurrent use. The Bindings its
// queries return are its own: read them, don't change them.
type Keymap struct {
	env       *Env
	leader    key.Key
	hasLeader bool
	binds     []bound
	groups    []boundGroup
	// byFirst indexes binds by their first key, so a key no sequence starts
	// with costs one map lookup and builds no activation.
	byFirst map[key.Key][]int
	// unbinds indexes the unbinds in binds.
	unbinds []int
}

// bound is a binding as the keymap holds it.
type bound struct {
	b     Binding
	ctx   *Context
	rank  int
	order int
}

// boundGroup is a group as the keymap holds it.
type boundGroup struct {
	g     Group
	keys  []key.Key
	ctx   *Context
	rank  int
	order int
}

// warning marks an error from a program's check as a warning (Warn).
type warning struct{ err error }

func (w warning) Error() string { return w.err.Error() }
func (w warning) Unwrap() error { return w.err }

// Warn makes err a warning: Build loads the bind and reports it.
func Warn(err error) error {
	if err == nil {
		return nil
	}
	return warning{err}
}

// Build resolves documents into one keymap, in the order given: the order
// All and the menus list bindings in. Precedence is the layer's, not the
// position's.
//
// check is the program's rules for a binding, its sequence resolved and its
// layer named: whether its action exists and takes its values, or which
// keys a layer may bind where. An error refuses the keymap and Warn(err) loads it with a
// warning. nil checks nothing. Every problem carries the Source of what it
// is about. With any error the Keymap is nil; warnings load.
func (e *Env) Build(docs []Doc, check func(Binding) error) (*Keymap, []Problem) {
	var problems []Problem
	fail := func(src, format string, args ...any) {
		problems = append(problems, Problem{Source: src, Text: fmt.Sprintf(format, args...)})
	}
	km := &Keymap{env: e, leader: e.leader, hasLeader: e.hasLeader, byFirst: map[key.Key][]int{}}

	// The layers, the leader and the drops, before any keys.
	dropped := map[string]bool{}
	leaderRank := -1
	for _, d := range docs {
		rank, ok := e.ranks[d.Layer]
		if !ok {
			fail(d.Source, "layer %q is not one of %s", d.Layer, strings.Join(e.schema.Layers, ", "))
			continue
		}
		for _, l := range d.Drops {
			switch r, ok := e.ranks[l]; {
			case !ok:
				fail(d.Source, "drops layer %q, which is not one of %s", l, strings.Join(e.schema.Layers, ", "))
			case r >= rank:
				fail(d.Source, "layer %s drops %s, which is not below it", d.Layer, l)
			default:
				dropped[l] = true
			}
		}
		if d.Leader == "" || rank < leaderRank {
			continue
		}
		k, err := key.Parse(d.Leader)
		if err != nil {
			fail(d.Source, "leader: %v", err)
			continue
		}
		km.leader, km.hasLeader, leaderRank = k, true, rank
	}

	ctxs := map[string]*Context{}
	compile := func(src, when string) (*Context, bool) {
		if c, ok := ctxs[when]; ok {
			return c, true
		}
		c, err := e.Compile(when)
		if err != nil {
			fail(src, "%v", err)
			return nil, false
		}
		ctxs[when] = c
		return c, true
	}
	order := 0
	for _, d := range docs {
		rank, ok := e.ranks[d.Layer]
		if !ok || dropped[d.Layer] {
			continue
		}
		for _, b := range d.Binds {
			switch {
			case b.Unbind && (b.Action != "" || len(b.Args) > 0):
				fail(b.Source, "unbind %q names an action; an unbind binds its keys to nothing", b.Keys)
				continue
			case !b.Unbind && b.Action == "":
				fail(b.Source, "bind %q names no action", b.Keys)
				continue
			}
			keys, ok := km.resolve(b.Keys, b.Source, fail)
			ctx, cok := compile(b.Source, b.When)
			if !ok || !cok {
				continue
			}
			bd := Binding{Bind: b, Seq: keys, Layer: d.Layer}
			if check != nil && !b.Unbind {
				if err := check(bd); err != nil {
					var w warning
					problems = append(problems, Problem{Source: b.Source, Text: err.Error(), Warning: errors.As(err, &w)})
					if !errors.As(err, &w) {
						continue
					}
				}
			}
			km.byFirst[keys[0]] = append(km.byFirst[keys[0]], len(km.binds))
			if b.Unbind {
				km.unbinds = append(km.unbinds, len(km.binds))
			}
			km.binds = append(km.binds, bound{b: bd, ctx: ctx, rank: rank, order: order})
			order++
		}
		for _, g := range d.Groups {
			keys, ok := km.resolve(g.Keys, g.Source, fail)
			ctx, cok := compile(g.Source, g.When)
			if !ok || !cok {
				continue
			}
			km.groups = append(km.groups, boundGroup{g: g, keys: keys, ctx: ctx, rank: rank, order: order})
			order++
		}
	}
	problems = append(problems, km.shallowUnbinds()...)
	for _, p := range problems {
		if !p.Warning {
			return nil, problems
		}
	}
	return km, problems
}

// ParseSeq reads a sequence as a Bind writes it, <leader> included, into
// this keymap's keys: for a sequence the program meets outside the keymap,
// such as an extension's manifest, to compare with the keymap's.
func (km *Keymap) ParseSeq(s string) ([]key.Key, error) {
	var err error
	keys, ok := km.resolve(s, "", func(_, format string, args ...any) { err = fmt.Errorf(format, args...) })
	if !ok {
		return nil, err
	}
	return keys, nil
}

// leaderToken is how a sequence writes the leader alias.
const leaderToken = "<leader>"

// CheckSeq checks a sequence as a Bind writes it, <leader> included, with
// no keymap: for a program that reports every mistake in its file in one
// pass, before it builds. Build reads the sequence the same way.
func CheckSeq(s string) error {
	fields := key.SplitSeq(s)
	if len(fields) == 0 {
		return errors.New("an empty key sequence")
	}
	for _, f := range fields {
		if f == leaderToken {
			continue
		}
		if _, err := key.Parse(f); err != nil {
			return err
		}
	}
	return nil
}

// resolve reads a sequence as written, the leader put in for <leader>.
func (km *Keymap) resolve(s, src string, fail func(src, format string, args ...any)) ([]key.Key, bool) {
	fields := key.SplitSeq(s)
	if len(fields) == 0 {
		fail(src, "an empty key sequence")
		return nil, false
	}
	out := make([]key.Key, len(fields))
	for i, f := range fields {
		if f == leaderToken {
			if !km.hasLeader {
				fail(src, "%s names %s, and no leader is set", strings.Join(fields, " "), leaderToken)
				return nil, false
			}
			out[i] = km.leader
			continue
		}
		k, err := key.Parse(f)
		if err != nil {
			fail(src, "%v", err)
			return nil, false
		}
		out[i] = k
	}
	return out, true
}

// shallowUnbinds warns of an unbind written with no context whose keys a
// lower layer binds in one: that binding reaches deeper wherever it holds,
// so it wins there, and the unbind does not take the keys away. The warning
// names the context to unbind them in.
func (km *Keymap) shallowUnbinds() []Problem {
	var out []Problem
	for _, i := range km.unbinds {
		u := &km.binds[i]
		if u.ctx != nil {
			continue
		}
		for j := range km.binds {
			b := &km.binds[j]
			if b.b.Unbind || b.ctx == nil || b.rank >= u.rank || !hasPrefix(b.b.Seq, u.b.Seq) {
				continue
			}
			out = append(out, Problem{Source: u.b.Source, Warning: true, Text: fmt.Sprintf(
				"unbind %q leaves %s → %s in context %q (%s), which wins there; unbind it in that context",
				key.FormatSeq(u.b.Seq), key.FormatSeq(b.b.Seq), b.b.Action, b.b.When, b.b.Source)})
		}
	}
	return out
}

// hasPrefix reports whether seq begins with prefix.
func hasPrefix(seq, prefix []key.Key) bool {
	return len(seq) >= len(prefix) && slices.Equal(seq[:len(prefix)], prefix)
}

// Leader is the keymap's leader key: the highest layer's that names one,
// else the schema's. false when neither does.
func (km *Keymap) Leader() (key.Key, bool) { return km.leader, km.hasLeader }

// All is every binding and unbind, the leader resolved, in load order: what
// a listing prints.
func (km *Keymap) All() []Binding {
	out := make([]Binding, len(km.binds))
	for i := range km.binds {
		out[i] = km.binds[i].b
	}
	return out
}

// Starts reports whether any binding, in any context, begins with k.
func (km *Keymap) Starts(k key.Key) bool { return len(km.byFirst[k]) > 0 }
