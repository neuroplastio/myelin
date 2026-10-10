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
