package myelin

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	cellite "neuroplast.io/cel-lite"

	"neuroplast.io/myelin/myelinkey"
)

// focusVar is the variable that names the innermost node on the path.
const focusVar = "focus"

// Env is the CEL environment a schema declares: the handle every context is
// compiled in, and every keymap built with. It is safe for concurrent use.
type Env struct {
	schema Schema
	cel    *cellite.Env
	// nodes are the declared node types by name; vars the variables a
	// context reads, by their index in cel.
	nodes map[string]*nodeInfo
	vars  []varInfo
	// ranks are the layers' ranks, lowest 0.
	ranks map[string]int
	// keys is how the schema reads keys.
	keys myelinkey.Options
	// aliases are the schema's, read.
	aliases map[string][]myelinkey.Key
}

// nodeInfo is a declared node type.
type nodeInfo struct {
	name string
	text bool
}

// varInfo is a variable a context reads: focus, a node, or a node's
// attribute.
type varInfo struct {
	node string // "" for focus
	attr string // "" for the node itself
	int  bool   // declared int: integers of any Go kind read as int64
}

// NewEnv declares a schema's path and layers to CEL. Every node is a bool
// variable (is it on the path?), every attribute a typed variable under its
// node ("list.dir"), and focus the innermost node's name. A name that isn't
// a CEL identifier, a node named with a reserved word (an attribute may
// be: `cluster.namespace`), a node declared twice, a type it can't read, no
// layers, a layer named twice and an alias that doesn't read are errors.
func NewEnv(s Schema) (*Env, error) {
	e := &Env{schema: s, nodes: map[string]*nodeInfo{}, ranks: map[string]int{}}
	decls := cellite.Decls{Vars: []cellite.Var{{Name: focusVar, Type: cellite.String}}}
	e.vars = []varInfo{{}}
	for _, n := range s.Nodes {
		switch {
		case !isIdent(n.Name):
			return nil, fmt.Errorf("node %q: not a CEL identifier", n.Name)
		case slices.Contains(cellite.ReservedWords(), n.Name):
			return nil, fmt.Errorf("node %q: the name is reserved in CEL", n.Name)
		case n.Name == focusVar:
			return nil, fmt.Errorf("node %q: the name is the focus's", n.Name)
		case e.nodes[n.Name] != nil:
			return nil, fmt.Errorf("node %q: declared twice", n.Name)
		}
		e.nodes[n.Name] = &nodeInfo{name: n.Name, text: n.Text}
		decls.Vars = append(decls.Vars, cellite.Var{Name: n.Name, Type: cellite.Bool})
		e.vars = append(e.vars, varInfo{node: n.Name})
		for _, a := range slices.Sorted(maps.Keys(n.Attrs)) {
			if !isIdent(a) {
				return nil, fmt.Errorf("node %q: attribute %q is not a CEL identifier", n.Name, a)
			}
			t, err := celType(string(n.Attrs[a]))
			if err != nil {
				return nil, fmt.Errorf("node %q: attribute %q: %w", n.Name, a, err)
			}
			decls.Vars = append(decls.Vars, cellite.Var{Name: n.Name + "." + a, Type: t})
			e.vars = append(e.vars, varInfo{node: n.Name, attr: a, int: t.Kind == cellite.IntKind})
		}
	}
	if len(s.Layers) == 0 {
		return nil, errors.New("a schema names its layers")
	}
	for i, l := range s.Layers {
		if _, dup := e.ranks[l]; dup || l == "" {
			return nil, fmt.Errorf("layer %q: empty or named twice", l)
		}
		e.ranks[l] = i
	}
	e.keys = myelinkey.Options{Strict: s.StrictKeys, ShiftedCapitals: s.ShiftedCapitals}
	e.aliases = make(map[string][]myelinkey.Key, len(s.Aliases))
	for name, keys := range s.Aliases {
		seq, err := e.readAlias(name, keys)
		if err != nil {
			return nil, err
		}
		e.aliases[name] = seq
	}
	env, err := cellite.NewEnv(decls)
	if err != nil {
		return nil, err
	}
	e.cel = env
	return e, nil
}

