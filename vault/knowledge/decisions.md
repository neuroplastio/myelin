# Decision log

Append-only, newest at the bottom. A decision is a constraint that later
work must not silently contradict. To change one, add a new decision that
supersedes it. How a leg was built belongs in the journal, not here.

D1–D9 are myelin's decisions as they stood when it went public
(2026-10-11), each restated as it stands.

---

### D1 — The module's layout and its gate
**2026-10-10.**
1. **One root package, `myelin`.**
   - It holds the data types (`data.go`, with the codec generated into
     `data_gen.go`), the functions and the three handles.
   - A subpackage exists only to keep a dependency out of the root, and is
     then a module of its own, as `myelintea` is (D5).
2. **The root module depends on hotty-go's root module** (key names), on
   msgp's runtime and on cel-go. Nothing else.
3. **msgp's generator is pinned in `tools/go.mod`** and runs as
   `go tool -modfile=tools/go.mod msgp`. The root go.mod never gets a `tool`
   directive, which would put the generator's dependencies into every
   importer's module graph.
   - The codec flags are `-marshal=false -tests=false`.
   - The generated file carries `//msgp:newtime` and
     `//msgp:timezone utc`, so a time decodes in UTC.
4. **HOTTY's conformance vectors** are copied from the hotty-go that go.mod
   pins (`make vectors`). The gate fails when the copy differs, so bumping
   hotty-go re-checks the key names.
5. **The gate is `scripts/check.sh`**, which `make check` and CI both run:
   - gofmt, vet and test, in both modules;
   - msgp's output is current, compared with the working tree, so
     uncommitted work is checked too;
   - the vectors are current;
   - the docs' generated parts are current (D9).

   It is a script, not Makefile recipes, because the stock runner image
   has no make.

### D2 — Data and handles
**2026-10-10.**
1. **The data types have exported fields, `msg` tags and their generated
   codec, and no other methods.** What works on them is functions.
2. **Three handles have methods,** because they hold compiled CEL: `Env`,
   `Keymap` and `Context`. They are never serialized, and are safe for
   concurrent use.
3. **myelin does no IO, reads no clock and starts no goroutine.** Time goes
   in as `now`, and comes out as a deadline.
4. **No file format.** A program builds the documents from its own files;
   myelin never reads one.

### D3 — Key identity and the one spelling
**2026-10-10.** Approved by the maintainer: "key spelling approved".
1. **A Key is a canonical HOTTY name** (SPEC §10.4), a string. Every Key
   myelin makes goes through hotty-go's `ParseKey`, so two Keys are the
   same keystroke exactly when they are `==`.
2. **Each key has exactly one spelling.** ParseKey accepts a key only as
   FormatKey writes it, and refuses any other form with "write <the
   spelling>": `shift+h` asks for `H`, `alt+ctrl+x` for `ctrl+alt+x`,
   `ctrl+D` for `ctrl+shift+d`, `" "` for `space`, `Escape` for `esc`.
   ParseKey checks its input against FormatKey of the key it read, so the
   rule and its test are one check.
3. **Modifiers** are `ctrl+`, `alt+`, `meta+` and `shift+`, written in
   HOTTY's order and in lower case. A repeated modifier is refused.
4. **Named keys** have short names: `up`, `down`, `left`, `right`, `enter`,
   `esc`, `tab`, `space`, `home`, `end`, `pgup`, `pgdn`, `backspace`,
   `delete`, `insert`. Function keys are spelled by rule, `f<n>`. A named
   key with no short name is written as HOTTY names it (`ContextMenu`). A
   HOTTY name that is a short name in another case (`Esc`, `Up`, `Ctrl`)
   is refused in favour of the short name; no host sends those.
5. **A capital is the shifted letter.** `H` is Shift+h. With another
   modifier, the shift is written and the letter is small: `ctrl+shift+d`.
