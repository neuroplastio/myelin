---
group: Guides
order: 10
nav: Using myelin
description: declaring what can be focused, turning your files into documents, building a keymap, and describing where the focus is
path: integrate
---

# Using myelin in a program

A program that takes its keys from myelin keeps everything that is its
own: its config files and their format, its actions, its default keymap,
how it draws menus, and what happens to a key no binding takes. myelin
does the part in between. It works out which binding a key sequence runs
where the focus is, and what can follow it.

This page goes from a schema to a keymap you can ask. [Typing
keys](typing.md) feeds it keys, and [Menus and help](menus.md) draws what
it holds.

The code on this page is the package's examples, which `go test` runs, so
it compiles and prints what it says.

## Declare what can be focused

A `Schema` names the kinds of node that can be focused or hold the focus,
outermost first, and their attributes. Contexts are written against these names, and
a context that names anything else fails when the keymap loads.

- **Nodes** are what the program shows: a window, a list, a preview, a
  dialog, a text field. Each one's name is a CEL identifier, and not one
  of the words CEL reserves (an attribute may be one: `cluster.namespace`).
- **Attributes** are typed: `string`, `bool`, `int`, `uint`, `double`,
  `bytes`, `dyn`, `list<T>` and `map<K,V>`, a map keyed by a `bool`, an
  `int`, a `uint`, a `string` or `dyn`. Read them in contexts as
  `node.attribute`.
- **`Text`** marks a node that takes typed text. While one is on the path,
  a key that types text goes to it, unless a binding at least as deep
  takes the key.
- **Layers** are the keymap's layers, lowest first, such as
  `["defaults", "user"]`. At the same depth, a higher layer's binding wins.
- **`Aliases`** name key sequences that bindings write as `<name>`, such
  as `{"leader": "space"}` for `<leader>`. A document can name its own,
  and the highest layer's wins. **`Timeout`** is how long a sequence waits
  for its next key.
- **How keys are read.** By default keys are read the way Ghostty reads
  them: modifiers in any order and case, W3C code values, a letter in
  either case. **`StrictKeys`** accepts only each key's canonical spelling,
  and the error names it. **`ShiftedCapitals`** reads a capital as Shift
  with its letter, as vim writes keys: `G` is `shift+g`.

`NewEnv` compiles the schema once. The `Env` it returns is safe for
concurrent use.

## Turn your file into documents

Your parser reads your file and builds one `Doc` per layer, or several
documents for a layer, in load order. A `Bind` binds a sequence to an
action, with its values in `Args`. An unbind takes keys away, and a
`Group` labels a prefix for the menus. Put where each one was written in
its `Source`, such as `config.kdl:12`, because every problem myelin
reports carries it. Properties of your own go in a bind's or a group's
`Props`, which myelin passes back without reading.

To report every mistake in a file in one pass, check each sequence as you
parse it. `Env.CheckSeq` reads a sequence as `Build` will, without a
keymap. It takes an alias on trust, since a document may name it:

<!-- example:ExampleEnv_CheckSeq -->

```go
env, err := myelin.NewEnv(myelin.Schema{
	Nodes:  []myelin.NodeType{{Name: "list"}},
	Layers: []string{"defaults"},
})
if err != nil {
	panic(err)
}
for _, s := range []string{"<leader> f n", "Ctrl+X g", "ctrl+", "ctrl+ctrl+x"} {
	fmt.Printf("%s: %v\n", s, env.CheckSeq(s))
}
// Output:
// <leader> f n: <nil>
// Ctrl+X g: <nil>
// ctrl+: "ctrl+": no key after the last +
// ctrl+ctrl+x: "ctrl+ctrl+x": ctrl twice
```

## Build the keymap

`Env.Build` resolves the documents into a `Keymap`. Your `check` function
sees each binding with its sequence resolved and its layer named. It
returns an error to refuse the keymap, or `Warn(err)` to load it with a
warning. Problems come back with their source. With any error the keymap
is nil, so a keymap that loads never fails on a key.

