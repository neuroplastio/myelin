# Task board

Open work only. A finished item is dropped; its record is the journal, the
commit and the decision log.

Item template:

```
- [ ] **ID** title
      status: todo | owner: — | added: YYYY-MM-DD
      notes: …
```

Statuses: `todo`, `in-progress`, `blocked`.

_Last updated: 2026-10-17. myelin is public: the engine, the docs and the
godoc are done, and keys are their own module, myelinkey, named in
Ghostty's vocabulary (D13–D15). Key names 0.1 is released (D17)._

## In progress

## Blocked

- [ ] **KEYS-02** HOTTY's key names follow myelinkey's; hotty-go takes them from it (D12 step 2, D15.10)
      status: blocked | owner: — | added: 2026-10-11
      notes: myelin's side is done (D13–D17). Key names 0.1 is released as
      `myelinkey/v0.1.0`. The hotty agent rewrites SPEC §10.4 to cite it
      for the names, `Mod` included as §4.4's `mod`, and keeps how a host
      names a press (D16.4). hotty-go moves onto
      `myelinkey` in the same train, now that myelin is public. When
      HOTTY's vectors move, run `make vectors HOTTY=main`, test them as
      bytes → names, and drop `ParseHotty`. When it lands, report the
      commits to the org's design record.

## Todo

- [ ] **FENCE-01** Fences: what a context does with the keys it doesn't bind, so programs can build modes from contexts
      status: todo | owner: — | added: 2026-10-17
      notes: Asked by plexos-key-presets for plx's modes (zellij's pane and
      locked modes, a modal dialog). Shaped in reply: `Doc.Fences []Fence`
      {When, Shadow, Swallow, Source}. The deepest holding fence decides
      both options (then layer, then later). Shadow is the Text floor for
      every key, and Swallow sets a new `Result.Swallowed` in place of
      Dropped. A Text node deeper than the fence keeps typed text, and a
      sequence that dies is decided on Pending.Path. Menu and Tree apply
      the floor, Explain gains `Candidate.Fenced`, and `(*Keymap).Fence(p)`
      returns the deciding fence. plx agreed to all of it and to "at least
      as deep": only the modes' own binds sit at a mode's depth, and the
      focused surfaces deeper keep their keys. plx will document
      `&& viewer.mode != 'locked'` for a user's own pane binds. Needs D18
      and a check against plx's keymap golden. Waits on the maintainer's go-ahead (token
      preservation). Tell plexos-key-presets the commit when it lands.

- [x] **RELEASE-01** The first tags, v0.1.0-next.1
      status: done | owner: myelin | added: 2026-10-11
      notes: On the maintainer's go-ahead ("Tag both packages"), and as gov
      P-5 has packages stay on 0.1.0 as `0.1.0-next.N` prereleases:
      myelinkey/v0.1.0-next.1, then the root's myelinkey requirement moved
      to it and v0.1.0-next.1 tagged there, each a GitHub prerelease.
      kubecom told, so it can require tags instead of pseudo-versions.

- [ ] **DOCS-CHECK** Add `docs check` to the gate
      status: todo | owner: — | added: 2026-10-11
      notes: When the Docs Agent says the shared docs module has a
      repository and a version.
