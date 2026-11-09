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

A key is its canonical name, in [Ghostty](https://ghostty.org)'s
vocabulary: `ctrl+d`, `shift+g`, `page_down`, `ctrl+key_w`. Two keys are
the same keystroke exactly when their names are equal. A terminal's bytes,
a Bubble Tea key and a hotty host's key event come out as the same name,
so one keymap serves them all.

A key is a character, which follows the keyboard layout, or a physical
key, which stays in its place. The program decides which it sends for
each key. That keeps a key one name: `ctrl+w` means "the key that types
w" on every layout, and a game's `key_w` means the same place on every
layout.

Keys are a package of their own, `neuroplast.io/myelin/myelinkey`,
which needs only the standard library and builds with TinyGo. Whatever
reads input can name keys with it without taking on CEL.

People write keys as they would for Ghostty, which reads modifiers in any
order and case, W3C code values and a letter in either case. A program
that wants one spelling per key reads strictly, and then every other
spelling is refused with the one to use. Either way a listing, a menu and
an error message show a key by its canonical name.

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

Three rules follow from depth:

- **An unbind is a binding to nothing,** and wins the same way. To take
  away a key that a default binds in a context, unbind it in that context.
- **A text field keeps its text.** While a node that takes text has the
  focus, a key that types text goes to it, unless a binding at least as
  deep takes the key. A binding of `x` never stops someone typing an `x`
  into a filter.
- **A mode is a fence.** Where a fence's context holds, it can make that
  rule apply to every key at the fence's depth, so the bindings shallower
  than the mode stop applying. It can also send the keys that nothing
  takes nowhere instead of back to the program. The deepest fence
  decides, as the deepest binding wins, so a palette opened in a mode can
  give the default back.

## Why CEL

Contexts are [CEL](https://cel.dev) expressions, compiled once when the
keymap loads.

- **Checked against the schema.** A misspelled node, an attribute of the
  wrong type or an expression that isn't a boolean fails at load, with the
  names the schema has. A keymap that loads never fails on a key.
- **Fast.** A compiled context evaluates in well under a microsecond, so
  matching a key with contexts costs under half a microsecond, and a key no
  binding starts with costs one map lookup.
- **Partial evaluation.** With some attributes unknown, CEL can still say
  whether an expression might hold. `MayHold` and `MayDepth` use that to
  answer questions about every list, not one list: could this binding hold
  wherever a list has the focus, whatever the list shows?

Contexts run on [cel-lite](https://github.com/neuroplastio/cel-lite),
which implements the part of CEL they need with Go's standard library
alone: no protobuf, no parser generator, and a small program on myelin
is about 2.5 MB. An expression means what CEL's language definition
says. What cel-lite doesn't implement, such as the macros (`exists`,
`all`), `matches`, timestamps and durations, fails at load and says what
it is. Its [language page](https://github.com/neuroplastio/cel-lite/blob/main/docs/language.md)
lists what it holds and what it refuses.

## What it costs

Little enough to ask on every key. A key no binding starts with is one
map lookup, a match in contexts takes under half a microsecond and
allocates nothing, and the leader's menu takes tens of microseconds.
[Performance](performance.md) has the numbers, how they were measured,
and what the gate holds.
