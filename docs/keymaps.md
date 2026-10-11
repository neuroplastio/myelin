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
`ctrl+x d`, `<leader> f n`. Keys are named as
[Ghostty](https://ghostty.org/docs/config/keybind) names them.

- **Modifiers** are `super`, `ctrl`, `alt` and `shift`. `super` is the
  command key: Command on macOS, the Windows key elsewhere. They can be
  written in any order and case, and `cmd`, `command`, `opt`, `option` and
  `control` work too. Keys are shown with them in the order
  `super+ctrl+alt+shift+`.
- **A key is a character or a physical key.**
  - A character is what the key types: `a`, `:`, `?`, `ä`. It follows the
    keyboard layout, so `ctrl+z` is Ctrl with whichever key types z.
  - A physical key is a place on the keyboard, named by its [W3C code
    value](https://www.w3.org/TR/uievents-code/) in snake case: `escape`,
    `page_down`, `arrow_up`, `f5`, `key_z`, `digit_1`. `ctrl+key_z` is the
    key where a US keyboard has Z, whatever the layout types there.
  - The W3C spelling works too (`PageDown`, `KeyZ`), and so do some names
    other keymaps use: `esc`, `pgup`, `pgdn`, `pageup`, `pagedown`, `up`,
    `down`, `left` and `right`.
- **Your program sends a key one way or the other.** A character binding
  never matches a physical key, or the other way round. Bind characters
  unless your program's docs say it sends physical keys.
- **Space, Enter, Tab, Backspace and Escape** are physical keys only:
  `space`, `enter`, `tab`, `backspace`, `escape`. A space in a sequence only
  separates keys.
- **Shift with a letter is written out:** `shift+g`, `ctrl+shift+d`.
  Otherwise a letter's case doesn't matter: `G` is `g`, and `ctrl+D` is
  `ctrl+d`. A program can read a capital as Shift with its letter instead,
  as vim does (`G` is `shift+g`); its docs say so.
- **Any other character already includes Shift.** Shift with `/` types
  `?`, so bind `?`. `shift+1` is a different key from `!`, and only a
  program that can't tell what Shift types sends it. The keymap warns you
  about one.
- **`<name>` is an alias** for a sequence that the program or your keymap
  names. `<leader>` is often `space` or `ctrl+x`. A keymap can redefine an
  alias, and the bindings that use it follow.

A program can read keys strictly. Then each key has one spelling, the
canonical one shown above, and any other is refused with that spelling:
`Ctrl+D` asks for `ctrl+d`, and `esc` for `escape`.

### Physical keys

<!-- keys:begin -->

| Write | W3C code value | Menus show |
|-------|----------------|------------|
| `escape` | `Escape` | `escape` |
| `enter` | `Enter` | `enter` |
| `tab` | `Tab` | `⇥` |
| `space` | `Space` | `space` |
| `backspace` | `Backspace` | `backspace` |
| `delete` | `Delete` | `delete` |
| `insert` | `Insert` | `insert` |
| `home` | `Home` | `home` |
| `end` | `End` | `end` |
| `page_up` | `PageUp` | `page_up` |
| `page_down` | `PageDown` | `page_down` |
| `arrow_up` | `ArrowUp` | `↑` |
| `arrow_down` | `ArrowDown` | `↓` |
| `arrow_left` | `ArrowLeft` | `←` |
| `arrow_right` | `ArrowRight` | `→` |
| `f1` … `f25` | `F1` … `F25` | `f1` … `f25` |

Every physical key: `backquote`, `backslash`, `bracket_left`, `bracket_right`, `comma`, `digit_0`, `digit_1`, `digit_2`, `digit_3`, `digit_4`, `digit_5`, `digit_6`, `digit_7`, `digit_8`, `digit_9`, `equal`, `intl_backslash`, `intl_ro`, `intl_yen`, `key_a`, `key_b`, `key_c`, `key_d`, `key_e`, `key_f`, `key_g`, `key_h`, `key_i`, `key_j`, `key_k`, `key_l`, `key_m`, `key_n`, `key_o`, `key_p`, `key_q`, `key_r`, `key_s`, `key_t`, `key_u`, `key_v`, `key_w`, `key_x`, `key_y`, `key_z`, `minus`, `period`, `quote`, `semicolon`, `slash`, `alt_left`, `alt_right`, `backspace`, `caps_lock`, `context_menu`, `control_left`, `control_right`, `enter`, `meta_left`, `meta_right`, `shift_left`, `shift_right`, `space`, `tab`, `convert`, `kana_mode`, `non_convert`, `delete`, `end`, `help`, `home`, `insert`, `page_down`, `page_up`, `arrow_down`, `arrow_left`, `arrow_right`, `arrow_up`, `num_lock`, `numpad_0`, `numpad_1`, `numpad_2`, `numpad_3`, `numpad_4`, `numpad_5`, `numpad_6`, `numpad_7`, `numpad_8`, `numpad_9`, `numpad_add`, `numpad_backspace`, `numpad_clear`, `numpad_clear_entry`, `numpad_comma`, `numpad_decimal`, `numpad_divide`, `numpad_enter`, `numpad_equal`, `numpad_memory_add`, `numpad_memory_clear`, `numpad_memory_recall`, `numpad_memory_store`, `numpad_memory_subtract`, `numpad_multiply`, `numpad_paren_left`, `numpad_paren_right`, `numpad_subtract`, `numpad_separator`, `numpad_up`, `numpad_down`, `numpad_right`, `numpad_left`, `numpad_begin`, `numpad_home`, `numpad_end`, `numpad_insert`, `numpad_delete`, `numpad_page_up`, `numpad_page_down`, `escape`, `f1`, `f2`, `f3`, `f4`, `f5`, `f6`, `f7`, `f8`, `f9`, `f10`, `f11`, `f12`, `f13`, `f14`, `f15`, `f16`, `f17`, `f18`, `f19`, `f20`, `f21`, `f22`, `f23`, `f24`, `f25`, `fn`, `fn_lock`, `print_screen`, `scroll_lock`, `pause`, `browser_back`, `browser_favorites`, `browser_forward`, `browser_home`, `browser_refresh`, `browser_search`, `browser_stop`, `eject`, `launch_app_1`, `launch_app_2`, `launch_mail`, `media_play_pause`, `media_select`, `media_stop`, `media_track_next`, `media_track_previous`, `power`, `sleep`, `audio_volume_down`, `audio_volume_mute`, `audio_volume_up`, `wake_up`, `copy`, `cut`, `paste`.

<!-- keys:end -->

Menus show `shift+tab` as `⇧⇥`.

A key from a terminal, from Bubble Tea or from a HOTTY host is the same
key however it was named, so one keymap serves them all.

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
  That means a character or `space`, with no `super`, `ctrl` or `alt`. It
  doesn't if a binding reaches at least as deep as the field, such as one
  in the context `field`. So binding `x` never stops you typing an `x` into a
  filter.
- **Aliases.** `<leader>`, or any `<name>`, is the alias's keys wherever
  it's written. If more than one layer names an alias, the highest one's
  applies.
- **Dropping the defaults.** A keymap can leave the layers below it out
  entirely, so that every key is yours to bind. Each program's docs say
  how its file writes that.

Each program prints its keymap in force and explains what a key does on a
given path, with its own commands. Its docs say how.

Writing a program? See [Using myelin in a program](integrate.md), and the
API on [pkg.go.dev](https://pkg.go.dev/github.com/neuroplastio/myelin).