// isIdent reports whether s is a CEL identifier.
func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		if !(c == '_' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || i > 0 && '0' <= c && c <= '9') {
			return false
		}
	}
	return true
}

// celType reads an attribute's Type: string, bool, int, uint, double,
// bytes, dyn, list<T> and map<K,V>, K a bool, an int, a uint, a string or
// dyn.
func celType(s string) (cellite.Type, error) {
	t, rest, err := readType(strings.ReplaceAll(s, " ", ""))
	if err == nil && rest != "" {
		err = fmt.Errorf("%q: unexpected %q", s, rest)
	}
	return t, err
}

func readType(s string) (cellite.Type, string, error) {
	for name, t := range map[string]cellite.Type{
		"string": cellite.String, "bool": cellite.Bool, "int": cellite.Int, "uint": cellite.Uint,
		"double": cellite.Double, "bytes": cellite.Bytes, "dyn": cellite.Dyn,
	} {
		if rest, ok := strings.CutPrefix(s, name); ok && (rest == "" || strings.ContainsAny(rest[:1], ",>")) {
			return t, rest, nil
		}
	}
	if rest, ok := strings.CutPrefix(s, "list<"); ok {
		elem, rest, err := readType(rest)
		if err != nil {
			return cellite.Type{}, "", err
		}
		if rest, ok = strings.CutPrefix(rest, ">"); !ok {
			return cellite.Type{}, "", fmt.Errorf("list<%s: no >", s)
		}
		return cellite.ListOf(elem), rest, nil
	}
	if rest, ok := strings.CutPrefix(s, "map<"); ok {
		key, rest, err := readType(rest)
		if err != nil {
			return cellite.Type{}, "", err
		}
		switch key.Kind {
		case cellite.BoolKind, cellite.IntKind, cellite.UintKind, cellite.StringKind, cellite.DynKind:
		default:
			return cellite.Type{}, "", fmt.Errorf("map<%s: a key is a bool, an int, a uint, a string or dyn", s)
		}
		if rest, ok = strings.CutPrefix(rest, ","); !ok {
			return cellite.Type{}, "", fmt.Errorf("map<%s: no ,", s)
		}
		val, rest, err := readType(rest)
		if err != nil {
			return cellite.Type{}, "", err
		}
		if rest, ok = strings.CutPrefix(rest, ">"); !ok {
			return cellite.Type{}, "", fmt.Errorf("map<%s: no >", s)
		}
		return cellite.MapOf(key, val), rest, nil
	}
	return cellite.Type{}, "", fmt.Errorf("%q is not a type (string, bool, int, uint, double, bytes, dyn, list<T>, map<K,V>)", s)
}

// Context is a compiled context: a CEL boolean over the path. A nil Context
// holds everywhere, at depth 0.
type Context struct {
	env *Env
	src string
	prg *cellite.Program
}

// Compile compiles a context. A syntax error, a name the schema does not
// declare, a type error, what cel-lite doesn't support and an expression
// that is not a boolean are all errors here, so a keymap that loads never
// fails on a key. An empty context is nil, which holds everywhere.
func (e *Env) Compile(when string) (*Context, error) {
	if strings.TrimSpace(when) == "" {
		return nil, nil
	}
	prg, err := e.cel.Compile(when)
	if err != nil {
		var ce *cellite.Error
		if !errors.As(err, &ce) {
			return nil, fmt.Errorf("context %q: %w", when, err)
		}
		err := fmt.Errorf("context %q: %s", when, ce.Msg)
		if ce.Kind == cellite.Undeclared {
			err = fmt.Errorf("%w (a context reads %s)", err, e.vocabulary())
		}
		return nil, err
	}
	if t := prg.Type(); t.Kind != cellite.BoolKind {
		return nil, fmt.Errorf("context %q is a %s, not a boolean", when, cellite.FormatType(t))
	}
	return &Context{env: e, src: when, prg: prg}, nil
}

// vocabulary is the names a context may read, for an error that met another.
func (e *Env) vocabulary() string {
	names := []string{focusVar}
	for _, n := range e.schema.Nodes {
		names = append(names, n.Name)
	}
	return strings.Join(names, ", ")
}