6. **Every HOTTY name has a spelling.** `shift+1` is Shift+1, a key apart
   from `!`, which a host sends when it has no shifted key to report. A
   modifier alone is that modifier's key: `shift`, `ctrl+shift`.
7. **Any single character HOTTY names is a key,** grapheme clusters
   included. Two or more ASCII letters and digits are always a name, so
   `gg` is refused with a hint to write `g g`.
8. **Sequences split only at ASCII white space,** as HOTTY splits
   `data-keys`, so a no-break space is a key. A literal space is refused:
   write `space`. `<leader>` is the keymap's to resolve.
9. **DisplayKey** shows the arrows as `←→↑↓`, Tab as `⇥` and Shift+Tab as
   `⇧⇥`. Every other key shows its short spelling.
10. **The tests hold all of this:** every name in the conformance vectors
    round-trips, and the fuzzers check that whatever parses is FormatKey's
    spelling.

A change to any of this is a new decision.

### D4 — The data's wire form
**2026-10-10.**
1. **Each data type is a msgpack map keyed by its `msg` tags.**
2. **Binding embeds Bind under the key `bind`:**
   `{"bind": {…}, "seq": […], "layer": "…"}`. msgp doesn't flatten an
   embedded struct.
3. **An absent field is the zero value.** Args absent or empty means no
   values; `[""]` survives the round trip and means one empty value. A
   zero Deadline is left out.
4. **A path's attribute values decode as msgp's interface types:**
   integers as int64, floats as float64, maps as `map[string]any`, lists
   as `[]any`. An attribute declared `int` reads an integer of any Go kind
   (a uint64 that fits included).
5. **Items nest** in a Tree. A program that reads menus from a stream sets
   the reader's recursion limit from the deepest menu it expects.
6. **Times are msgpack timestamps** (extension −1), read back in UTC.

### D5 — myelintea, and how myelin's modules pin each other
**2026-10-10.**
1. **myelintea is a module of its own,**
   `github.com/neuroplastio/myelin/myelintea`, so Bubble Tea stays out of
   the root module. Its one function is `Key(tea.Key) (myelin.Key, bool)`,
   through `hottytea.KeyName`.
2. **It requires a pushed myelin commit** and replaces it with `../` in
   the tree. An importer gets the version it requires, which moves when
   myelintea needs newer API from the root.
3. **myelintea's hottytea is the hotty-go commit the root pins,** so there
   is one way of naming keys. The gate checks it.
4. **CI runs with `GOTOOLCHAIN=auto`.** The root is checked at its go 1.24
   floor, and myelintea fetches its own toolchain.

### D6 — Precedence
**2026-10-10.**
1. **Of the bindings of a sequence whose contexts hold on the path, the
   deepest wins,** then the higher layer, then the later bind.
   - A context's depth is the shortest prefix of the path it holds on,
     given that it holds on the whole path. No context is depth 0, and so
     is a context that holds with nothing on the path (`!dialog`).
2. **A binding that is also a prefix waits** only for longer bindings at
   least as deep as itself, and runs when the sequence times out.
3. **An unbind is a binding to nothing** and wins the same way. `unbind
   "g"` also takes away every `g …` it beats. An unbind with no context
   that can't reach a deeper binding of a lower layer is a warning that
   names the context to unbind it in.
4. **A node that takes text keeps its text:** a sequence that starts with
   a key that types text is the innermost text node's unless a binding at
   least as deep as that node takes it.
5. **Layers:** a document drops only lower layers. The leader is the
   highest layer's that names one, a dropped layer's included, else the
   schema's.

A change to any of this is a new decision.

### D7 — The engine's API
**2026-10-10.**
1. **Build's check receives the Binding,** `func(Binding) error`: the Bind
   with its sequence resolved and its layer named, so a program's rules can
   depend on the layer and the resolved keys. `Warn(err)` loads with a
   warning; any other error refuses the keymap.
