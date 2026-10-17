---
group: Guides
order: 10
nav: Using myelin
description: declaring what can be focused, building a keymap, typing keys and drawing menus
path: integrate
---

# Using myelin in a program

A program that takes its keys from myelin keeps everything that is its
own: its config files and their format, its actions, its default keymap,
how it draws menus, and what happens to a key no binding takes. myelin
does the part in between. It works out which binding a key sequence runs
where the focus is, and what can follow it.

The code on this page is the package's examples, which `go test` runs, so
it compiles and prints what it says.

## Declare what can be focused

A `Schema` names the kinds of node that can be focused or hold the focus,
outermost first, and their attributes. Contexts are written against these names, and
a context that names anything else fails when the keymap loads.

- **Nodes** are what the program shows: a window, a list, a preview, a
  dialog, a text field. Each one's name is a CEL identifier.
- **Attributes** are typed: `string`, `bool`, `int`, `uint`, `double`,
  `bytes`, `dyn`, `list<T>` and `map<K,V>`. Read them in contexts as
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
reports carries it. Properties of your own go in `Props`, which myelin
passes back without reading.

To report every mistake in a file in one pass, check each sequence as you
parse it. `Env.CheckSeq` reads a sequence as `Build` will, without a
keymap. It takes an alias on trust, since a document may name it:

<!-- example:ExampleEnv_CheckSeq -->

```go
env, err := myelin.NewEnv(myelin.Schema{Nodes: []myelin.NodeType{{Name: "list"}}, Layers: []string{"defaults"}})
if err != nil {
	panic(err)
}
for _, s := range []string{"<leader> f n", "Ctrl+X g", "meta+x"} {
	fmt.Printf("%s: %v\n", s, env.CheckSeq(s))
}
// Output:
// <leader> f n: <nil>
// Ctrl+X g: <nil>
// meta+x: "meta+x" is not a key (meta is no modifier: write super+ for the command or Windows key, alt+ for Emacs's Meta)
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
	{Keys: "meta+k", Action: "up", Source: "config:3"},
	{Keys: "x", Action: "explode", Source: "config:4"},
	{Keys: "y", Action: "up", When: "lsit", Source: "config:5"},
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
// config:3: error: "meta+k" is not a key (meta is no modifier: write super+ for the command or Windows key, alt+ for Emacs's Meta)
// config:4: error: "explode" is not an action
// config:5: error: context "lsit": undeclared reference to 'lsit' (a context reads focus, list)
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

`FormatPath` writes a path for logs and error messages:
`window > list(dir="/tmp") > field`.

## Type keys

The sequencer is three functions over a `Pending` value, which your
program keeps. `Feed` takes a key, the path and the time. It returns the
new state and a `Result`, in which at most one thing is set:

- `Run`, a binding to run;
- `Deadline`, when keys are buffered because a longer binding continues
  them. Wake at the deadline and call `Expire`, which runs the binding the
  buffered keys finish, if any. Expire does nothing before the deadline,
  so a stale timer is harmless;
- `Dropped`, keys that matched nothing. They are yours: type them into a
  field, forward them to a terminal, or swallow them. Feed never retries a
  key on its own;
- `Swallowed`, keys that matched nothing where a mode sends them nowhere
  ([Modes](#modes)). Do nothing with them.

<!-- example:ExampleFeed -->

```go
// 1. A keymap where g runs one thing and g g another: after g, it waits.
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

// 2. Feed keys with the time they came. A deadline means: wake then,
// and call Expire.
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

`Arm` returns the state as if an alias's keys had been typed, such as
`Arm(km, "leader", path, now)`, for a tap or a touch-screen button that
opens the leader's menu.

## Modes

A mode changes what happens to the keys that its own bindings don't take.
A multiplexer's locked mode passes every key to the terminal but its own,
and a dialog lets no key past it. A document's `Fences` say this for a
context, with two options:

- `Shadow` leaves out the bindings whose contexts are shallower than the
  fence's, so only those at least as deep apply. Menus show only them,
  and `Explain` marks the others `Fenced`.
- `Swallow` sends the keys that match nothing nowhere: Feed and Expire
  put them in `Swallowed`, not `Dropped`. A key that types text still
  drops where a text field is deeper than the fence, so a palette opened
  in the mode keeps its text. A sequence that dies is decided on the path
  its first key was typed on.

One fence decides on a path, as a binding wins: the deepest, then the
higher layer, then the later. A fence with neither option set gives a
deeper context the default back. `Keymap.Fence` returns the fence that
decides, for a status line that names the mode.

<!-- example:ExampleFence -->

