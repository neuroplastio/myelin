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

_Last updated: 2026-10-30. myelin is public: the engine, the docs and the
godoc are done, and keys are their own module, myelinkey, named in
Ghostty's vocabulary (D13–D15). Key names 0.1 is released (D17).
Contexts run on cel-lite (D20)._

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

- [x] **RELEASE-01** The first tags, v0.1.0-next.1
      status: done | owner: myelin | added: 2026-10-11
      notes: On the maintainer's go-ahead ("Tag both packages"), and as gov
      P-5 has packages stay on 0.1.0 as `0.1.0-next.N` prereleases:
      myelinkey/v0.1.0-next.1, then the root's myelinkey requirement moved
      to it and v0.1.0-next.1 tagged there, each a GitHub prerelease.
      kubecom told, so it can require tags instead of pseudo-versions.

- [ ] **TINYGO-01** The root's tests under TinyGo, in the gate
      status: todo | owner: — | added: 2026-10-30
      notes: Since D20 the root builds with TinyGo, and a small program
      runs as wasip1 (523 KB). The suite doesn't run yet: it reads
      testdata with os.ReadFile, which TinyGo's js/wasm target can't, and
      t.Fatal doesn't stop a test there. Embed the testdata, as
      myelinkey's vectors are, then add the root to the gate's TinyGo
      step, with 64 KB stacks. Until then the docs claim TinyGo for
      myelinkey only.

- [ ] **DOCS-CHECK** Add `docs check` to the gate
      status: todo | owner: — | added: 2026-10-11
      notes: When the Docs Agent says the shared docs module has a
      repository and a version.