// Holds reports whether the context holds on the whole path. A context that
// reads an attribute of a node not on the path, or one the path's step does
// not give, does not hold.
func (c *Context) Holds(p Path) bool {
	if c == nil {
		return true
	}
	a := c.env.act(p)
	return c.holds(a)
}

// Depth is how deep into the path the context reaches: the shortest prefix
// of the path it holds on, given that it holds on the whole path; -1 when it
// does not hold. A nil context is depth 0, and so is one that holds with
// nothing on the path (`!dialog`): saying what is not there adds no depth.
func (c *Context) Depth(p Path) int {
	if c == nil {
		return 0
	}
	return c.depth(c.env.act(p))
}

// MayHold reports whether the context can hold on the path when the
// attributes its steps do not give are unknown: it holds, or its answer
// depends on them. Asked of a path of nodes with no attributes, it says
// whether a binding can hold wherever those nodes have the focus.
func (c *Context) MayHold(p Path) bool {
	if c == nil {
		return true
	}
	a := c.env.act(p)
	return c.may(a)
}

// MayDepth is Depth with the attributes the path's steps do not give
// unknown: the shortest prefix the context may hold on, given that it may
// hold on the whole path; -1 when it cannot.
func (c *Context) MayDepth(p Path) int {
	if c == nil {
		return 0
	}
	a := c.env.act(p)
	return a.depthOf(c.may)
}

// String is the context as written; "" for none.
func (c *Context) String() string {
	if c == nil {
		return ""
	}
	return c.src
}

// depth is Depth over an activation of the path.
func (c *Context) depth(a *pathAct) int {
	if c == nil {
		return 0
	}
	return a.depthOf(c.holds)
}

// holds evaluates the context on the activation's prefix. An evaluation
// error, such as an absent attribute, does not hold.
func (c *Context) holds(a *pathAct) bool {
	holds, _ := c.prg.Test(a)
	return holds
}

// may evaluates the context on the activation's prefix with the attributes
// its steps do not give unknown: true when it holds or might.
func (c *Context) may(a *pathAct) bool {
	a.may = true
	holds, unknown := c.prg.Test(a)
	a.may = false
	return holds || unknown
}

// pathAct is a path as CEL reads it, cut to its first d steps: a node on
// them is true, any other declared node false, an attribute of a node on
// them its step's value, and focus the d-th step's name.
//
// A keymap's query takes one from the keymap's pool (Keymap.act), and it
// keeps the depth of each of the keymap's contexts on the whole path once
// it is known: bindings share contexts, and each is evaluated once per
// query.
type pathAct struct {
	env  *Env
	path Path
	d    int
	// may makes an attribute the steps don't give unknown rather than
	// absent: for MayHold and MayDepth.
	may bool
	// depths are the depths of the keymap's contexts on the whole path, by
	// index, unknownDepth until asked; nil outside a keymap's query.
	depths []int32
	// text is the path's textDepth.
	text int
	// fence is the fence that decides on the path, nil for none, and
	// fenceDepth its context's depth there. shadow is the depth a binding
	// must reach: the fence's when it shadows, else 0.
	fence              *boundFence
	fenceDepth, shadow int
	// keys is next's scratch: the index of each key it has listed.
	keys map[myelinkey.Key]int
	// props is groupAt's scratch, cleared after each use.
	props []groupProps
}

// unknownDepth marks a context's depth not yet evaluated.
const unknownDepth = -2

// act is an activation of the whole path, with no memory of depths: for a
// Context's own methods.
func (e *Env) act(p Path) *pathAct { return &pathAct{env: e, path: p, d: len(p), text: e.textDepth(p)} }

// ctxDepth is the depth on the whole path of the keymap's context number
// cid, c: evaluated the first time a query asks.
func (a *pathAct) ctxDepth(cid int32, c *Context) int {
	if c == nil {
		return 0
	}
	if d := a.depths[cid]; d != unknownDepth {
		return int(d)
	}
	d := a.depthOf(c.holds)
	a.depths[cid] = int32(d)
	return d
}