2. **Partial evaluation takes a Path.** `MayHold(p)` and `MayDepth(p)`
   treat any attribute a step doesn't give as unknown, so a program can
   ask about every list rather than one.
3. **KeysFor and KeyFor are separate.** `KeysFor(action, args, p)` is every
   sequence that runs the action with exactly those values, in load order,
   for a help listing. `KeyFor(action, p)` is the one binding a command
   palette shows: no values before values, then the shorter, then the
   first.
4. **Effective leaves out a binding a shorter one runs first,** one that
   doesn't wait for it. What Effective lists is what typing those keys
   there runs.
5. **`focus`** is a variable in every context: the innermost step's name.
   Partial evaluation knows it.
6. **Also public:** `(*Keymap).ParseSeq` (a written sequence, `<leader>`
   resolved), `CheckSeq` (a written sequence checked without a keymap),
   `FormatPath`, `Shadowed` (bindings a deeper lower-layer binding beats on
   the paths given), `Explain` (candidates as data), and `Mount`.

### D8 — Going public, and CI
**2026-10-11.**
1. **myelin goes public after its first consumer integrates it locally and
   after the maintainer's final review** ("don't publish without my final
   review").
2. **myelin stands on its own:** no mention of private repositories in the
   code, the docs or the vault.
3. **CI runs on `ubuntu-latest`** and skips its job while the repository
   is private. Until it is public, the gate is a green local `make check`
   before each push, and every journal entry says so.

### D9 — The docs
**2026-10-11.** The maintainer asked for a documentation package; the
format is the org's docs standard, worked out with the Docs Agent.
1. **`README.md` and `docs/`**, read on GitHub. The API reference is the
   godoc, on pkg.go.dev. No man pages: myelin is a library.
2. **Pages:** `docs/integrate.md` (Guides), `docs/keymaps.md` (Reference,
   for people writing keymaps; programs link their users to it) and
   `docs/design.md` (Concepts).
3. **What the code defines is generated and checked:**
   - the named-key table in `keymaps.md` (TestReferenceKeys);
   - every code block after an `<!-- example:NAME -->` marker is the
     Example function NAME, or a whole example file, which `go test`
     compiles and runs (TestDocsExamples).

   `go test -run 'TestReferenceKeys|TestDocsExamples' -update` rewrites
   them.
4. **The godoc stands alone:** every exported identifier is documented,
   with runnable Examples, and no comment points at a private document.

### D10 — A fresh history, the old one dropped
**2026-10-11.** The maintainer: "Just force push, drop the history".
- `main` starts again at one root commit holding the reviewed tree, and a
  second one that points myelintea at it. The old history is not kept,
  locally or on GitHub, by the maintainer's choice rather than by keeping
  an original copy. Commits cited in other repositories from before this
  don't exist any more.
- Another program's test files that myelin no longer carries were handed
  over outside the repository first.
- Nothing is force-pushed again (AGENTS.md).

### D11 — Tools are pinned in mise.toml
**2026-10-11.** The maintainer: "Move the tool into mise toml instead of
this special package". Supersedes D1.3 and D5.4.
1. **msgp's generator is pinned in `mise.toml`,** beside Go:
   `"go:github.com/tinylib/msgp" = "1.6.5"`. `//go:generate` runs it as
   `msgp -marshal=false -tests=false`. There is no `tools/` module, and no
   go.mod has a `tool` directive.
2. **The gate needs mise's tools on PATH** (`mise exec -- make check`), and
   says so when msgp is missing.
3. **CI installs the same tools** with `jdx/mise-action`, runs the gate,
   and then vets and tests the root module under the toolchain go.mod's go
   directive names (`GOTOOLCHAIN=go1.24.0`), the floor importers build
   with.

