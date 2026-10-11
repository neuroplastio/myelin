package myelin

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/neuroplastio/myelin/myelinkey"
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

// Keymap is a built keymap: every layer's bindings, their aliases resolved,
// in load order. It is immutable and safe for concurrent use. The Bindings its
// queries return are its own: read them, don't change them.
type Keymap struct {
	env     *Env
	aliases map[string][]myelinkey.Key
	binds   []bound
	groups  []boundGroup
	// byFirst indexes binds by their first key, so a key no sequence starts
	// with costs one map lookup and builds no activation.
	byFirst map[myelinkey.Key][]int
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
	keys  []myelinkey.Key
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
	km := &Keymap{env: e, aliases: maps.Clone(e.aliases), byFirst: map[myelinkey.Key][]int{}}

	// The layers and the drops, then the aliases, before any keys.
	dropped := map[string]bool{}
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
	}
	aliasRank := map[string]int{}
	for _, d := range docs {
		rank, ok := e.ranks[d.Layer]
		if !ok || dropped[d.Layer] {
			continue
		}
		for _, name := range slices.Sorted(maps.Keys(d.Aliases)) {
			seq, err := e.readAlias(name, d.Aliases[name])
			if err != nil {
				fail(d.Source, "%v", err)
				continue
			}
			if r, set := aliasRank[name]; !set || r <= rank {
				km.aliases[name], aliasRank[name] = seq, rank
			}
		}
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
			for _, k := range keys {
				if shiftUnseen(k) {
					problems = append(problems, Problem{Source: b.Source, Warning: true, Text: fmt.Sprintf(
						"%s: Shift with a character that has no case comes in as the character it types "+
							"(shift+1 as !), except from a program that can't tell; bind that character, or a physical key",
						k)})
				}
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

// ParseSeq reads a sequence as a Bind writes it, its aliases resolved,
// into this keymap's keys: for a sequence the program meets outside the
// keymap, such as an extension's manifest, to compare with the keymap's.
func (km *Keymap) ParseSeq(s string) ([]myelinkey.Key, error) {
	var err error
	keys, ok := km.resolve(s, "", func(_, format string, args ...any) { err = fmt.Errorf(format, args...) })
	if !ok {
		return nil, err
	}
	return keys, nil
}

// CheckSeq checks a sequence as a Bind writes it, with no keymap: for a
// program that reports every mistake in its file in one pass, before it
// builds. Its keys are read as Build reads them. An alias is taken on
// trust, since a document may name it; Build reports one that none does.
func (e *Env) CheckSeq(s string) error {
	words := myelinkey.SplitSeq(s)
	if len(words) == 0 {
		return errors.New("an empty key sequence")
	}
	for _, w := range words {
		if name, ok := aliasOf(w); ok {
			if !isAliasName(name) {
				return fmt.Errorf("%s: %s", w, aliasNameRule)
			}
			continue
		}
		if _, err := myelinkey.ParseWith(w, e.keys); err != nil {
			return err
		}
	}
	return nil
}

// resolve reads a sequence as written, each alias's keys put in for it.
func (km *Keymap) resolve(s, src string, fail func(src, format string, args ...any)) ([]myelinkey.Key, bool) {
	words := myelinkey.SplitSeq(s)
	if len(words) == 0 {
		fail(src, "an empty key sequence")
		return nil, false
	}
	out := make([]myelinkey.Key, 0, len(words))
	for _, w := range words {
		if name, ok := aliasOf(w); ok {
			seq, ok := km.aliases[name]
			if !ok {
				fail(src, "%q: no alias %s", strings.Join(words, " "), w)
				return nil, false
			}
			out = append(out, seq...)
			continue
		}
		k, err := myelinkey.ParseWith(w, km.env.keys)
		if err != nil {
			fail(src, "%v", err)
			return nil, false
		}
		out = append(out, k)
	}
	return out, true
}

// aliasOf reads a word of a sequence written as an alias, "<leader>", into
// its name.
func aliasOf(word string) (string, bool) {
	if len(word) < 3 || word[0] != '<' || word[len(word)-1] != '>' {
		return "", false
	}
	return word[1 : len(word)-1], true
}

// aliasNameRule is what isAliasName checks, for the errors.
const aliasNameRule = "an alias's name is letters, digits, _ and -"

// isAliasName reports whether s can name an alias: ASCII letters, digits,
// _ and -.
func isAliasName(s string) bool {
	for _, c := range []byte(s) {
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return s != ""
}

// readAlias reads an alias's keys the way the schema reads keys.
func (e *Env) readAlias(name, keys string) ([]myelinkey.Key, error) {
	if !isAliasName(name) {
		return nil, fmt.Errorf("alias %q: %s", name, aliasNameRule)
	}
	words := myelinkey.SplitSeq(keys)
	if len(words) == 0 {
		return nil, fmt.Errorf("alias <%s> has no keys", name)
	}
	seq := make([]myelinkey.Key, len(words))
	for i, w := range words {
		if _, ok := aliasOf(w); ok {
			return nil, fmt.Errorf("alias <%s>: %s is an alias, and an alias's keys name none", name, w)
		}
		k, err := myelinkey.ParseWith(w, e.keys)
		if err != nil {
			return nil, fmt.Errorf("alias <%s>: %w", name, err)
		}
		seq[i] = k
	}
	return seq, nil
}

// shiftUnseen reports whether a key is Shift with a character that has no
// case, which a program sends as the character Shift types: shift+1 as !.
func shiftUnseen(k myelinkey.Key) bool {
	mods, value, char := myelinkey.Split(k)
	return char && mods&myelinkey.Shift != 0 && strings.ToUpper(value) == value
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
				written(u.b.Keys), written(b.b.Keys), b.b.Action, b.b.When, b.b.Source)})
		}
	}
	return out
}

// written is a sequence as its Bind writes it, its words separated by one
// space: what a message quotes, so that it reads as the user's file does.
func written(keys string) string { return strings.Join(myelinkey.SplitSeq(keys), " ") }

// hasPrefix reports whether seq begins with prefix.
func hasPrefix(seq, prefix []myelinkey.Key) bool {
	return len(seq) >= len(prefix) && slices.Equal(seq[:len(prefix)], prefix)
}

// Alias is the keys an alias stands for, by its name ("leader"): the
// highest layer's, else the schema's. false when none names it. The keys
// are the keymap's own: read them, don't change them.
func (km *Keymap) Alias(name string) ([]myelinkey.Key, bool) {
	seq, ok := km.aliases[name]
	return seq, ok
}

// All is every binding and unbind, its aliases resolved, in load order:
// what a listing prints.
func (km *Keymap) All() []Binding {
	out := make([]Binding, len(km.binds))
	for i := range km.binds {
		out[i] = km.binds[i].b
	}
	return out
}

// Starts reports whether any binding, in any context, begins with k.
func (km *Keymap) Starts(k myelinkey.Key) bool { return len(km.byFirst[k]) > 0 }
