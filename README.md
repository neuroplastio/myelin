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
  `list.dir.startsWith('/tmp')`. The deepest context wins. They run on
  [cel-lite](https://github.com/neuroplastio/cel-lite), CEL in Go's
  standard library alone.
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

It needs Go 1.24 or later. myelin requires msgp's runtime, cel-lite and
its own key names. Keys are named in
`github.com/neuroplastio/myelin/myelinkey`, a module of its own that needs
only the standard library. A program that only names keys can require it
alone.

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

`j` does nothing in the filter because the field takes text: there, `j`
types a j.

[Getting started](docs/getting-started.md) goes on from here: your own
config format, what wins, sequences, which-key menus and keys.

## Documentation

- [Getting started](docs/getting-started.md): install, a first keymap,
  your own config, sequences, menus and keys.
- [Using myelin in a program](docs/integrate.md): the schema, documents,
  building a keymap and the path.
- [Typing keys](docs/typing.md): the sequencer, modes, and keys from your
  input.
- [Menus and help](docs/menus.md): which-key menus, and explaining what a
  key does.
- [Across a process](docs/process.md): the data types as msgpack.
- [Keymaps](docs/keymaps.md): how keys are written, what a context is and
  which binding wins. Programs link their users here.
- [Key names](myelinkey/SPEC.md): the spec for key names, version 0.1,
  with [conformance vectors](myelinkey/conformance/keys.json) for other
  implementations.
- [Performance](docs/performance.md): what a key, a match and a menu cost.
- [How myelin works](docs/design.md): why the keymap is data, why time is a
  value, and how depth and layers decide.
- The API reference is on pkg.go.dev:
  [myelin](https://pkg.go.dev/github.com/neuroplastio/myelin) and
  [myelinkey](https://pkg.go.dev/github.com/neuroplastio/myelin/myelinkey).

## Status

Pre-1.0. The current version is on
[pkg.go.dev](https://pkg.go.dev/github.com/neuroplastio/myelin) and among
the [tags](https://github.com/neuroplastio/myelin/tags). myelin follows
semantic versioning: until 1.0, a release that breaks the API raises the
minor version (0.1 to 0.2), and a patch release never does. Before 1.0
the API's edges may still change, as programs that use it ask for things,
and so may the wording of problems. How keys are written and which
binding wins change only by a recorded decision, checked against the
keymaps of programs that use myelin. The key names are a spec of their
own, versioned apart from the module.

## Developing

Go, msgp's generator and TinyGo are pinned in `mise.toml`.

- `mise exec -- make check`: gofmt, vet and the tests, myelinkey's under
  TinyGo too; the msgp codec, the conformance vectors and the docs' code
  current.
- `make generate`: the data types' msgp codec, after `data.go` changes.
- `make bench`: the benchmarks, with their allocations.
- `make fuzz`: a minute each of myelinkey's parsers.

## License

Apache License 2.0: see [LICENSE](LICENSE).
