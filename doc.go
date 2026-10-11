// Package myelin is a key-binding library for Go programs: keymaps whose
// bindings hold in contexts, key sequences, and which-key menus, all as
// plain data.
//
// A program reads its keymap from its own files, in its own format, and
// hands it to myelin as values: one [Doc] per layer, holding [Bind] and
// [Group] nodes. It gets data back: what a sequence runs ([Match]), what can
// follow it ([Menu]), why a binding wins ([Candidate]), and the state of a
// sequence in progress ([Pending]). myelin reads no files and knows no file
// format. It does no IO, reads no clock and starts no goroutine.
//
// # Keys
//
// A [myelinkey.Key] is a keystroke's canonical name, in Ghostty's
// vocabulary: "ctrl+d", "shift+g", "page_down", "ctrl+key_w", "?". A key is
// a character, which follows the keyboard layout, or a physical key, named
// by its W3C code value, which stays in its place; the program decides which
// it sends. Two keys are the same keystroke exactly when they are ==,
// whether they came from a terminal, from Bubble Tea or from a HOTTY host.
//
// People write keys as they would for Ghostty: modifiers in any order and
// case, W3C code values, a letter in either case. A [Schema] can read them
// strictly instead, with one spelling per key (StrictKeys), or read a
// capital as Shift with its letter, as vim does (ShiftedCapitals). A
// sequence is keys separated by spaces, "g g", and "<name>" stands for an
// alias's keys, such as "<leader>" ([Schema].Aliases, [Doc].Aliases).
// [myelinkey.Display] writes a key for a menu, with glyphs for the arrows
// and Tab.
//
// # Paths and contexts
//
// A program declares what can be focused in a [Schema]: kinds of node,
// each with typed attributes, and the keymap's layers. At any moment the
// focus is a [Path]: the focused node and the nodes around it, outermost
// first, such as a window, a list in it, and a filter field over the list.
//
// A context is a CEL expression (https://cel.dev) over the path. A node's
// name is true while the node is on the path, its attributes are
// node.attribute, and focus is the innermost node's name:
//
//	list && !field
//	list.dir.startsWith('/tmp')
//	focus == 'list'
//
// [Env.Compile] checks a context against the schema, so a mistake fails
// when the keymap loads and a keymap that loads never fails on a key.
//
// # Precedence
//
// When more than one binding of the same keys holds on a path, the winner
// is:
//
//  1. the one whose context reaches deepest into the path ([Context.Depth]);
//  2. at the same depth, the one in the higher layer;
//  3. in the same layer, the later one.
//
// An unbind is a binding to nothing, and wins the same way. A key that types
// text goes to a node that takes text ([NodeType].Text) unless a binding at
// least as deep as that node takes it.
//
// # Building and querying
//
// [NewEnv] compiles a schema into an [Env], and [Env.Build] resolves a
// program's documents into a [Keymap], checking each binding with the
// program's own rules. A keymap answers questions about a path:
// [Keymap.Match], [Keymap.Effective], [Keymap.KeysFor], [Keymap.KeyFor],
// [Keymap.Explain] and [Keymap.Shadowed].
//
// [Keymap.Menu] is the which-key level after a sequence: an [Item] for each
// key that can follow it, a group or the binding that wins there.
// [Keymap.Tree] nests every level under it. Both are data that any front
// end can draw.
//
// # The sequencer
//
// [Feed], [Expire] and [Arm] are pure functions over [Pending]. A key and
// the time go in. The new state and a [Result] come out: a binding to run,
// keys that matched nothing, or a deadline. The program keeps the state,
// and when a deadline comes, its own event loop calls Expire.
//
// # Data and handles
//
// The data types have exported fields and no methods except their
// generated msgpack codec (EncodeMsg and DecodeMsg, from
// github.com/tinylib/msgp), so a keymap, a path or a menu can cross a
// process as a msgpack map. Times are msgpack timestamps. [Env], [Keymap] and
// [Context] are handles: they hold compiled CEL, are never serialized, and
// are safe for concurrent use.
//
// Keys are named in the package github.com/neuroplastio/myelin/myelinkey, a
// module of its own that needs only the standard library and builds with
// TinyGo, so a program or a library that reads input can name keys without
// taking on CEL.
//
// docs/keymaps.md in the repository explains, for people writing keymaps,
// how keys are spelled, how contexts work and which binding wins.
package myelin
