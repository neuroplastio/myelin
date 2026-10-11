# myelin

Context-bound keymaps and which-key menus for Go programs.

Practice wraps a nerve pathway in myelin, and the signal along it gets fast
enough that the hand moves before you think. This library is for that kind
of muscle memory: keys that mean the same thing wherever you are, in every
program built on it.

- **Data in, data out.** A program reads its keymap from its own files, in
  its own format, and hands it to myelin as values. It gets data back:
  matches, menus, explanations and the state of a sequence in progress.
  myelin reads no files.
- **Keys** are written in one short spelling: `ctrl+d`, `pgdn`, `shift+tab`,
  `g g`, `<leader> f n`. They compare by their
  [HOTTY](https://hotty.neuroplast.io) names, so a key from a terminal,
  from Bubble Tea or from a HOTTY host is the same value.
- **Contexts** are [CEL](https://cel.dev) expressions over what holds the
  keyboard, such as `list && !field`. The deepest context wins, then the
  layer, then the later binding.
- **Menus** are which-key levels, or the whole tree, as data that any front
  end can draw.
- **Time is a value.** The program passes `now` in and gets a deadline back.
  myelin reads no clock and starts no goroutine.

## Install

```sh
go get github.com/neuroplastio/myelin
```

Keys are named in `github.com/neuroplastio/myelin/myelinkey`, a module of
its own that needs only the standard library. A program that only names
keys can require it alone.

## A first keymap

A file manager's defaults and a user's config, matched on the paths the
keyboard takes:

<!-- example:Example -->

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
defaults := myelin.Doc{Layer: "defaults", Binds: []myelin.Bind{
	{Keys: "j", Action: "down"},
	{Keys: "g g", Action: "top"},
	{Keys: "esc", Action: "back"},
	{Keys: "esc", Action: "close-filter", When: "field"},
	{Keys: "enter", Action: "open", When: "list"},
}}
user := myelin.Doc{Layer: "user", Binds: []myelin.Bind{
	{Keys: "enter", Action: "preview", When: "list.dir.startsWith('/tmp')"},
}}
km, problems := env.Build([]myelin.Doc{defaults, user}, nil)
if km == nil {
	panic(fmt.Sprint(problems))
}

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
fmt.Println("esc in the filter:", match("esc", filter))
fmt.Println("j in the filter:", match("j", filter))
fmt.Println("g:", match("g", home))
// Output:
// enter in a list: open
// enter in /tmp: preview
// esc in the filter: close-filter
// j in the filter: nothing
// g: wait for more
```

## Documentation

- [Using myelin in a program](docs/integrate.md): the schema, building a
  keymap, the sequencer and menus.
- [Keymaps](docs/keymaps.md): how keys are written, what a context is and
  which binding wins. Programs link their users here.
- [How myelin works](docs/design.md): why the keymap is data, why time is a
  value, and how depth and layers decide.
- The API reference is on pkg.go.dev:
  [myelin](https://pkg.go.dev/github.com/neuroplastio/myelin) and
  [myelinkey](https://pkg.go.dev/github.com/neuroplastio/myelin/myelinkey).

## Status

Early: the API can still change.

## License

Apache-2.0.
