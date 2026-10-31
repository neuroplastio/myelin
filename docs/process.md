---
group: Guides
order: 40
nav: Across a process
description: a keymap's documents, paths, menus and the sequencer's state as msgpack, and the limits on reading them
path: process
---

# Across a process

Everything myelin takes and gives back is a plain value, so it can cross
a process. A keymap built in one process can be matched in another, and a
menu can be drawn by a front end written in another language: a web page,
a phone's native sheet, a plugin.

## What crosses

The data types have no methods except their generated
[msgp](https://github.com/tinylib/msgp) codec, `EncodeMsg` and
`DecodeMsg`:

- the schema: `Schema` and `NodeType`;
- the keymap's documents: `Doc`, `Bind`, `Group` and `Fence`;
- where the focus is: `Path` and `Step`;
- the answers: `Binding`, `Match`, `Candidate`, `Problem`, `Menu` and
  `Item`;
- the sequencer: `Pending` and `Result`.

`Env`, `Keymap` and `Context` hold compiled CEL, so they stay in the
process that built them. A process that has the schema and the documents
builds a keymap of its own from them.

## On the wire

Each value is a msgpack map keyed by its fields' names, so a reader in any
language finds a field by name. A deadline is a msgpack timestamp
(extension −1), in UTC. Write values to the connection through a
`msgp.Writer` and read them through a `msgp.Reader`, one value after
another, with no buffer of the whole message in between:

<!-- example:Example_process -->

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
	Binds: []myelin.Bind{
		{Keys: "<leader> n", Action: "file.new", Label: "New file"},
		{Keys: "<leader> q", Action: "quit", Label: "Quit"},
	},
}}, nil)
if km == nil {
	panic(fmt.Sprint(problems))
}

// 1. The program writes the leader's menu to the connection.
var conn bytes.Buffer // a socket or a pipe in a real program
w := msgp.NewWriter(&conn)
leader, _ := km.Alias("leader")
menu := km.Menu(leader, myelin.Path{{Name: "list"}}, nil)
if err := menu.EncodeMsg(w); err != nil {
	panic(err)
}
if err := w.Flush(); err != nil {
	panic(err)
}

// 2. The front end reads it, with limits for bytes it didn't write.
r := msgp.NewReader(&conn)
r.SetMaxRecursionDepth(32)
r.SetMaxStringLength(4096)
var got myelin.Menu
if err := got.DecodeMsg(r); err != nil {
	panic(err)
}
for _, it := range got.Items {
	fmt.Println(myelinkey.Display(it.Key), it.Label)
}
// Output:
// n New file
// q Quit
```

Menus nest, a level inside each group's item in a tree, so give the reader
a recursion limit that fits your deepest menu.

## Limits on reading

Decoding is bounded for bytes you didn't write yourself:

- an array holds at most 1,024 elements;
- a map holds at most 256 entries;
- a document holds at most 16,384 bindings.

More fails with `msgp.ErrLimitExceeded` before anything is allocated for
it. Strings and a step's attribute values are bounded by the reader. When
the bytes come from a peer, set `SetMaxStringLength`, `SetMaxElements` and
`SetMaxRecursionDepth` on your `msgp.Reader`.

## Next

- [Performance](performance.md): what a key, a match and a menu cost.
- The data types' fields are on
  [pkg.go.dev](https://pkg.go.dev/github.com/neuroplastio/myelin).
