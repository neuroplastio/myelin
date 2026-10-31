---
group: Guides
order: 30
nav: Menus and help
description: which-key menus and the whole tree as data, a group's own properties, and explaining, listing and linting a keymap
path: menus
---

# Menus and help

A keymap knows what can follow any sequence and why each binding wins or
loses. It gives that back as data: menus for a which-key popup or a card
of every binding, and answers for a help screen, a command palette or a
check of the user's config. Drawing them is the program's part.

## Draw menus

`Keymap.Menu` is the which-key level after a sequence: an item for each
key that can follow it, a group where more keys follow, or the binding
that wins there. `Keymap.Tree` nests every level, for a card of every
binding. Your `applies` function says whether a binding would do anything
now, and a group is enabled when anything under it is. You word a binding
that has no label, and you draw the menu, with `myelinkey.Display` for
the keys.

A group's `Props` come back on its item and on the menu after it, so you
can ask each level of a sequence what it does, such as whether it stays
open after a binding runs. Where several groups of the same keys hold,
each property is the winning one's among those that set it, by the same
precedence as the label. So a user's group that only relabels a prefix
keeps your defaults' properties, and to turn one off it sets that
property to a value you read as off. A group can be an alias alone, such
as `<leader>`, to give the leader's own level a label and properties.

<!-- example:ExampleKeymap_Menu -->

```go
// 1. A leader key, which bindings write as the alias <leader>, a group
// that labels a prefix, and labelled bindings.
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

// 2. Draw the menu after the leader, and after the leader and f. The
// program says what applies here: in a read-only list, not editing.
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

## Help, explain and lint

A help screen, a command palette and a check of the user's config ask the
keymap about bindings rather than about one key:

- `All` lists every binding in load order, with its source and layer, for
  a listing of the keymap in force.
- `Effective` is what each key runs on a path. `KeysFor` gives every
  sequence that runs an action there, and `KeyFor` the one to show beside
  it in a command palette.
- `Explain` says what a sequence does on a path and why: every binding
  considered, ranked, with its depth, whether it wins, and whether a
  mode's fence shadows it.
- `Shadowed` finds bindings that a deeper binding in a lower layer beats
  on any of the paths you pass, such as a user's binding that a default
  always beats in a dialog.
- `Context.MayHold` and `MayDepth` answer with the attributes a path
  leaves out unknown. For example, they can check whether a binding could
  hold wherever a list has the focus, whatever the list shows.

`Explain` is what a program's "why does this key do that?" command
prints:

<!-- example:ExampleKeymap_Explain -->

```go
// 1. The file manager's defaults, and a user's override.
env, err := myelin.NewEnv(myelin.Schema{
	Nodes: []myelin.NodeType{
		{Name: "window"},
		{Name: "list", Attrs: map[string]myelin.Type{"dir": "string"}},
		{Name: "field", Text: true},
	},
	Layers: []string{"defaults", "user"},
})
if err != nil {
	panic(err)
}
km, problems := env.Build([]myelin.Doc{
	{Layer: "defaults", Binds: []myelin.Bind{
		{Keys: "escape", Action: "back"},
		{Keys: "escape", Action: "close-filter", When: "field"},
		{Keys: "enter", Action: "open", When: "list"},
	}},
	{Layer: "user", Binds: []myelin.Bind{
		{Keys: "enter", Action: "preview", When: "list.dir == '/tmp'"},
	}},
}, nil)
if km == nil {
	panic(fmt.Sprint(problems))
}

// 2. Explain a key on a path: every binding of it, the winner marked.
explain := func(keys string, p myelin.Path) {
	seq, err := km.ParseSeq(keys)
	if err != nil {
		panic(err)
	}
	fmt.Println(keys, "on", myelin.FormatPath(p))
	for _, c := range km.Explain(seq, p) {
		mark := " "
		if c.Wins {
			mark = "*"
		}
		b := c.Binding
		fmt.Printf("  %s %-12s %-8s depth %d\n",
			mark, b.Action, b.Layer, c.Depth)
	}
}
tmp := myelin.Path{
	{Name: "window"},
	{Name: "list", Attrs: map[string]any{"dir": "/tmp"}},
}
filter := append(slices.Clone(tmp), myelin.Step{Name: "field"})
explain("escape", filter)
explain("enter", tmp)

// 3. The keys to show beside an action in a command palette.
if b := km.KeyFor("close-filter", filter); b != nil {
	fmt.Println("close-filter:", myelinkey.FormatSeq(b.Seq))
}
// Output:
// escape on window > list(dir="/tmp") > field
//   * close-filter defaults depth 3
//     back         defaults depth 0
// enter on window > list(dir="/tmp")
//   * preview      user     depth 2
//     open         defaults depth 2
// close-filter: escape
```

## Next

- [Across a process](process.md): a menu drawn by a front end in another
  process.
- [Keymaps](keymaps.md): what your users read about keys, contexts and
  what wins.