// depthOf is the shortest prefix test holds on, given that it holds on the
// whole path; -1 when it does not. It leaves the activation whole.
func (a *pathAct) depthOf(test func(*pathAct) bool) int {
	n := len(a.path)
	defer func() { a.d = n }()
	a.d = n
	if !test(a) {
		return -1
	}
	for d := 0; d < n; d++ {
		a.d = d
		if test(a) {
			return d
		}
	}
	return n
}

// step is the index of the innermost step of a node among the first d, or
// -1.
func (a *pathAct) step(node string) int {
	for i := a.d - 1; i >= 0; i-- {
		if a.path[i].Name == node {
			return i
		}
	}
	return -1
}

// Var is variable i's value (cellite.Activation): focus the innermost
// step's name, a node whether it is on the steps, and an attribute its
// node's innermost step's value. An attribute of a node not on the steps
// is absent, and so is one the step doesn't give, unknown instead when
// the activation asks what may hold.
func (a *pathAct) Var(i int) (cellite.Value, cellite.Presence) {
	v := a.env.vars[i]
	if v.node == "" {
		if a.d == 0 {
			return cellite.StringValue(""), cellite.Present
		}
		return cellite.StringValue(a.path[a.d-1].Name), cellite.Present
	}
	st := a.step(v.node)
	if v.attr == "" {
		return cellite.BoolValue(st >= 0), cellite.Present
	}
	if st < 0 {
		return cellite.Value{}, cellite.Absent
	}
	x, ok := a.path[st].Attrs[v.attr]
	switch {
	case !ok && a.may:
		return cellite.Value{}, cellite.Unknown
	case !ok:
		return cellite.Value{}, cellite.Absent
	case v.int:
		return intValue(x), cellite.Present
	}
	return cellite.ValueOf(x), cellite.Present
}

// intValue reads an integer of any Go kind as CEL's int, so a path decoded
// from msgpack (int64, or uint64 from another encoder) or built in Go (int)
// compares the same. A uint64 too big for an int stays a uint, and anything
// else is read as it is.
func intValue(v any) cellite.Value {
	switch n := v.(type) {
	case int:
		return cellite.IntValue(int64(n))
	case int8:
		return cellite.IntValue(int64(n))
	case int16:
		return cellite.IntValue(int64(n))
	case int32:
		return cellite.IntValue(int64(n))
	case uint:
		if uint64(n) <= 1<<63-1 {
			return cellite.IntValue(int64(n))
		}
	case uint8:
		return cellite.IntValue(int64(n))
	case uint16:
		return cellite.IntValue(int64(n))
	case uint32:
		return cellite.IntValue(int64(n))
	case uint64:
		if n <= 1<<63-1 {
			return cellite.IntValue(int64(n))
		}
	}
	return cellite.ValueOf(v)
}

// textDepth is the depth at which the innermost node that takes text joins
// the path (its step's position, from 1), or -1 with none: the depth a
// binding must reach to take a key that types text from it (the field
// rule).
func (e *Env) textDepth(p Path) int {
	for i := len(p) - 1; i >= 0; i-- {
		if n := e.nodes[p[i].Name]; n != nil && n.text {
			return i + 1
		}
	}
	return -1
}

// FormatPath writes a path from the outside in, each step with the
// attributes it sets, sorted: `window(name="main") > list(dir="/tmp",
// hidden) > field`. An empty string, a false bool and an empty map or
// list are left out.
func FormatPath(p Path) string {
	parts := make([]string, len(p))
	for i, st := range p {
		var attrs []string
		for a, v := range st.Attrs {
			switch v := v.(type) {
			case string:
				if v != "" {
					attrs = append(attrs, fmt.Sprintf("%s=%q", a, v))
				}
			case bool:
				if v {
					attrs = append(attrs, a)
				}
			case nil:
			default:
				if s := fmt.Sprint(v); s != "map[]" && s != "[]" {
					attrs = append(attrs, a+"="+s)
				}
			}
		}
		slices.Sort(attrs)
		parts[i] = st.Name
		if len(attrs) > 0 {
			parts[i] += "(" + strings.Join(attrs, ", ") + ")"
		}
	}
	return strings.Join(parts, " > ")
}
