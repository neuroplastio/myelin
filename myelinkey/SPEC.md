# myelin key names

    Version:  0.1
    Status:   Released on 2026-10-17. Versions 0.x may change
              incompatibly (§9).
    Licence:  Apache-2.0, as the rest of this repository, the conformance
              vectors included.

## Abstract

This document gives every keystroke one name, which programs, terminals,
protocols and keymaps can share. A name is the key's modifiers and its
value: a physical key, a character or a modifier key. Two names are the
same keystroke exactly when they are equal strings. Physical keys are the
W3C's code values, and Appendix C says where the other names come from.

It also says how a name written by a person is read, how a sequence of keys
is written and how a key is displayed. It is written so that a protocol,
such as [HOTTY](https://hotty.neuroplast.io), can cite it instead of
defining key names of its own. The Go package
[myelinkey](https://pkg.go.dev/github.com/neuroplastio/myelin/myelinkey) is
its reference implementation, and
[conformance/keys.json](conformance/keys.json) holds its test vectors.

## Contents

1. Conventions
2. Key names
3. Identity and matching
4. Reading written keys
5. Sequences
6. Display
7. Naming input (informative)
8. Conformance
9. Versioning

Appendices: A. Physical keys · B. HOTTY's names before this document
(informative) · C. Prior art (informative)

---

## 1. Conventions

The key words **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT** and **MAY**
are to be interpreted as described in BCP 14 (RFC 2119, RFC 8174) when, and
only when, they appear in bold capitals.

- Names are UTF-8 strings, and they are compared byte for byte.
- **ASCII white space** is U+0009, U+000A, U+000C, U+000D and U+0020.
- A **source** is whatever turns a press into a name: a terminal decoder,
  a GUI toolkit's adapter, a protocol's host.
- A **keymap** binds keys and sequences to actions.

## 2. Key names

### 2.1 Form

A key's name is its modifiers, each followed by `+`, then its value:

    key      = *(modifier "+") value
    modifier = "super" / "ctrl" / "alt" / "shift"
    value    = physical / character / modifier

A name is **canonical** when:
- each modifier appears at most once, in the order `super`, `ctrl`, `alt`,
  `shift`;
- the value is a physical key's name from Appendix A.1 (§2.3), a character
  as §2.4 writes it, or the name of a modifier (§2.5);
- a modifier key's own modifier is not among the modifiers before it.

A **key** is a canonical name. Every other way of writing a key is a
spelling, which a reading (§4) turns into the key or refuses.

To take a key apart, strip the leading `modifier+` words while something
remains after them. What remains is the value. A value is a physical key
if Appendix A.1 lists it, a modifier key if it is one of the four modifier
names, and a character otherwise.

| Key | Modifiers | Value | Kind |
| --- | --- | --- | --- |
| `a` | | `a` | character |
| `shift+g` | shift | `g` | character |
| `ctrl+shift+a` | ctrl, shift | `a` | character |
| `?` | | `?` | character |
| `ctrl++` | ctrl | `+` | character |
| `page_down` | | `page_down` | physical |
| `ctrl+key_w` | ctrl | `key_w` | physical |
| `super+ctrl+alt+shift+enter` | super, ctrl, alt, shift | `enter` | physical |
| `shift` | | `shift` | modifier |
| `ctrl+shift` | ctrl | `shift` | modifier |

### 2.2 Modifiers

- **`super`** is the platform's command key: Command on macOS, the Windows
  key or Super elsewhere. The W3C calls it Meta.
- **`ctrl`** is Control.
- **`alt`** is Alt, which is Option on macOS.
- **`shift`** is Shift (§2.6).

Caps Lock, Num Lock, Scroll Lock, Fn, Hyper and AltGr are not modifiers. A
key pressed with them is named without them. What AltGr does shows in the
character it types: `@` typed with AltGr on a German layout is `@`.

`mod`, the command modifier (§4.4), is no modifier of a key. A reading
turns it into `super` or `ctrl`.

### 2.3 Physical keys

A physical key is a place on the keyboard. Its name is its W3C UI Events
code value ([UIEVENTS-CODE](https://www.w3.org/TR/uievents-code/)) in snake
case: `PageDown` is `page_down` and `KeyW` is `key_w`. Appendix A.1 lists
every one, and the list is normative.
- **A physical key is the same whatever the layout types there.** `key_z`
  is the key where a US keyboard has Z, which types y on a German layout.
- **Some keys are only ever physical.** These are the space bar, Enter,
  Tab, Backspace and Escape (`space`, `enter`, `tab`, `backspace`,
  `escape`), the keys that type no text (`page_down`, `arrow_up`, `f1`),
  and the keypad's keys (`numpad_enter`).
- **A modifier's keys have sides:** `shift_left`, `shift_right`,
  `control_left`, `control_right`, `alt_left`, `alt_right`, `meta_left` and
  `meta_right`. The command key is `meta_*`, as the W3C names it.

### 2.4 Characters

A character is what a key types, as the layout makes it. It is one extended
grapheme cluster ([UAX #29](https://www.unicode.org/reports/tr29/)) whose
first code point is neither U+0020 SPACE nor a control (general category
Cc).
- **Letters are written small:** each code point is replaced by its simple
  lowercase mapping (UnicodeData.txt), as in `a`, `é`, `ǆ`.
- **Everything else is written as it is:** `?`, `+`, `<`, `中`, `👩‍💻`,
  U+00A0 NO-BREAK SPACE.
- **Space and control characters are never characters.** The keys that type
  them are physical keys (`space`, `enter`, `tab`, `backspace`,
  `escape`), and a control character that a terminal sends for Ctrl with a
  letter is that letter with `ctrl` (§7).

### 2.5 Modifier keys

A modifier key is a modifier pressed and released with no other key
pressed in between: a **tap**.
- **Its name is the modifier's,** whichever side it was on: `shift`,
  `ctrl`, `alt`, `super`.
- **A source names a modifier key on release,** because only then is it a
  tap.
- **The modifiers held at the release come before it,** in §2.1's order,
  and the tapped one is last: `ctrl+shift` is Shift tapped while Ctrl is
  held, and `shift+ctrl` is Ctrl tapped while Shift is held. A modifier let
  go during the tap is not held: Ctrl down, Shift down, Ctrl up, Shift up
  is `shift`.
- **A source that knows the side has the sided physical key too**
  (`shift_left`; see §3).
- **A modifier held for a chord is no key.** Shift held while g is pressed
  is `shift+g`, not `shift` followed by `shift+g`.

A keymap binds a double tap of Shift, on either side, as the sequence
`shift shift`.

### 2.6 Shift

- **With a letter, Shift is written as a modifier:** `shift+g`,
  `ctrl+shift+a`. A letter is a character with case: its simple uppercase
  mapping differs from it. The letter is written small whatever case it was
  typed in.
- **Caps Lock changes nothing,** since it isn't a modifier. A letter typed
  with Caps Lock and no Shift is `a`.
- **Any other character already includes Shift.** Shift with / types `?` on
  a US layout, so the key is `?`. A source names the character Shift typed,
  without `shift`.
- **`shift` with a character that has no case** (`shift+1`) is a different
  key from the character Shift types (`!`). A source that can't tell what
  Shift types names the key this way. Keymaps **SHOULD NOT** bind it, and
  an implementation **MAY** warn about one.
- **With a physical key or a modifier key, Shift is a modifier like any
  other:** `shift+tab`, `shift+key_a`, `shift+ctrl`.

## 3. Identity and matching

- **Two keys are the same keystroke exactly when their canonical names are
  equal.**
- **Keys of different kinds never match.** `ctrl+w` is Ctrl with the key
  that types w, and `ctrl+key_w` is Ctrl with the key in the US layout's W
  position. Likewise `shift` is either Shift and `shift_left` is the left
  one.
- **A source that delivers one key per press decides which kind to name.**
- **A source may have two names for one press.** It may have a physical key
  and a character, or a sided modifier and a modifier key. If it looks them
  up in a keymap, it **MUST** try the physical key first and the other
  name second. The first one bound wins.

## 4. Reading written keys

People write keys in config files. A **reading** turns what they wrote into
a key, or refuses it. An implementation **MUST** offer the default reading,
and **MAY** offer strict reading, shifted capitals and the command modifier
as options.

### 4.1 The default reading

1. **Parts.** The text is read from the left, one part at a time. A part
   runs to the next `+`, or to the end, and that `+` is consumed.
   - An empty part is the character `+`: `+` is one empty part, and `ctrl++`
     is `ctrl` and an empty part.
   - Text that ends in `+` after a non-empty part is refused: `ctrl+`, `a+`.
2. **Modifiers.** A part that is a modifier's name or an alias of one, in
   any case, is a modifier:
   - `super`, `ctrl`, `alt` and `shift`;
   - `cmd` and `command` for super, `opt` and `option` for alt, and
     `control` for ctrl.

   A modifier written twice is refused. So is `meta`, which some programs
   use for super and others (Emacs) for alt, and so is `mod`, unless the
   reading says what it stands for (§4.4).
3. **The value.** Any other part is the value, and there is at most one. It
   is read as:
   - **one character** (§2.4), written as §2.4 writes it;
   - otherwise **a physical key:** a name from Appendix A.1 in any case;
     one of the other names in Appendix A.2, in any case; or a W3C code
     value, if it converts to a name in Appendix A.1. To convert, write it in lower case
     and start a new word, after `_`, at each capital letter or digit after
     the first character: `PageDown` is `page_down`, `KeyA` is `key_a`, and
     `Digit1` is `digit_1`.

   Anything else is refused.
4. **No value.** If no part is a value, the last modifier written is the
   value, as a modifier key: `shift`, `ctrl+shift`, `shift+ctrl`.
5. **The key** is the modifiers and the value in canonical form (§2.1).
6. **Aliases.** Text of the form `<name>` is an alias (§5), not a key, and
   is refused.

### 4.2 Strict reading

Strict reading accepts only a key's canonical name and refuses every other
spelling.
- A refusal **SHOULD** give the canonical name: "`Ctrl+d`: write `ctrl+d`".
- Where the text has a capital letter and shifted capitals are off, it
  **SHOULD** give the key with Shift as well: "`G`: write `g`, or `shift+g`
  for Shift with it".

### 4.3 Shifted capitals

Shifted capitals are for keymaps that write keys as vim does. A **capital**
is one code point c whose simple lowercase mapping l differs from c, and
whose simple uppercase mapping of l is c again.
- **A capital reads as `shift` with its small letter:** `G` is `shift+g`,
  and `ctrl+A` is `ctrl+shift+a`.
- **A title-case letter is no capital:** ǅ is `ǆ`.
- **With strict reading also on,** both the canonical name and the spelling
  with the capital are accepted.

### 4.4 The command modifier

`mod` is the platform's modifier for commands. One keymap binds `mod+s`,
and it is Command+s on macOS and Ctrl+s elsewhere. A reading that offers
it is told which modifier `mod` stands for, `super` or `ctrl`. Without
that, `mod` is refused.
- **`mod` reads as the modifier it stands for,** in any case and in any
  place among the others. Where it stands for `ctrl`, `mod+s` is `ctrl+s`,
  `Mod+Shift+ArrowLeft` is `ctrl+shift+arrow_left`, and `mod` alone is the
  modifier key `ctrl` (§4.1, step 4).
- **`mod` with `super` or `ctrl` is refused,** or with an alias of either,
  whichever one `mod` stands for. So a keymap reads on every platform or
  on none. `mod` written twice is refused, as any modifier is.
- **No key is named with `mod`.** The key has the modifier `mod` stands
  for, and a source never names a press with it.
- **With strict reading also on,** the canonical name is accepted, and so
  is the canonical name with `mod` in the place of the modifier it stands
  for: `mod+shift+s`, `alt+mod`. With shifted capitals on as well, so is
  the spelling with the capital, with `mod` in that place: `mod+S`.
- **What `mod` stands for is the program's to say,** or a protocol's. It
  is usually `super` on macOS and `ctrl` elsewhere. A HOTTY host reports
  it as `mod`.

## 5. Sequences

A sequence is keys typed one after another.
- **Its keys are separated by ASCII white space:** `g g`, `ctrl+x d`,
  `shift shift`. Other white space, such as U+00A0, is part of a key.
- **Its canonical form** separates the keys with one U+0020.

**Aliases.** A word of the form `<name>` stands for an alias: a sequence
that a keymap names and puts in the word's place, as in `<leader> f`.
- **The name** is one or more ASCII letters, digits, `_` and `-`.
- **An alias's keys** name no alias.
- **An alias is not a key.** An implementation without aliases refuses
  one, and so does a reading of one key (§4.1).
- **The characters `<` and `>`** on their own are keys.

## 6. Display

A menu or a hint **SHOULD** show a key by its canonical name, except for
the keys in Appendix A.3, which it shows by their symbols. Other modifiers
are written before them as usual: `ctrl+←`, `ctrl+⇧⇥`. A program **MAY**
show keys its own way, for example a capital for `shift` with a letter
where its keymap reads shifted capitals.

## 7. Naming input (informative)

How common sources name a press. A protocol that carries keys, such as
HOTTY (its §10.4), defines how its own hosts name them.

- **A browser's `KeyboardEvent`** gives both kinds of key.
  - Its `code` is the physical key: convert it as §4.1 converts a W3C code
    value.
  - Its `key` is the character, when that is one character (§2.4); a `key`
    of `" "` is `space`.
  - The modifiers are `metaKey` (super), `ctrlKey`, `altKey` and
    `shiftKey`.
  - A `keyup` of a modifier with no other key since its `keydown` is a
    modifier key. Its `code` gives the side, and the other modifiers'
    flags give the ones held (§2.5).
- **A terminal's bytes** carry characters, and the library or protocol that
  reads them decodes them, as HOTTY §10.4 does. The result follows this
  document:
  - the space bar and the control keys are physical keys;
  - Ctrl with a letter is the character with `ctrl` (`ctrl+w`);
  - a capital letter that arrives with no modifier information is `shift`
    with its letter.

  A legacy terminal reports no release, and so no modifier keys. The kitty
  keyboard protocol can report them.

## 8. Conformance

[conformance/keys.json](conformance/keys.json) holds the vectors. An
implementation conforms when it gives every vector's result. Error messages
are not part of conformance.

- **`physical`:** Appendix A.1's names, in order.
- **`read`:** the default reading (§4.1). `in` is the text, and `key` the
  key, or null where it is refused.
- **`strict`:** strict reading (§4.2), as `read`.
- **`shifted_capitals`:** the default reading with shifted capitals
  (§4.3).
- **`strict_shifted_capitals`:** strict reading with shifted capitals.
- **`mod`:** a reading with the command modifier (§4.4). `mod` is what it
  stands for, `super` or `ctrl`. `strict` and `shifted_capitals`, where
  true, turn those options on. `in` and `key` are as in `read`.
- **`split`:** a key taken apart (§2.1): `mods` in order, `value`, and
  `kind`, which is `character`, `physical` or `modifier`.
- **`sequences`:** a sequence read with the default reading and no
  aliases (§5). `keys` is the keys, or null where it is refused.
- **`display`:** a key as a menu shows it (§6).

## 9. Versioning

This is version 0.1, released on 2026-10-17. Versions 0.x may change
incompatibly. A change to any name or rule is a new version, and the
vectors carry the version they test.

---

## Appendix A. Physical keys

### A.1 Names

The physical keys, by the sections of the W3C's code values. Every name
also reads as its W3C code value (§4.1), shown beside it.

<!-- physical:begin -->

**Writing system keys**

| Key | W3C code value |
| --- | --- |
| `backquote` | `Backquote` |
| `backslash` | `Backslash` |
| `bracket_left` | `BracketLeft` |
| `bracket_right` | `BracketRight` |
| `comma` | `Comma` |
| `digit_0` | `Digit0` |
| `digit_1` | `Digit1` |
| `digit_2` | `Digit2` |
| `digit_3` | `Digit3` |
| `digit_4` | `Digit4` |
| `digit_5` | `Digit5` |
| `digit_6` | `Digit6` |
| `digit_7` | `Digit7` |
| `digit_8` | `Digit8` |
| `digit_9` | `Digit9` |
| `equal` | `Equal` |
| `intl_backslash` | `IntlBackslash` |
| `intl_ro` | `IntlRo` |
| `intl_yen` | `IntlYen` |
| `key_a` | `KeyA` |
| `key_b` | `KeyB` |
| `key_c` | `KeyC` |
| `key_d` | `KeyD` |
| `key_e` | `KeyE` |
| `key_f` | `KeyF` |
| `key_g` | `KeyG` |
| `key_h` | `KeyH` |
| `key_i` | `KeyI` |
| `key_j` | `KeyJ` |
| `key_k` | `KeyK` |
| `key_l` | `KeyL` |
| `key_m` | `KeyM` |
| `key_n` | `KeyN` |
| `key_o` | `KeyO` |
| `key_p` | `KeyP` |
| `key_q` | `KeyQ` |
| `key_r` | `KeyR` |
| `key_s` | `KeyS` |
| `key_t` | `KeyT` |
| `key_u` | `KeyU` |
| `key_v` | `KeyV` |
| `key_w` | `KeyW` |
| `key_x` | `KeyX` |
| `key_y` | `KeyY` |
| `key_z` | `KeyZ` |
| `minus` | `Minus` |
| `period` | `Period` |
| `quote` | `Quote` |
| `semicolon` | `Semicolon` |
| `slash` | `Slash` |

**Functional keys**

| Key | W3C code value |
| --- | --- |
| `alt_left` | `AltLeft` |
| `alt_right` | `AltRight` |
| `backspace` | `Backspace` |
| `caps_lock` | `CapsLock` |
| `context_menu` | `ContextMenu` |
| `control_left` | `ControlLeft` |
| `control_right` | `ControlRight` |
| `enter` | `Enter` |
| `meta_left` | `MetaLeft` |
| `meta_right` | `MetaRight` |
| `shift_left` | `ShiftLeft` |
| `shift_right` | `ShiftRight` |
| `space` | `Space` |
| `tab` | `Tab` |
| `convert` | `Convert` |
| `kana_mode` | `KanaMode` |
| `non_convert` | `NonConvert` |

**The control pad**

| Key | W3C code value |
| --- | --- |
| `delete` | `Delete` |
| `end` | `End` |
| `help` | `Help` |
| `home` | `Home` |
| `insert` | `Insert` |
| `page_down` | `PageDown` |
| `page_up` | `PageUp` |

**The arrow pad**

| Key | W3C code value |
| --- | --- |
| `arrow_down` | `ArrowDown` |
| `arrow_left` | `ArrowLeft` |
| `arrow_right` | `ArrowRight` |
| `arrow_up` | `ArrowUp` |

**The numpad**

| Key | W3C code value |
| --- | --- |
| `num_lock` | `NumLock` |
| `numpad_0` | `Numpad0` |
| `numpad_1` | `Numpad1` |
| `numpad_2` | `Numpad2` |
| `numpad_3` | `Numpad3` |
| `numpad_4` | `Numpad4` |
| `numpad_5` | `Numpad5` |
| `numpad_6` | `Numpad6` |
| `numpad_7` | `Numpad7` |
| `numpad_8` | `Numpad8` |
| `numpad_9` | `Numpad9` |
| `numpad_add` | `NumpadAdd` |
| `numpad_backspace` | `NumpadBackspace` |
| `numpad_clear` | `NumpadClear` |
| `numpad_clear_entry` | `NumpadClearEntry` |
| `numpad_comma` | `NumpadComma` |
| `numpad_decimal` | `NumpadDecimal` |
| `numpad_divide` | `NumpadDivide` |
| `numpad_enter` | `NumpadEnter` |
| `numpad_equal` | `NumpadEqual` |
| `numpad_memory_add` | `NumpadMemoryAdd` |
| `numpad_memory_clear` | `NumpadMemoryClear` |
| `numpad_memory_recall` | `NumpadMemoryRecall` |
| `numpad_memory_store` | `NumpadMemoryStore` |
| `numpad_memory_subtract` | `NumpadMemorySubtract` |
| `numpad_multiply` | `NumpadMultiply` |
| `numpad_paren_left` | `NumpadParenLeft` |
| `numpad_paren_right` | `NumpadParenRight` |
| `numpad_subtract` | `NumpadSubtract` |
| `numpad_separator` | `NumpadSeparator` |
| `numpad_up` | `NumpadUp` |
| `numpad_down` | `NumpadDown` |
| `numpad_right` | `NumpadRight` |
| `numpad_left` | `NumpadLeft` |
| `numpad_begin` | `NumpadBegin` |
| `numpad_home` | `NumpadHome` |
| `numpad_end` | `NumpadEnd` |
| `numpad_insert` | `NumpadInsert` |
| `numpad_delete` | `NumpadDelete` |
| `numpad_page_up` | `NumpadPageUp` |
| `numpad_page_down` | `NumpadPageDown` |

**The function section**

| Key | W3C code value |
| --- | --- |
| `escape` | `Escape` |
| `f1` | `F1` |
| `f2` | `F2` |
| `f3` | `F3` |
| `f4` | `F4` |
| `f5` | `F5` |
| `f6` | `F6` |
| `f7` | `F7` |
| `f8` | `F8` |
| `f9` | `F9` |
| `f10` | `F10` |
| `f11` | `F11` |
| `f12` | `F12` |
| `f13` | `F13` |
| `f14` | `F14` |
| `f15` | `F15` |
| `f16` | `F16` |
| `f17` | `F17` |
| `f18` | `F18` |
| `f19` | `F19` |
| `f20` | `F20` |
| `f21` | `F21` |
| `f22` | `F22` |
| `f23` | `F23` |
| `f24` | `F24` |
| `f25` | `F25` |
| `fn` | `Fn` |
| `fn_lock` | `FnLock` |
| `print_screen` | `PrintScreen` |
| `scroll_lock` | `ScrollLock` |
| `pause` | `Pause` |

**Media keys**

| Key | W3C code value |
| --- | --- |
| `browser_back` | `BrowserBack` |
| `browser_favorites` | `BrowserFavorites` |
| `browser_forward` | `BrowserForward` |
| `browser_home` | `BrowserHome` |
| `browser_refresh` | `BrowserRefresh` |
| `browser_search` | `BrowserSearch` |
| `browser_stop` | `BrowserStop` |
| `eject` | `Eject` |
| `launch_app_1` | `LaunchApp1` |
| `launch_app_2` | `LaunchApp2` |
| `launch_mail` | `LaunchMail` |
| `media_play_pause` | `MediaPlayPause` |
| `media_select` | `MediaSelect` |
| `media_stop` | `MediaStop` |
| `media_track_next` | `MediaTrackNext` |
| `media_track_previous` | `MediaTrackPrevious` |
| `power` | `Power` |
| `sleep` | `Sleep` |
| `audio_volume_down` | `AudioVolumeDown` |
| `audio_volume_mute` | `AudioVolumeMute` |
| `audio_volume_up` | `AudioVolumeUp` |
| `wake_up` | `WakeUp` |

**Legacy keys**

| Key | W3C code value |
| --- | --- |
| `copy` | `Copy` |
| `cut` | `Cut` |
| `paste` | `Paste` |

<!-- physical:end -->

### A.2 Other names

The default reading also reads these names as physical keys, in any case
(§4.1).

| Other name | Key |
| --- | --- |
| `up`, `down`, `left`, `right` | `arrow_up`, `arrow_down`, `arrow_left`, `arrow_right` |
| `pageup`, `pgup` | `page_up` |
| `pagedown`, `pgdn` | `page_down` |
| `esc` | `escape` |

### A.3 Display

A menu or a hint shows these keys by their symbols (§6).

| Key | Shown |
| --- | --- |
| `arrow_left`, `arrow_right`, `arrow_up`, `arrow_down` | `←`, `→`, `↑`, `↓` |
| `tab` | `⇥` |
| `shift+tab` | `⇧⇥` |

## Appendix B. HOTTY's names before this document (informative)

HOTTY 0.2 and 0.3 write keys in their own way (§10.4). myelinkey's
`ParseHotty` reads those names into keys while HOTTY moves to this
document. HOTTY 0.3 adds `Mod` to bindings, which this document reads as
`mod` (§4.4). No pressed key is named with it, so `ParseHotty` refuses it.

| HOTTY 0.2, 0.3 | Key |
| --- | --- |
| `a` | `a` |
| `A` | `shift+a` |
| `Control+a` | `ctrl+a` |
| `Control+A`, `Shift+Control+a` | `ctrl+shift+a` |
| `!` | `!` |
| `Shift+1` | `shift+1` |
| `Meta+Enter` | `super+enter` |
| `PageDown` | `page_down` |
| `ArrowLeft` | `arrow_left` |
| `F12` | `f12` |
| `ContextMenu` | `context_menu` |
| `Space`, a space | `space` |
| `Control+ ` | `ctrl+space` |
| `Control++` | `ctrl++` |
| `Shift` | `shift` |
| `Control+Shift` | `ctrl+shift` |

## Appendix C. Prior art (informative)

- **[Ghostty's key bindings](https://ghostty.org/docs/config/keybind)**
  gave the vocabulary:
  - the modifier names and their aliases;
  - the W3C code values in snake case;
  - one character as a key, with case folded;
  - the order in which a press with several names is looked up.
- **The W3C UI Events code values** name the physical keys.
- **VS Code, Zed and Helix** gave the other names for physical keys (`esc`,
  `pgdn`, `up`).
- **vim** writes Shift with a letter as a capital, which shifted capitals
  read.
- **IntelliJ's** "Search Everywhere" is a double tap of Shift, which
  modifier keys write as `shift shift`.
- **HOTTY's `Mod`,** CodeMirror's and ProseMirror's `Mod-` and Electron's
  `CommandOrControl` gave the command modifier, `mod`.