### D12 — myelin owns the keys; hotty-go depends on myelin
**2026-10-11.** The maintainer: "hotty-go will depend on myelin, as myelin
owns the keys", to be done after myelin goes public. Once done, it
supersedes D1.2's hotty-go requirement and D3.1's "through hotty-go's
`ParseKey`".
1. **The Go code that names keys lives in myelin:** a key's canonical name
   (what hotty-go's `ParseKey` does today), the short spelling and
   sequences. hotty-go requires it; myelin requires nothing of hotty-go.
2. **It is a module of its own with no requirements.** hotty-go's root
   module needs only the standard library and builds under TinyGo.
   Requiring myelin's root would put cel-go, protobuf and ANTLR into every
   hotty-go importer's module graph, even though they would never be linked.
   The module's name is settled when it is built.
3. **The names stay HOTTY's.** SPEC §10.4 defines them, and HOTTY's
   conformance vectors stay the test.
4. **Order, with no module cycle at any step:** myelin first (the keys
   module, the root off hotty-go), then hotty-go moves, then each consumer
   bumps.
5. **What stays in hotty-go is settled with its owner.** The lean:
   `DecodeKeys` (a terminal's bytes), SPEC §10.2's text-field keymaps and
   `hottytea.KeyName` stay there, built on myelin's names.
6. **Settled with the hotty agent** (2026-10-11):
   - **The key module is safe in hotty-go's root.** It has no requirements,
     no reflection-heavy code, no large tables or regexps built at init,
     and ints that fit in 32 bits. It passes `tinygo test` before step 1
     ends.
   - **SPEC §10.4 defines the canonical names, and wins.** If myelin's
     canonical name for a key ever differs from §10.4, the module follows
     §10.4. The short spelling stays myelin's own (D3); it writes §10.4's
     names and isn't part of §10.4.
   - **The module runs HOTTY's vectors** from the hotty repository's
     `conformance/vectors.json`: all of `keys`, and the key-name part of
     `keymap`. The text-field keymaps themselves stay in hotty-go.
   - **hotty-go keeps the rest:** `ParseKey`, as a thin wrapper that
     delegates; `DecodeKeys`; `Keymap`, `ParseKeymap`, `Resolve` and
     `TerminalKeys`; `hottytea.KeyName` and `hottyterm.KeyName`.
   - **Step 1 also waits for HOTTY's msgpack release** (hotty-go's move
     to msgpack). myelin messages the hotty agent before it starts.

### D13 — Keys are a module of their own; no Bubble Tea adapter
**2026-10-11.** The maintainer: "Make the key a submodule. Also, drop
myelintea package, it doesn't belong in a headless library. Let's do this
before publishing". This brings D12's step 1 before going public. It
supersedes D5, D1.2 and D1.4, and D3.1's "through hotty-go's `ParseKey`".
1. **`github.com/neuroplastio/myelin/key`** is a module in `key/`, package
   `key`. It holds `Key`, the canonical names (`ParseHotty`, ported from
   hotty-go's `ParseKey`), the short spelling (`Parse`, `Format`,
   `Display`, `Named`), sequences (`ParseSeq`, `FormatSeq`, `SplitSeq`) and
   `TypesText`. `<leader>` stays the keymap's: the root keeps `CheckSeq` and
   `(*Keymap).ParseSeq`.
2. **`key` requires nothing,** and its tests pass under TinyGo's wasm with
   64 KB stacks, which the gate runs (TinyGo is pinned in mise.toml). Its
   vectors are HOTTY's, from the hotty repository at the commit in
   `key/testdata/conformance/hotty-commit`, embedded in the tests so that
   they run without a file system.
3. **The root requires `key` at a pushed commit** and replaces it with
   `./key` in the tree. A change to `key` is pushed before the root's
   requirement moves to it.
4. **The root requires `key`, cel-go and msgp's runtime,** and no longer
   hotty-go. Its data types carry a key as `key.Key`, a msgpack string
   (`//msgp:replace key.Key with:string`); the wire form doesn't change.
5. **myelin is headless: myelintea is gone,** and no UI toolkit's adapter
   lives in myelin. A program names its toolkit's keys (for Bubble Tea,
   hotty-go's `hottytea.KeyName`) and passes the name to `key.ParseHotty`.
6. **CI checks both modules at the root's go directive,** after the gate.

### D14 — The keys module is myelinkey
**2026-10-11.** The maintainer: "key package should have "myelinkey" name,
"key" is a bad import name". Supersedes D13's name for the module; the
rest of D13 stands.
- The module is `github.com/neuroplastio/myelin/myelinkey`, in
  `myelinkey/`, and its package is `myelinkey`, so the import path and the
  package name agree. Its API is D13's under the new name:
  `myelinkey.Key`, `myelinkey.Parse`, `myelinkey.ParseHotty`, and so on.
- The `key` module (5790483) stays in the history only; nothing requires
  it once consumers move.

### D15 — Ghostty's key names: characters, physical keys and aliases
**2026-10-11.** The maintainer: "we should follow ghostty's decisions here,
they are very sound"; "myelin should support both, it's up to the app to
decide which ones to use / send"; "<leader> should be a registered key
sequence … Just having <leader> as a magic string is bad"; "let's not
refuse. For stricter parsing, we should use configuration options"; "I
would change HOTTY spec to match what we are working on here". Supersedes
D3 whole, D12.3 and D12.6's "§10.4 wins" (HOTTY's names now follow
myelinkey's), and D13.1's `Format`.
1. **A Key is a canonical name in Ghostty's vocabulary,** a string: its
   modifiers as `super+ctrl+alt+shift+`, in that order, then a physical key
   or a character. Two Keys are the same keystroke exactly when they are
   `==`. `super` is the command key: Command, or Windows or Super.
2. **Physical keys are Ghostty's names:** the W3C UI Events code values in
   snake case (`escape`, `page_down`, `arrow_up`, `key_a`, `digit_1`,
   `f1`–`f25`, `numpad_enter`, `context_menu`). A modifier key alone is a
   sided key, such as `shift_left`.
3. **A character is one grapheme cluster** that isn't white space or a
   control, and a letter is written small. Space, Enter, Tab, Backspace
   and Escape are physical keys only.
4. **One key per press, physical or character: the program's choice.**
   myelin never matches one kind against the other. Where something sees
   both for one press and a keymap binds both, the physical binding wins,
   as in Ghostty. Feed takes one key; one that takes a press with both
   kinds may come later, alongside it.
5. **Shift with a letter is a modifier:** `shift+g`, `ctrl+shift+a`.
   Otherwise a letter's case doesn't count, so Caps Lock without Shift is
   `a`. Any other character shows Shift itself: Shift with / is `?`.
   `shift+1` is a key apart, which a program sends only when it can't tell
   what Shift types, and Build warns of one in a binding.
6. **The default reading is Ghostty's.** It reads:
   - modifiers in any order and case, with Ghostty's aliases (`cmd`,
     `command`, `opt`, `option`, `control`);
   - W3C code values (`PageDown`, `KeyA`);
   - a letter in either case;
   - other keymaps' names: `esc`, `pgup`, `pgdn`, `pageup`, `pagedown`,
     `up`, `down`, `left` and `right`.

   It refuses a repeated modifier, and `meta`, which could mean `super` or
   Emacs's Alt: the error names both.
