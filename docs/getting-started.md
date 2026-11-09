---
group: Start
order: 10
nav: Getting started
description: install myelin, build a first keymap, read your own config, and add sequences, menus and keys
path: getting-started
---

# Getting started

myelin decides what a key does where the focus is. You declare the kinds
of thing that can be focused, hand myelin your bindings as plain Go
values, build a keymap from them once, and then ask it about each key.
This page builds the keymap of a small file manager: a window, a list of
files, and a filter field over the list.

The code on this page is the package's examples, which `go test` runs, so
it compiles and prints what it says.

## Install

```sh
go get neuroplast.io/myelin
```

myelin needs Go 1.24 or later. It requires msgp's runtime, for its data
types' codec, [cel-lite](https://github.com/neuroplastio/cel-lite), for
contexts, and its own key names: the module
`neuroplast.io/myelin/myelinkey`, which needs only the standard
library.

## A first keymap

**1. Declare what can be focused.** A schema names each kind of node, its
typed attributes, and whether it takes text. It also names the keymap's
layers, lowest first, and how long a sequence waits for its next key.

<!-- example:Example#1 -->

```go
env, err := myelin.NewEnv(myelin.Schema{
	Nodes: []myelin.NodeType{
		{Name: "window"},
		{Name: "list", Attrs: map[string]myelin.Type{"dir": "string"}},
		{Name: "field", Text: true},
	},
	Layers:  []string{"defaults", "user"},
	Timeout: 500 * time.Millisecond,
})
if err != nil {
	panic(err)
}
```