```go
// 1. A window in a mode, its pane, and a dialog over it.
env, err := myelin.NewEnv(myelin.Schema{
	Nodes: []myelin.NodeType{
		{Name: "window", Attrs: map[string]myelin.Type{"mode": "string"}},
		{Name: "pane"}, {Name: "dialog"},
	},
	Layers: []string{"defaults"},
})
if err != nil {
	panic(err)
}
locked := "window.mode == 'locked'"
km, problems := env.Build([]myelin.Doc{{
	Layer: "defaults",
	Binds: []myelin.Bind{
		{Keys: "ctrl+q", Action: "quit"},
		{Keys: "ctrl+g", Action: "unlock", When: locked},
		{Keys: "escape", Action: "close", When: "dialog"},
	},
	Fences: []myelin.Fence{
		{When: locked, Shadow: true, Source: "locked"},
		{When: "dialog", Shadow: true, Swallow: true, Source: "dialog"},
	},
}}, nil)
if km == nil {
	panic(fmt.Sprint(problems))
}

// 2. Feed keys where the focus is. The Result says what to do.
feed := func(where string, p myelin.Path, k myelinkey.Key) {
	_, res := myelin.Feed(km, myelin.Pending{}, k, p, time.Time{})
	switch {
	case res.Run != nil:
		fmt.Println(where, k, "runs", res.Run.Action)
	case res.Dropped != nil:
		fmt.Println(where, k, "goes to the terminal")
	case res.Swallowed != nil:
		fmt.Println(where, k, "does nothing")
	}
}
window := func(mode string) myelin.Step {
	return myelin.Step{Name: "window", Attrs: map[string]any{"mode": mode}}
}
normal := myelin.Path{window("normal"), {Name: "pane"}}
lockedPane := myelin.Path{window("locked"), {Name: "pane"}}
dialog := myelin.Path{window("normal"), {Name: "dialog"}}
feed("normal:", normal, "ctrl+q")
feed("normal:", normal, "x")
feed("locked:", lockedPane, "ctrl+q")
feed("locked:", lockedPane, "ctrl+g")
feed("dialog:", dialog, "ctrl+q")
feed("dialog:", dialog, "escape")

// 3. The fence that decides names the mode, for a status line.
fmt.Println("the mode:", km.Fence(lockedPane).Source)
// Output:
// normal: ctrl+q runs quit
// normal: x goes to the terminal
// locked: ctrl+q goes to the terminal
// locked: ctrl+g runs unlock
// dialog: ctrl+q does nothing
// dialog: escape runs close
// the mode: locked
```

## Keys from your input

A `myelinkey.Key` is a keystroke's canonical name, and two keys are the
same keystroke exactly when they are `==`. Name each key from your input
as either a character or a physical key. Which one is your choice, and a
binding matches only a key of its own kind:

- **A character** follows the layout. Ctrl with the key that types w is
  `ctrl+w` on any layout. Send characters when a key's letter matters, as
  in `ctrl+w` to delete a word.
- **A physical key** stays where it is. `ctrl+key_w` is the key where a
  US keyboard has W. Send physical keys when position matters, as with
  WASD.

A key that types no character, such as `enter` or `page_down`, is a
physical key either way. A modifier pressed and released on its own is a
modifier key, `shift`, or its side, `shift_left`: send it on release, and
only when no other key was pressed since the modifier went down. `myelinkey.Parse` reads a name you build from
your event's modifiers and its character or its W3C code value:
`ctrl+w`, `ctrl+KeyW`.

The [key names spec](../myelinkey/SPEC.md) says how each kind is named,
and its conformance vectors test an implementation in any language.

hotty-go names keys the way HOTTY's key names are written now, and
`myelinkey.ParseHotty` reads those names:

<!-- example:myelinkey.ExampleParseHotty -->

```go
for _, name := range []string{"Shift+Control+a", "A", "Control+ ", "Meta+PageDown", "Hyper+x"} {
	k, ok := myelinkey.ParseHotty(name)
	if !ok {
		fmt.Printf("%q is not a key\n", name)
		continue
	}
	fmt.Printf("%q is %s\n", name, k)
}
// Output:
// "Shift+Control+a" is ctrl+shift+a
// "A" is shift+a
// "Control+ " is ctrl+space
// "Meta+PageDown" is super+page_down
// "Hyper+x" is not a key
```

- A HOTTY host's key event carries the name.
- hotty-go's `DecodeKeys` names a terminal's bytes.
- hotty-go's `hottytea.KeyName` names a Bubble Tea v2 key.

Feed the key from your event loop. For a `Deadline`, start a timer that
calls `Expire` with the time it fired; in Bubble Tea that is `tea.Tick`.

## Draw menus

`Keymap.Menu` is the which-key level after a sequence: an item for each
key that can follow it, a group where more keys follow, or the binding
that wins there. `Keymap.Tree` nests every level, for a card of every
binding. Your `applies` function says whether a binding would do anything
now, and a group is enabled when anything under it is. You word a binding
that has no label, and you draw the menu, with `myelinkey.Display` for
the keys.

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

## Help, explain and lint

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

## A widget's keys inside a host

`Mount` puts a widget's document under the node where its host shows the
widget: every bind, group and fence gets the host's context joined to its
own.

## Across a process

The data types have no methods except their generated msgpack codec. A
keymap's documents, a path, a menu or a sequencer state can cross a
process as msgpack maps, through `EncodeMsg` and `DecodeMsg` on a
[msgp](https://github.com/tinylib/msgp) writer and reader. Times are
msgpack timestamps. Menus nest, so give the reader a recursion limit that
fits your deepest menu. `Env`, `Keymap` and `Context` hold compiled CEL,
so they stay in the process that built them.

Decoding is bounded for bytes you didn't write yourself:
- an array holds at most 1,024 elements;
- a map holds at most 256 entries;
- a document holds at most 16,384 bindings.

More fails with `msgp.ErrLimitExceeded` before anything is allocated for
it. Strings and a step's attribute values are bounded by the reader. When
the bytes come from a peer, set `SetMaxStringLength`, `SetMaxElements` and
`SetMaxRecursionDepth` on your `msgp.Reader`.

The full API is on
[pkg.go.dev](https://pkg.go.dev/github.com/neuroplastio/myelin). For what
your users read about writing keymaps, link them to
[Keymaps](keymaps.md).
