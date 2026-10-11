---
group: Concepts
order: 10
nav: How myelin works
description: why the keymap is data, why time is a value, and how depth and layers decide
path: design
---

# How myelin works

myelin makes a few choices that shape how a program uses it. This page says
what they are and why.

## The keymap is data

A program hands myelin its keymap as values (documents of bindings) and
gets values back: a match, a menu, an explanation, the state of a sequence.
myelin never reads a file.

- **Each program keeps its own format.** One reads KDL, another YAML,
  another a plugin's manifest. They all build the same documents, so
  keys, contexts and precedence mean the same in all of them.
- **Everything can be tested as values.** A keymap, a path and a key go
  in, and the answer comes out, with no terminal or file system involved.
- **Values can cross a process.** Every data type has a generated msgpack
  codec, so a keymap built in one process can be matched in another, or a
  menu drawn by a front end in another language.

Only three types hold more than data: `Env`, `Keymap` and `Context` hold
compiled CEL. They are built once and are safe for concurrent use.

## Time is a value

A key sequence such as `g g` has to wait for its second key, and give up
after a while. myelin doesn't start a timer for that. `Feed` takes the
current time and returns a deadline, and the program wakes itself at the
deadline and calls `Expire`.

- **The program owns its event loop.** A Bubble Tea program uses
  `tea.Tick`, a server a timer in its `select`, a web page `setTimeout`.
  myelin fits all of them because it has no loop of its own.
- **Tests need no clock.** Pass any time and the result is the same every
  run.
- **A stale wake-up is harmless.** Expire before the deadline does
  nothing.

## A key is a name

A key is its canonical name in the [HOTTY](https://hotty.neuroplast.io)
protocol: `Control+d`, `PageDown`, `A`. A terminal's bytes, a Bubble Tea
key and a HOTTY host's key event come out as the same name, so one keymap
serves them all.

Keys are a package of their own, `github.com/neuroplastio/myelin/myelinkey`,
which needs only the standard library and builds with TinyGo. Whatever
reads input can name keys with it without taking on CEL.

People write keys in one short spelling, `ctrl+d`, `pgdn`, `H`, and every
key has exactly one. A key written another way is refused with the
spelling to use. With one spelling, a listing, a menu and an error message
always show a key the way it was written, and two people never write the
same binding two ways.

## Depth, then layer, then order

The focus is a path of nodes, from the outside in, such as
`window > list > field`: the focused node and the nodes around it. A binding's context is an expression over that
path. When several bindings of the same keys hold, the one whose context
reaches deepest wins.

Depth comes first because the deepest context is the most specific. A key
bound for a text field has to beat the same key bound for the whole
window, but only while the field is open. A context that only says what
isn't there, such as `!dialog`, adds no depth, because it says nothing
about where the focus is.

At the same depth, the higher layer wins, so a user's binding overrides a
default that is just as specific. Within a layer, the later binding wins,
as it would in any config file.

Two rules follow from depth:

- **An unbind is a binding to nothing,** and wins the same way. To take
  away a key that a default binds in a context, unbind it in that context.
- **A text field keeps its text.** While a node that takes text has the
  focus, a key that types text goes to it, unless a binding at least as
  deep takes the key. A binding of `x` never stops someone typing an `x`
  into a filter.

## Why CEL

Contexts are [CEL](https://cel.dev) expressions, compiled once when the
keymap loads.

- **Checked against the schema.** A misspelled node, an attribute of the
  wrong type or an expression that isn't a boolean fails at load, with the
  names the schema has. A keymap that loads never fails on a key.
- **Fast.** A compiled context evaluates in well under a microsecond, so
  matching a key with contexts costs about half a microsecond, and a key no
  binding starts with costs one map lookup.
- **Partial evaluation.** With some attributes unknown, CEL can still say
  whether an expression might hold. `MayHold` and `MayDepth` use that to
  answer questions about every list, not one list: could this binding hold
  wherever a list has the focus, whatever the list shows?

CEL has costs. It adds several megabytes to a binary, and it doesn't run
under TinyGo, because it relies on protobuf's reflection.

## What it costs

Measured on a laptop (AMD Ryzen AI 9 HX 370) with a keymap of a hundred
bindings, half of them in contexts:

| Operation | Time |
|-----------|------|
| A key no binding starts with | 36 ns |
| A key matched against bindings with contexts | 0.5 µs |
| The leader's menu | 47 µs |
| Compiling a context, once, at load | 30 µs |

`go test -bench .` in the repository runs these.