7. **Two options, in the Schema.**
   - `StrictKeys` reads only canonical names, and refuses any other
     spelling with the canonical one.
   - `ShiftedCapitals` reads a capital as Shift with its letter, as vim
     does: `G` is `shift+g`.
8. **Aliases.** `<name>` in a sequence stands for a whole sequence, from
   `Schema.Aliases` or `Doc.Aliases`.
   - For each name, the highest layer's alias wins, then the later
     document's, then the schema's. A dropped layer's aliases go with it.
   - A name is ASCII letters, digits, `_` and `-`. An alias's keys can't
     name an alias, and an unknown alias is a build error.
   - `<leader>` is just an alias. `Keymap.Alias(name)` and
     `Arm(km, name, …)` replace `Leader()`.
   - `CheckSeq` becomes `Env.CheckSeq`, which reads keys with the schema's
     options and takes an alias on trust.
9. **Display** is the canonical name, with glyphs for the arrows and Tab:
   `ctrl+←`, `⇥`, `⇧⇥`.
10. **HOTTY follows.** SPEC §10.4 is amended in place to these names. The
    maintainer, relayed by the hotty agent: "Keep your work in 0.2, do not
    bump the spec. Breaking changes are fine as we control the whole stack
    today".
    - Until then, `ParseHotty` reads §10.4's names onto these keys:
      `Control+A` is `ctrl+shift+a`, `PageDown` is `page_down`, `Meta` is
      `super`, and a lone `Shift` is `shift_left`. The vectors test that
      mapping.
    - When §10.4 moves, `ParseHotty` goes, and the vectors test the names
      directly.
