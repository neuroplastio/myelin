# myelin: agent guide

myelin is a key-binding library for Go programs: context-bound keymaps and
which-key menus, as data. Myelin Agent (myelin@neuroplast.io) owns it. Its
license is Apache-2.0.

## What myelin is, and is not

- **Data in, data out.** A program hands myelin a keymap as documents (Doc,
  Bind, Group) and gets data back: matches, menus, explanations and the
  sequencer's state.
- **No files, no file format, no IO.** Every program keeps its own files
  and parser.
- **No clock, no goroutines.** `now` goes in and a deadline comes out.
- **Keys are canonical HOTTY names** (hotty-go's `ParseKey`), written in
  one short spelling: `ctrl+d`, `pgdn`, `g g`, `<leader>` (D3).
- **Contexts are CEL** over a path the program declares. The deepest
  context wins, then the layer, then the later binding (D6).
- **Programs keep what is theirs:** their files, their actions, their
  default keymaps, how they build the path, their own rules for which keys
  may be bound where, what happens to keys that matched nothing, and how
  menus are drawn.
- **myelin stands on its own** (D8): no mention of private repositories in
  the code, the docs or the vault.

## Rules

- **Data types have no hand-written methods** (D2). msgp's generated codec
  is the one exception. Every data type has `msg:"…"` tags; the codec is
  generated with `msgp -marshal=false`, `//msgp:newtime` and
  `//msgp:timezone utc`. Only the three handles have methods (Env, Keymap,
  Context), because they hold compiled CEL.
- **No `tool` directive in the root go.mod.** msgp's generator is pinned in
  `tools/go.mod` and run as `go tool -modfile=tools/go.mod msgp` (D1).
- **The root module depends only on** hotty-go's root module, msgp's
  runtime and cel-go. Bubble Tea lives in the `myelintea` module (D5).
- **Never edit a consumer's repository.** Each program makes its own
  switch, and its own golden proves it. When a consumer needs something, it
  asks here.
- **Key spelling and precedence are decisions.** A change to either is a
  new decision in `vault/knowledge/decisions.md`.
- **Docs are checked like code** (D9). The godoc documents every exported
  identifier. The code in README and `docs/` is the Example functions, and
  the key table is generated: after changing either, run
  `go test -run 'TestReferenceKeys|TestDocsExamples' -update`.

## Where things live

- `vault/knowledge/decisions.md`: the binding decisions, Dn, append-only.
  Supersede a decision instead of contradicting it.
- `vault/tasks/board.md`: open work. Claim an item before working on it,
  and drop it when done.
- `vault/journal/YYYY-MM.md`: one bullet per leg. The journal is the
  changelog.
- `vault/feedback/`: notes from the maintainer to the agent. Read them
  before the board, and delete each one once it is addressed.
- `docs/`: the user docs (README is the overview). Notes for people working
  on myelin belong in the vault, not there.

## Working

1. Pull, read the board's top and the latest journal entries, then check
   `vault/feedback/`.
2. Take one small leg and claim it on the board. Shared files can be
   changed by other sessions, so pull before editing them.
3. Implement it. `mise exec -- make check` must pass. Go isn't on PATH in
   an agent shell, so it runs through mise. While myelin is private, CI is
   skipped (D8): this local run is the gate, and the journal entry says so.
4. Record any decision, add a journal bullet, update the board, then commit
   and push to `main`. There are no feature branches, and nothing is ever
   force-pushed.

Commits are authored as Myelin Agent by the session's environment. Never
set `--author`, `GIT_AUTHOR_*` or `user.*`.

## Toolchain

```
mise exec -- make check      # the gate: gofmt, vet, test, msgp and vectors current
mise exec -- make generate   # regenerate data_gen.go after changing data.go
mise exec -- make vectors    # copy HOTTY's conformance vectors from the pinned hotty-go
mise exec -- make fuzz       # fuzz the short spelling's parser for 60 s
```

CI (`.github/workflows/ci.yml`) runs `scripts/check.sh` on `ubuntu-latest`
once myelin is public (D8).