<!-- example:ExampleEnv_Build -->

```go
env, err := myelin.NewEnv(myelin.Schema{
	Nodes:  []myelin.NodeType{{Name: "list"}},
	Layers: []string{"defaults", "user"},
})
if err != nil {
	panic(err)
}
actions := map[string]bool{"down": true, "up": true, "scroll": true}
check := func(b myelin.Binding) error {
	switch {
	case !actions[b.Action]:
		return fmt.Errorf("%q is not an action", b.Action)
	case b.Action == "scroll":
		return myelin.Warn(errors.New(`"scroll" is going away: use "down"`))
	}
	return nil
}
user := myelin.Doc{Layer: "user", Binds: []myelin.Bind{
	{Keys: "J", Action: "scroll", Source: "config:2"},
	{Keys: "ctrl+ctrl+k", Action: "up", Source: "config:3"},
	{Keys: "x", Action: "explode", Source: "config:4"},
	{Keys: "y", Action: "up", When: "list &&", Source: "config:5"},
}}
km, problems := env.Build([]myelin.Doc{user}, check)
for _, p := range problems {
	kind := "error"
	if p.Warning {
		kind = "warning"
	}
	fmt.Printf("%s: %s: %s\n", p.Source, kind, p.Text)
}
fmt.Println("loaded:", km != nil)
// Output:
// config:2: warning: "scroll" is going away: use "down"
// config:3: error: "ctrl+ctrl+k": ctrl twice
// config:4: error: "explode" is not an action
// config:5: error: context "list &&": syntax error: unexpected end of input
// loaded: false
```

## Describe where the focus is

A `Path` is the focused node and the nodes around it, outermost first, each
with its attributes. Build it from your program's state whenever you
look up a key. Building a path is cheap, and matching a key against it
takes well under a microsecond.

A context's depth is how far into the path it reaches. The deepest context
wins, then the higher layer, then the later binding:

<!-- example:ExampleContext_Depth -->

```go
env, err := myelin.NewEnv(myelin.Schema{
	Nodes: []myelin.NodeType{
		{Name: "window"}, {Name: "list"}, {Name: "preview"},
		{Name: "field", Text: true},
	},
	Layers: []string{"defaults"},
})
if err != nil {
	panic(err)
}
p := myelin.Path{{Name: "window"}, {Name: "list"}, {Name: "field"}}
for _, when := range []string{
	"", "window", "list", "field", "list && !preview", "!preview", "preview",
} {
	c, err := env.Compile(when)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%q reaches %d\n", when, c.Depth(p))
}
// Output:
// "" reaches 0
// "window" reaches 1
// "list" reaches 2
// "field" reaches 3
// "list && !preview" reaches 2
// "!preview" reaches 0
// "preview" reaches -1
```

`FormatPath` writes a path for logs and error messages:
`window > list(dir="/tmp") > field`.

## A widget's keys inside a host

A widget that another program embeds, such as a pager, can ship its own
document of bindings, written as if it were the whole program. `Mount`
puts that document under the node where the host shows the widget: it
returns a copy with every bind's, group's and fence's context joined to
the host's, as `(host) && (own)`. A binding with no context of its own
gets the host's alone. So `Mount(pager, "pane.kind == 'pager'")` keeps
the pager's keys to the panes that show it. The host then builds the
mounted document with its own, in whichever layer it chooses.

## Next

- [Typing keys](typing.md): feed keys to the sequencer, with modes and
  keys from your input.
- [Menus and help](menus.md): which-key menus, and explaining what a key
  does.
- [Across a process](process.md): documents, paths and menus as msgpack.
- [Keymaps](keymaps.md): link your users here for writing keymaps.

The full API is on
[pkg.go.dev](https://pkg.go.dev/github.com/neuroplastio/myelin).
