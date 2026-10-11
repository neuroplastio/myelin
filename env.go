package myelin

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/interpreter"

	"github.com/neuroplastio/myelin/myelinkey"
)

// focusVar is the variable that names the innermost node on the path.
const focusVar = "focus"

// Env is the CEL environment a schema declares: the handle every context is
// compiled in, and every keymap built with. It is safe for concurrent use.
type Env struct {
	schema Schema
	cel    *cel.Env
	// nodes are the declared node types by name; attrs the declared
	// attributes by qualified name ("list.dir").
	nodes map[string]*nodeInfo
	attrs map[string]attrInfo
	// ranks are the layers' ranks, lowest 0.
	ranks map[string]int
	// keys is how the schema reads keys.
	keys myelinkey.Options
	// aliases are the schema's, read.
	aliases map[string][]myelinkey.Key
}

// nodeInfo is a declared node type.
type nodeInfo struct {
	name  string
	text  bool
	attrs []string // its attributes' qualified names, sorted
}

// attrInfo is a declared attribute.
type attrInfo struct {
	node, name string
	int        bool // declared int: integers of any Go kind read as int64
}

// NewEnv declares a schema's path and layers to CEL. Every node is a bool
// variable (is it on the path?), every attribute a typed variable under its
// node ("list.dir"), and focus the innermost node's name. A name that isn't
// a CEL identifier, a node declared twice, a type it can't read, no layers,
// a layer named twice and an alias that doesn't read are errors.
func NewEnv(s Schema) (*Env, error) {
	e := &Env{schema: s, nodes: map[string]*nodeInfo{}, attrs: map[string]attrInfo{}, ranks: map[string]int{}}
	opts := []cel.EnvOption{cel.Variable(focusVar, cel.StringType)}
	for _, n := range s.Nodes {
		switch {
		case !isIdent(n.Name):
			return nil, fmt.Errorf("node %q: not a CEL identifier", n.Name)
		case n.Name == focusVar:
			return nil, fmt.Errorf("node %q: the name is the focus's", n.Name)
		case e.nodes[n.Name] != nil:
			return nil, fmt.Errorf("node %q: declared twice", n.Name)
		}
		info := &nodeInfo{name: n.Name, text: n.Text}
		e.nodes[n.Name] = info
		opts = append(opts, cel.Variable(n.Name, cel.BoolType))
		for a, typ := range n.Attrs {
			if !isIdent(a) {
				return nil, fmt.Errorf("node %q: attribute %q is not a CEL identifier", n.Name, a)
			}
			t, err := celType(string(typ))
			if err != nil {
				return nil, fmt.Errorf("node %q: attribute %q: %w", n.Name, a, err)
			}
			q := n.Name + "." + a
			info.attrs = append(info.attrs, q)
			e.attrs[q] = attrInfo{node: n.Name, name: a, int: t == cel.IntType}
			opts = append(opts, cel.Variable(q, t))
		}
		slices.Sort(info.attrs)
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
	env, err := cel.NewEnv(opts...)
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
// bytes, dyn, list<T> and map<K,V>.
func celType(s string) (*cel.Type, error) {
	t, rest, err := readType(strings.ReplaceAll(s, " ", ""))
	if err == nil && rest != "" {
		err = fmt.Errorf("%q: unexpected %q", s, rest)
	}
	return t, err
}

func readType(s string) (*cel.Type, string, error) {
	for name, t := range map[string]*cel.Type{
		"string": cel.StringType, "bool": cel.BoolType, "int": cel.IntType, "uint": cel.UintType,
		"double": cel.DoubleType, "bytes": cel.BytesType, "dyn": cel.DynType,
	} {
		if rest, ok := strings.CutPrefix(s, name); ok && (rest == "" || strings.ContainsAny(rest[:1], ",>")) {
			return t, rest, nil
		}
	}
	if rest, ok := strings.CutPrefix(s, "list<"); ok {
		elem, rest, err := readType(rest)
		if err != nil {
			return nil, "", err
		}
		if rest, ok = strings.CutPrefix(rest, ">"); !ok {
			return nil, "", fmt.Errorf("list<%s: no >", s)
		}
		return cel.ListType(elem), rest, nil
	}
	if rest, ok := strings.CutPrefix(s, "map<"); ok {
		key, rest, err := readType(rest)
		if err != nil {
			return nil, "", err
		}
		if rest, ok = strings.CutPrefix(rest, ","); !ok {
			return nil, "", fmt.Errorf("map<%s: no ,", s)
		}
		val, rest, err := readType(rest)
		if err != nil {
			return nil, "", err
		}
		if rest, ok = strings.CutPrefix(rest, ">"); !ok {
			return nil, "", fmt.Errorf("map<%s: no >", s)
		}
		return cel.MapType(key, val), rest, nil
	}
	return nil, "", fmt.Errorf("%q is not a type (string, bool, int, uint, double, bytes, dyn, list<T>, map<K,V>)", s)
}

// Context is a compiled context: a CEL boolean over the path. A nil Context
// holds everywhere, at depth 0.
type Context struct {
	env *Env
	src string
	ast *cel.Ast
	prg cel.Program
	// partial is prg for partial evaluation (MayHold, MayDepth), built on
	// first use.
	partial func() cel.Program
}

// Compile compiles a context. A syntax error, a name the schema does not
// declare, a type error and an expression that is not a boolean are all
// errors here, so a keymap that loads never fails on a key. An empty
// context is nil, which holds everywhere.
func (e *Env) Compile(when string) (*Context, error) {
	if strings.TrimSpace(when) == "" {
		return nil, nil
	}
	ast, iss := e.cel.Compile(when)
	if iss.Err() != nil {
		var msgs []string
		unknown := false
		for _, err := range iss.Errors() {
			msgs = append(msgs, strings.TrimSuffix(err.Message, " (in container '')"))
			unknown = unknown || strings.HasPrefix(err.Message, "undeclared reference")
		}
		err := fmt.Errorf("context %q: %s", when, strings.Join(msgs, "; "))
		if unknown {
			err = fmt.Errorf("%w (a context reads %s)", err, e.vocabulary())
		}
		return nil, err
	}
	if !ast.OutputType().IsExactType(cel.BoolType) {
		return nil, fmt.Errorf("context %q is a %s, not a boolean", when, ast.OutputType())
	}
	prg, err := e.cel.Program(ast, cel.EvalOptions(cel.OptOptimize))
	if err != nil {
		return nil, fmt.Errorf("context %q: %w", when, err)
	}
	c := &Context{env: e, src: when, ast: ast, prg: prg}
	c.partial = sync.OnceValue(func() cel.Program {
		p, err := e.cel.Program(ast, cel.EvalOptions(cel.OptPartialEval))
		if err != nil {
			return nil
		}
		return p
	})
	return c, nil
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
	out, _, err := c.prg.Eval(a)
	return err == nil && out == types.True
}

// may evaluates the context on the activation's prefix with the attributes
// its steps do not give unknown: true when it holds or might.
func (c *Context) may(a *pathAct) bool {
	prg := c.partial()
	if prg == nil {
		return true
	}
	vars, err := cel.PartialVars(a, a.unknowns()...)
	if err != nil {
		return true
	}
	out, _, err := prg.Eval(vars)
	return err == nil && (out == types.True || types.IsUnknown(out))
}

// pathAct is a path as CEL reads it, cut to its first d steps: a node on
// them is true, any other declared node false, an attribute of a node on
// them its step's value, and focus the d-th step's name.
type pathAct struct {
	env  *Env
	path Path
	d    int
}

// act is the activation of the whole path.
func (e *Env) act(p Path) *pathAct { return &pathAct{env: e, path: p, d: len(p)} }

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

// ResolveName is CEL's lookup of a variable (interpreter.Activation).
func (a *pathAct) ResolveName(name string) (any, bool) {
	if name == focusVar {
		if a.d == 0 {
			return "", true
		}
		return a.path[a.d-1].Name, true
	}
	if _, node := a.env.nodes[name]; node {
		return a.step(name) >= 0, true
	}
	at, ok := a.env.attrs[name]
	if !ok {
		return nil, false
	}
	i := a.step(at.node)
	if i < 0 {
		return nil, false
	}
	v, ok := a.path[i].Attrs[at.name]
	if !ok {
		return nil, false
	}
	if at.int {
		v = asInt(v)
	}
	return v, true
}

// Parent is nil: a path is the whole activation.
func (a *pathAct) Parent() interpreter.Activation { return nil }

// unknowns are the attributes the prefix's steps do not give, as patterns
// for partial evaluation.
func (a *pathAct) unknowns() []*cel.AttributePatternType {
	var out []*cel.AttributePatternType
	for i, st := range a.path[:a.d] {
		n := a.env.nodes[st.Name]
		if n == nil || a.step(st.Name) != i {
			continue
		}
		for _, q := range n.attrs {
			if _, given := st.Attrs[a.env.attrs[q].name]; !given {
				out = append(out, cel.AttributePattern(q))
			}
		}
	}
	return out
}

// asInt reads an integer of any Go kind as CEL's int, so a path decoded from
// msgpack (int64, or uint64 from another encoder) or built in Go (int)
// compares the same. A uint64 too big for an int stays as it is.
func asInt(v any) any {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int8:
		return int64(n)
	case int16:
		return int64(n)
	case int32:
		return int64(n)
	case uint:
		if uint64(n) <= 1<<63-1 {
			return int64(n)
		}
	case uint8:
		return int64(n)
	case uint16:
		return int64(n)
	case uint32:
		return int64(n)
	case uint64:
		if n <= 1<<63-1 {
			return int64(n)
		}
	}
	return v
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