11. **myelinkey's API:**
    - `Parse`, and `ParseWith(s, Options)`;
    - `Split(k)`, returning the modifiers, the value and whether the value
      is a character;
    - `Mods`, with `Super`, `Ctrl`, `Alt` and `Shift`;
    - `Named`, which lists the physical keys.

    `Format` is gone, because a Key is its own spelling.

### D16 — Modifier keys, and the key names as a spec
**2026-10-11.** The maintainer asked "Does myelin support pure modifier
keys? Something like double-shift", then chose unsided tap keys. They also
said "Need a key reference document. Should be published as a spec that
HOTTY can refer to, v0.1". This supersedes D15.2's "a modifier key alone is
a sided key", D15.10's lone `Shift` → `shift_left`, and D15.11's `Split`.
1. **A modifier alone is a key.** It is a tap: pressed and released with
   no other key between.
   - **Name:** the modifier's, on either side: `shift`, `ctrl`, `alt`,
     `super`. Modifiers held during the tap come before it: `ctrl+shift`.
   - **Reading:** when no part of a spelling is a value, the last modifier
     written is the key: `shift+ctrl` is Ctrl tapped while Shift is held.
     `shift+shift` is refused, like any repeated modifier.
   - **Sides:** the sided keys (`shift_left`) stay physical. A source that
     has both tries the sided key first, as it tries a physical key before
     a character.
   - **When it's sent:** on release, and only if no other key was pressed
     in between. A double tap of Shift is `shift shift`.
   - **ParseHotty:** reads HOTTY 0.2's lone `Shift` as `shift`, and refuses
     `Shift+Shift`.
2. **Three kinds:** `Split` returns a `Kind`, one of `Character`,
   `Physical` or `Modifier`, in place of the `char` flag.
3. **The names are a spec:** `myelinkey/SPEC.md`, version 0.1, with
   hand-written vectors in `myelinkey/conformance/keys.json`, which
   myelinkey's tests run under Go and TinyGo.
   - **Generated parts:** Appendix A, the physical keys, comes from
     myelinkey, and the gate fails when it drifts.
   - **Changes:** a change to a name or a rule is a new spec version, with
     its vectors and a decision here. The hotty agent hears of it first.
4. **The split with HOTTY,** agreed with the hotty agent.
   - **This spec:** the names, the readings, the Shift rules, the lookup
     order for a press with two names, sequences and aliases, and display.
   - **HOTTY's §10.4:** how a host names a press. That covers the bytes
     table (legacy, modifyOtherKeys and kitty), the modifier bits, and
     offering the physical name first.
   - **Vectors:** HOTTY's `keys` vectors stay bytes → names.
   - **Order:** hotty-go moves onto myelinkey in the same train as §10.4,
     and `ParseHotty` goes then.
