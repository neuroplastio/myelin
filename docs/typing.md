---
group: Guides
order: 20
nav: Typing keys
description: the sequencer's Feed, Expire and Arm, modes that fence keys off, and naming keys from your input
path: typing
---

# Typing keys

A keymap answers what a sequence does. The sequencer is what a program
feeds keys to as they come: it buffers a key that a longer binding
continues, runs the binding a sequence finishes, and gives back the keys
nothing takes. It holds no clock and starts no goroutine, so the program's
own event loop drives it.

## Feed and Expire

`Feed`, `Expire` and `Arm` work on a `Pending` value, which your program
keeps. `Feed` takes a key, the path and the time. It returns the
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
only when no other key was pressed since the modifier went down.
`myelinkey.Parse` reads a name you build from your event's modifiers and
its character or its W3C code value: `ctrl+w`, `ctrl+KeyW`.

The [key names spec](../myelinkey/SPEC.md) says how each kind is named,
and its conformance vectors test an implementation in any language.

hotty-go names keys the way hotty's key names are written now, and
`myelinkey.ParseHotty` reads those names:

<!-- example:myelinkey.ExampleParseHotty -->

```go
for _, name := range []string{
	"Shift+Control+a", "A", "Control+ ", "Meta+PageDown", "Hyper+x",
} {
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

- A hotty host's key event carries the name.
- hotty-go's `DecodeKeys` names a terminal's bytes.
- hotty-go's `hottytea.KeyName` names a Bubble Tea v2 key.

Feed the key from your event loop. For a `Deadline`, start a timer that
calls `Expire` with the time it fired; in Bubble Tea that is `tea.Tick`.

## Next

- [Menus and help](menus.md): what can follow a sequence, as a menu.
- [Keymaps](keymaps.md): how your users write keys and modes.
- [Key names](../myelinkey/SPEC.md): the spec, for implementing key
  names elsewhere.
