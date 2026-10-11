# myelin

Context-bound keymaps and which-key menus for Go programs.

[![Go Reference](https://pkg.go.dev/badge/github.com/neuroplastio/myelin.svg)](https://pkg.go.dev/github.com/neuroplastio/myelin)
[![CI](https://github.com/neuroplastio/myelin/actions/workflows/ci.yml/badge.svg)](https://github.com/neuroplastio/myelin/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/neuroplastio/myelin)](go.mod)
[![License: Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

myelin decides what a key does in your Go program, given what is focused
right now. You declare the things that can be focused (a window, a list, a
text field), bind keys in contexts over them, and layer the user's config
over your defaults. Sequences like `g g`, a leader key and which-key
menus are built in. It has no UI and reads no files, so it works the same in
a TUI, a GUI or anything else that takes keys.

## Highlights

- 🧭 **Contexts.** A binding holds where its [CEL](https://cel.dev)
  expression holds, over the path to what is focused: `list && !field`,
  `list.dir.startsWith('/tmp')`. The deepest context wins.
- 🧱 **Layers.** Defaults, extensions, the user's config: a higher layer
  overrides a lower one, and a config can start from nothing.
- ⌨️ **Keys named as Ghostty names them.** `ctrl+d`, `shift+g`,
  `page_down`, `g g`, `<leader> f n`. A key is a character or a physical
  key, and every way of writing it comes out as one name.
- 🗂️ **Which-key menus.** One level, or the whole tree, as data that a TUI,
  a GUI or a phone can draw.
- ⏱️ **Time is a value.** `now` goes in and a deadline comes out. No clock,
  no goroutines, and tests that need no clock either.
- 📦 **Any config format.** myelin reads no files. Decode JSON, KDL, TOML or
  Lua into plain Go values.
- 🪶 **Headless.** No UI toolkit. Key names live in `myelinkey`, which needs
  only the standard library and builds with TinyGo.

## Why myelin

Editors worked out key binding a long time ago.

- In **VS Code**, each rule in `keybindings.json` has a key, a command and a
  `when` clause. Your rules go after the defaults, and the last rule that
  matches wins, so yours do.
- In **Zed**, bindings sit in contexts, in a tree of what's focused:
  Workspace, Pane, Editor. A binding in a deeper context beats one higher
  up, and a later one beats an earlier one. Sequences like `g g` and a
  which-key popup come built in.

Most other programs, terminal ones especially, hard-code their keys in a
`switch`: no overrides, no menus, and `ctrl+d` means something else in each.
myelin takes the editors' model and makes it a Go library.

|             | VS Code                          | Zed                                         | myelin                                                      |
|-------------|----------------------------------|---------------------------------------------|-------------------------------------------------------------|
| A binding   | `key`, `command`, `when`, `args` | a key and an action, in a `context`         | `Keys`, `Action`, `When`, `Args`                            |
| Context     | `when` over context keys         | a predicate over the context tree           | CEL over the path your program declares                     |
| Which wins  | the last rule that matches       | the deepest context, then the later binding | the deepest context, then the layer, then the later binding |
| Your config | appended after the defaults      | loaded after the defaults                   | a layer above the defaults, which it can drop               |
| Unbinding   | `-command`                       | `null`                                      | `Unbind`, which wins like a binding                         |
| Sequences   | chords: `ctrl+k ctrl+w`          | `g g`, with a timeout                       | `g g`, `<leader> f n`, with a deadline you wake at          |
| Menus       | —                                | a which-key popup                           | which-key levels and the whole tree, as data                |

From VS Code it takes flat rules, and a user's config layered over the
defaults. From Zed it takes contexts over a tree, the rule that the deepest
wins, and unbinding that wins the same way. The menus under a leader key
come from Spacemacs, and the key names from Ghostty.

## Install

```sh
go get github.com/neuroplastio/myelin
```

Keys are named in `github.com/neuroplastio/myelin/myelinkey`, a module of
its own that needs only the standard library. A program that only names
keys can require it alone.

## Quick start

A file manager: a window, a list of files, and a filter field over the
list.

**1. Declare what can be focused.** Each kind of node can have typed
attributes, and a node that takes text keeps the keys that type it.

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

**2. Write the bindings.** A program usually reads them from its files and
the user's; here they are literals.

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

**3. Build the keymap.** Every mistake is reported at once, with where it
was written.

<!-- example:Example#3 -->

```go
km, problems := env.Build([]myelin.Doc{defaults, user}, nil)
if km == nil {
	panic(fmt.Sprint(problems))
}
```

**4. Ask what keys do.** The path is the focused node and the nodes around
it, outermost first.

<!-- example:Example#4 -->

```go
home := myelin.Path{{Name: "window"}, {Name: "list", Attrs: map[string]any{"dir": "/home/me"}}}
tmp := myelin.Path{{Name: "window"}, {Name: "list", Attrs: map[string]any{"dir": "/tmp"}}}
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

`j` does nothing in the filter because the field takes text: there, `j`
types a j.

## Bring your own config

myelin has no file format. A program decodes its config, in whatever format
it likes, into documents: plain Go values.

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
  { "keys": "enter", "action": "preview", "when": "list.dir.startsWith('/tmp')" },
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
		defaults.Binds = append(defaults.Binds, myelin.Bind{Keys: keys, Action: s.Bindings[keys], When: s.Context})
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
home := myelin.Path{{Name: "window"}, {Name: "list", Attrs: map[string]any{"dir": "/home/me"}}}
tmp := myelin.Path{{Name: "window"}, {Name: "list", Attrs: map[string]any{"dir": "/tmp"}}}
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

## Contexts: the deepest wins

A context's depth is how far along the path it has to look to hold: `list`
holds at the list, `field` at the field inside it. A context that holds
with nothing on the path, such as `!preview`, is as shallow as none.

<!-- example:ExampleContext_Depth -->

```go
env, err := myelin.NewEnv(myelin.Schema{
	Nodes:  []myelin.NodeType{{Name: "window"}, {Name: "list"}, {Name: "preview"}, {Name: "field", Text: true}},
	Layers: []string{"defaults"},
})
if err != nil {
	panic(err)
}
p := myelin.Path{{Name: "window"}, {Name: "list"}, {Name: "field"}}
for _, when := range []string{"", "window", "list", "field", "list && !preview", "!preview", "preview"} {
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

When several bindings of the same keys hold, the deepest wins, then the
higher layer, then the later binding. [Keymaps](docs/keymaps.md) has the
whole rule.

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

A menu is data: its items, their labels and whether each applies here. The
program draws it however it likes.

<!-- example:ExampleKeymap_Menu#2 -->

```go
readOnly := func(b myelin.Binding) bool { return b.Action != "file.new" && b.Action != "file.delete" }

leader, _ := km.Alias("leader")
show := func(m myelin.Menu) {
	fmt.Printf("%s (%s)\n", myelinkey.FormatSeq(m.Seq), m.Label)
	for _, it := range m.Items {
		label := it.Label
		if label == "" {
			label = it.Binding.Action // the program words a binding with no label
		}
		fmt.Printf("  %s  %s enabled=%v\n", myelinkey.Display(it.Key), label, it.Enabled)
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

A key is named as [Ghostty](https://ghostty.org/docs/config/keybind) names
it: its modifiers in the order `super+ctrl+alt+shift+`, then a character
(`g`, `?`), which follows the keyboard layout, or a physical key
(`page_down`, `key_w`), which stays in its place. myelin reads keys as
Ghostty does, and every way of writing a key comes out as one name, so a
keymap, a menu and an error message show it the same way. A program that
wants one spelling per key reads strictly, and any other spelling is
refused with the one to use.

<!-- example:myelinkey.ExampleParse -->

```go
for _, s := range []string{"ctrl+d", "Shift+Ctrl+A", "cmd+s", "PageDown", "pgdn", "G", "ctrl+key_w", "?"} {
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

## Documentation

- [Using myelin in a program](docs/integrate.md): the schema, building a
  keymap, the sequencer and menus.
- [Keymaps](docs/keymaps.md): how keys are written, what a context is and
  which binding wins. Programs link their users here.
- [How myelin works](docs/design.md): why the keymap is data, why time is a
  value, and how depth and layers decide.
- [Key names](myelinkey/SPEC.md): the spec for key names, version 0.1,
  with [conformance vectors](myelinkey/conformance/keys.json) for other
  implementations.
- The API reference is on pkg.go.dev:
  [myelin](https://pkg.go.dev/github.com/neuroplastio/myelin) and
  [myelinkey](https://pkg.go.dev/github.com/neuroplastio/myelin/myelinkey).

## Status

Early: the API can still change.

## License

Apache-2.0.