**2. Write the bindings.** A document holds one layer's bindings: keys,
an action, and the context where the binding holds, a
[CEL](https://cel.dev) expression over what is focused. A program usually
reads them from its files and the user's; here they are literals.

<!-- example:Example#2 -->

```go
defaults := myelin.Doc{Layer: "defaults", Binds: []myelin.Bind{
	{Keys: "j", Action: "down"},
	{Keys: "g g", Action: "top"},
	{Keys: "escape", Action: "back"},
	{Keys: "escape", Action: "close-filter", When: "field"},
	{Keys: "enter", Action: "open", When: "list"},
}}
user := myelin.Doc{Layer: "user", Binds: []myelin.Bind{
	{Keys: "enter", Action: "preview", When: "list.dir.startsWith('/tmp')"},
}}
```

**3. Build the keymap.** Build checks every key and every context against
the schema, and reports every mistake at once, each with where it was
written. With any error the keymap is nil, so a keymap that loads never
fails on a key.

<!-- example:Example#3 -->

```go
km, problems := env.Build([]myelin.Doc{defaults, user}, nil)
if km == nil {
	panic(fmt.Sprint(problems))
}
```

**4. Ask what keys do.** A path is the focused node and the nodes around
it, outermost first, each with its attributes. Build one from your
program's state when a key comes in.

<!-- example:Example#4 -->

```go
home := myelin.Path{
	{Name: "window"},
	{Name: "list", Attrs: map[string]any{"dir": "/home/me"}},
}
tmp := myelin.Path{
	{Name: "window"},
	{Name: "list", Attrs: map[string]any{"dir": "/tmp"}},
}
filter := append(slices.Clone(home), myelin.Step{Name: "field"})
match := func(keys string, p myelin.Path) string {
	seq, err := myelinkey.ParseSeq(keys)
	if err != nil {
		return err.Error()
	}
	switch m := km.Match(seq, p); {
	case m.Prefix:
		return "wait for more"
	case m.Exact != nil:
		return m.Exact.Action
	}
	return "nothing"
}
fmt.Println("enter in a list:", match("enter", home))
fmt.Println("enter in /tmp:", match("enter", tmp))
fmt.Println("escape in the filter:", match("escape", filter))
fmt.Println("j in the filter:", match("j", filter))
fmt.Println("g:", match("g", home))
// Output:
// enter in a list: open
// enter in /tmp: preview
// escape in the filter: close-filter
// j in the filter: nothing
// g: wait for more
```

## What wins

Each answer above comes from one rule: the binding whose context reaches
deepest into the path wins, then the one in the higher layer, then the
later one.

- **escape in the filter** runs `close-filter`, not `back`: the context
  `field` holds at the third node of the path, and a binding with no
  context holds at none.
- **enter in /tmp** runs the user's `preview`: both contexts hold at the
  list, so the higher layer wins.
- **j in the filter** does nothing: the field takes text, so a key that
  types text goes to it unless a binding at least as deep takes it. There,
  `j` types a j.
- **g** waits, because `g g` continues it. The sequencer below says what
  happens next.

[Keymaps](keymaps.md) has the whole rule, as your users read it.

## Your own config

myelin has no file format. A program decodes its config, in whatever
format it likes, into documents.

Here the defaults are sections in the style of Zed's `keymap.json`, a
context and the keys bound in it:

<!-- file:testdata/config/defaults.json -->

```json
[
  { "bindings": { "j": "down", "k": "up", "g g": "top", "escape": "back" } },
  { "context": "list", "bindings": { "enter": "open", "d d": "delete" } },
  { "context": "field", "bindings": { "escape": "close-filter" } }
]
```

and the user's overrides are rules in the style of VS Code's
`keybindings.json`, with an unbind:

<!-- file:testdata/config/user.json -->

```json
[
  {
    "keys": "enter",
    "action": "preview",
    "when": "list.dir.startsWith('/tmp')"
  },
  { "keys": "d d", "unbind": true, "when": "list" },
  { "keys": "ctrl+d", "action": "delete", "when": "list && !field" }
]
```

The sections become one bind per key:

<!-- example:Example_config#1 -->

```go
var sections []struct {
	Context  string
	Bindings map[string]string
}
data, err := os.ReadFile("testdata/config/defaults.json")
if err != nil {
	panic(err)
}
if err := json.Unmarshal(data, &sections); err != nil {
	panic(err)
}
defaults := myelin.Doc{Layer: "defaults", Source: "defaults.json"}
for _, s := range sections {
	for _, keys := range slices.Sorted(maps.Keys(s.Bindings)) {
		b := myelin.Bind{Keys: keys, Action: s.Bindings[keys], When: s.Context}
		defaults.Binds = append(defaults.Binds, b)
	}
}
```

The rules decode straight into binds:

<!-- example:Example_config#2 -->

```go
user := myelin.Doc{Layer: "user", Source: "user.json"}
if data, err = os.ReadFile("testdata/config/user.json"); err != nil {
	panic(err)
}
if err := json.Unmarshal(data, &user.Binds); err != nil {
	panic(err)
}
```

Then the keymap is built and asked as before:

<!-- example:Example_config#3 -->

```go
km, problems := env.Build([]myelin.Doc{defaults, user}, nil)
if km == nil {
	panic(fmt.Sprint(problems))
}
home := myelin.Path{
	{Name: "window"},
	{Name: "list", Attrs: map[string]any{"dir": "/home/me"}},
}
tmp := myelin.Path{
	{Name: "window"},
	{Name: "list", Attrs: map[string]any{"dir": "/tmp"}},
}
filter := append(slices.Clone(home), myelin.Step{Name: "field"})
for _, q := range []struct {
	keys, where string
	p           myelin.Path
}{
	{"enter", "a list", home},
	{"enter", "/tmp", tmp},
	{"d d", "a list", home},
	{"ctrl+d", "a list", home},
	{"escape", "the filter", filter},
	{"g g", "a list", home},
} {
	seq, _ := myelinkey.ParseSeq(q.keys)
	action := "nothing"
	if m := km.Match(seq, q.p); m.Exact != nil {
		action = m.Exact.Action
	}
	fmt.Printf("%s in %s: %s\n", q.keys, q.where, action)
}
// Output:
// enter in a list: open
// enter in /tmp: preview
// d d in a list: nothing
// ctrl+d in a list: delete
// escape in the filter: close-filter
// g g in a list: top
```

KDL, TOML, YAML or Lua work the same way: myelin only sees the values.

## Sequences and time

When `g` runs one thing and `g g` another, a `g` has to wait:

<!-- example:ExampleFeed#1 -->

```go
env, err := myelin.NewEnv(myelin.Schema{
	Nodes:   []myelin.NodeType{{Name: "list"}},
	Layers:  []string{"defaults"},
	Timeout: 500 * time.Millisecond,
})
if err != nil {
	panic(err)
}
km, problems := env.Build([]myelin.Doc{{Layer: "defaults", Binds: []myelin.Bind{
	{Keys: "g", Action: "go"},
	{Keys: "g g", Action: "top"},
}}}, nil)
if km == nil {
	panic(fmt.Sprint(problems))
}
```

myelin starts no timer. `Feed` takes the time each key came and returns a
deadline, and the program wakes at it and calls `Expire`:

<!-- example:ExampleFeed#2 -->

```go
p := myelin.Path{{Name: "list"}}
now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

var st myelin.Pending
st, res := myelin.Feed(km, st, "g", p, now)
fmt.Println("g waits until", res.Deadline.Format("15:04:05.000"))
st, res = myelin.Feed(km, st, "g", p, now.Add(200*time.Millisecond))
fmt.Println("g g runs", res.Run.Action)

st, _ = myelin.Feed(km, st, "g", p, now)
st, res = myelin.Expire(km, st, now.Add(100*time.Millisecond))
fmt.Println("before the deadline, Expire does nothing:", res.Run == nil)
st, res = myelin.Expire(km, st, now.Add(500*time.Millisecond))
fmt.Println("at the deadline, g runs", res.Run.Action)

_, res = myelin.Feed(km, st, "x", p, now)
fmt.Println("x matches nothing:", res.Dropped)
// Output:
// g waits until 12:00:00.500
// g g runs top
// before the deadline, Expire does nothing: true
// at the deadline, g runs go
// x matches nothing: [x]
```

[Typing keys](typing.md) has the rest of the sequencer, and modes.

## Which-key menus

A leader key, which bindings write as the alias `<leader>`, a group that
labels a prefix, and labelled bindings:

<!-- example:ExampleKeymap_Menu#1 -->

```go
env, err := myelin.NewEnv(myelin.Schema{
	Nodes:   []myelin.NodeType{{Name: "list"}},
	Layers:  []string{"defaults"},
	Aliases: map[string]string{"leader": "space"},
})
if err != nil {
	panic(err)
}
km, problems := env.Build([]myelin.Doc{{Layer: "defaults",
	Groups: []myelin.Group{{Keys: "<leader> f", Label: "+file"}},
	Binds: []myelin.Bind{
		{Keys: "<leader> f n", Action: "file.new", Label: "New file"},
		{Keys: "<leader> f d", Action: "file.delete", Label: "Delete"},
		{Keys: "<leader> q", Action: "quit", Label: "Quit"},
		{Keys: "<leader> arrow_left", Action: "back"},
	},
}}, nil)
if km == nil {
	panic(fmt.Sprint(problems))
}
```

A menu is data: its items, their labels and whether each applies here.
The program draws it however it likes.

<!-- example:ExampleKeymap_Menu#2 -->

```go
readOnly := func(b myelin.Binding) bool {
	return b.Action != "file.new" && b.Action != "file.delete"
}

leader, _ := km.Alias("leader")
show := func(m myelin.Menu) {
	fmt.Printf("%s (%s)\n", myelinkey.FormatSeq(m.Seq), m.Label)
	for _, it := range m.Items {
		label := it.Label
		if label == "" {
			// The program words a binding with no label.
			label = it.Binding.Action
		}
		key := myelinkey.Display(it.Key)
		fmt.Printf("  %s  %s enabled=%v\n", key, label, it.Enabled)
	}
}
p := myelin.Path{{Name: "list"}}
show(km.Menu(leader, p, readOnly))
show(km.Menu(slices.Concat(leader, []myelinkey.Key{"f"}), p, readOnly))
// Output:
// space ()
//   f  +file enabled=false
//   q  Quit enabled=true
//   ←  back enabled=true
// space f (+file)
//   n  New file enabled=false
//   d  Delete enabled=false
```

## Keys

A key is named as [Ghostty](https://ghostty.org/docs/config/keybind)
names it: its modifiers in the order `super+ctrl+alt+shift+`, then a
character (`g`, `?`), which follows the keyboard layout, or a physical key
(`page_down`, `key_w`), which stays in its place. Every way of writing a
key comes out as one name, so a keymap, a menu and an error message show
it the same way.

<!-- example:myelinkey.ExampleParse -->

```go
for _, s := range []string{
	"ctrl+d", "Shift+Ctrl+A", "cmd+s", "PageDown",
	"pgdn", "G", "ctrl+key_w", "?",
} {
	k, err := myelinkey.Parse(s)
	if err != nil {
		fmt.Println(err)
		continue
	}
	fmt.Printf("%s is %s\n", s, k)
}
// Output:
// ctrl+d is ctrl+d
// Shift+Ctrl+A is ctrl+shift+a
// cmd+s is super+s
// PageDown is page_down
// pgdn is page_down
// G is g
// ctrl+key_w is ctrl+key_w
// ? is ?
```

## Next

- [Using myelin in a program](integrate.md): the schema, documents,
  building a keymap and the path, in detail.
- [Typing keys](typing.md): the sequencer, modes, and naming keys from
  your input.
- [Menus and help](menus.md): drawing which-key menus, and explaining
  what a key does.
- [Keymaps](keymaps.md): what your users read about writing keymaps.
