---
group: Reference
order: 10
nav: Keymaps
description: how keys are written, what a context is, and which binding wins
path: keymaps
---

# Keymaps: keys, contexts and what wins

Programs that take their keys from myelin read a keymap the same way. Each
one reads its keymap from its own config file, in its own format, and its
docs say what that file looks like, which actions it has and what its
contexts are called. This page is what they share: how a key is written,
what a context is, and which binding wins.

## Keys

A key is written as its modifiers, then the key: `ctrl+d`, `alt+x`,
`ctrl+alt+delete`. A sequence is keys separated by spaces: `g g`,
`ctrl+x d`, `<leader> f n`.

Every key has exactly one spelling:

- **Modifiers** are `ctrl`, `alt`, `meta` and `shift`, in that order and in
  lower case: `ctrl+alt+x`, not `alt+ctrl+x`.
- **A character is itself:** `a`, `:`, `!`, `ä`. Write the character the
  key types: `!`, not `shift+1`.
- **A capital letter is written as the capital,** with no `shift+`: `G`,
  not `shift+g`. With another modifier, write `shift+` and the small
  letter: `ctrl+shift+d`, not `ctrl+D`.
- **Named keys** are written as in the table below. A named key that isn't
  in it is written as the browser names it: `ContextMenu`,
  `ctrl+PrintScreen`.
- **A modifier alone** is that modifier's key: `shift`, or `ctrl+shift`
  for Control and Shift together.
- **Space is `space`.** A space in a sequence only separates keys.
- **`<leader>`** stands for the program's leader key, such as `space` or
  `ctrl+x`. A keymap can name its own leader, and `<leader>` follows it.

A key written any other way is refused, with the spelling to use:
`shift+h` asks for `H`, `Escape` for `esc`, `alt+ctrl+x` for `ctrl+alt+x`,
`F5` for `f5`.

`shift+1` is a key apart from `!`, which a few terminals send when they
report Shift with the unshifted key. Bind `!`, the key as you type it,
unless your program says otherwise.

### Named keys

<!-- keys:begin -->

| Write | Key | Menus show |
|-------|-----|------------|
| `alt` | Alt | `alt` |
| `backspace` | Backspace | `backspace` |
| `ctrl` | Control | `ctrl` |
| `delete` | Delete | `delete` |
| `down` | ArrowDown | `↓` |
| `end` | End | `end` |
| `enter` | Enter | `enter` |
| `esc` | Escape | `esc` |
| `home` | Home | `home` |
| `insert` | Insert | `insert` |
| `left` | ArrowLeft | `←` |
| `meta` | Meta | `meta` |
| `pgdn` | PageDown | `pgdn` |
| `pgup` | PageUp | `pgup` |
| `right` | ArrowRight | `→` |
| `shift` | Shift | `shift` |
| `space` | Space | `space` |
| `tab` | Tab | `⇥` |
| `up` | ArrowUp | `↑` |
| `shift+tab` | Shift+Tab | `⇧⇥` |
| `f1`, `f2`, … | F1, F2, … | `f1`, `f2`, … |

<!-- keys:end -->

The middle column is the key's name in the [HOTTY
protocol](https://hotty.neuroplast.io), which is how myelin compares keys.
So a key from a terminal, from Bubble Tea or from a HOTTY host is the same
key.

## Contexts

A program describes where the focus is as a **path**, from the outside
in. In a file manager it might be `window > list`, or
`window > list > field` while a filter is open over the list. Each
program's docs list its nodes and their attributes.

A **context** is a [CEL](https://cel.dev) expression over the path. A
binding with a context works only where the expression holds:

- a node's name is true while the node is on the path: `list`,
  `list && !field`;
- a node's attributes are `node.attribute`: `list.dir == '/tmp'`,
  `list.dir.startsWith('/home')`, `buffer.tags['lang'] == 'go'`;
- `focus` is the name of the innermost node: `list` holds even with a
  dialog open over the list, and `focus == 'list'` only while the list
  itself has the keys;
- strings are in single quotes: `'/tmp'`.

An expression that reads an attribute of a node that isn't on the path
doesn't hold: `list.dir == '/tmp'` is false with no list open. A binding
with no context works everywhere.

A mistake in a context fails when the keymap loads, not when you press the
key:
- a name the program doesn't declare (the error lists the names it does);
- a type that doesn't fit, such as `list.dir == 1`;
- an expression that isn't true or false.

## What wins

When more than one binding of the same keys can run, the winner is:

1. **The binding whose context reaches deepest.** A context's depth is how
   far into the path it has to look to hold. On `window > list > field`,
   `field` reaches depth 3 and `list` depth 2, and a binding with no
   context is depth 0. So a key bound in the context `field` beats the
   same key bound everywhere, but only while the field is open. A context
   that only says what isn't there, such as `!dialog`, adds no depth.
2. **At the same depth, the higher layer.** Your keymap is above the
   defaults. A program may have more layers, such as extensions below its
   defaults.
3. **In the same layer, the later binding.**

Some keys need more than one binding or a binding to nothing:

- **Sequences.** When a key finishes one binding and starts a longer one,
  the program waits for the next key. If none comes in time (each program
  sets how long), the shorter binding runs. A longer binding waits like
  this only if it reaches at least as deep as the shorter one. If it
  doesn't, the shorter binding runs at once.
- **Unbinding** binds keys to nothing, and wins the way a binding does. To
  take away a key that the defaults bind in a context, unbind it in that
  context. Unbinding a key with no context leaves the default's deeper
  binding in place, and the keymap warns you and names the context.
  Unbinding `g` also unbinds every `g …` sequence it beats by these rules:
  the ones bound no deeper than the unbind.
- **Text fields keep their text.** In a program with text fields, such as
  a filter or a search box, a key that types text goes into the field
  while one is open.
  That means a character or `space`, with no `ctrl`, `alt` or `meta`. It
  doesn't if a binding reaches at least as deep as the field, such as one
  in the context `field`. So binding `x` never stops you typing an `x` into a
  filter.
- **The leader.** `<leader>` is the leader key wherever it's written. If
  more than one layer names a leader, the highest one's applies.
- **Dropping the defaults.** A keymap can leave the layers below it out
  entirely, so that every key is yours to bind. Each program's docs say
  how its file writes that.

Each program prints its keymap in force and explains what a key does on a
given path, with its own commands. Its docs say how.

Writing a program? See [Using myelin in a program](integrate.md), and the
API on [pkg.go.dev](https://pkg.go.dev/github.com/neuroplastio/myelin).
