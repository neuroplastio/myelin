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

_Last updated: 2026-10-11. myelin is public: the engine, the docs and the
godoc are done, and keys are their own module, myelinkey, named in
Ghostty's vocabulary (D13–D15)._

## In progress

## Blocked

- [ ] **KEYS-02** HOTTY's key names follow myelinkey's; hotty-go takes them from it (D12 step 2, D15.10)
      status: blocked | owner: — | added: 2026-10-11
      notes: myelin's side is done (D13–D15). The hotty agent amends SPEC
      §10.4 and the vectors to D15's names in place, then hotty-go takes
      its names from `myelinkey`. It plans this after its msgpack consumer
      bumps. hotty-go is public, so it can require `myelinkey` only once
      myelin is public. When the vectors move, run `make vectors HOTTY=main`,
      test the names directly and drop `ParseHotty`. When it lands, report
      the commits to the org's design record.

## Todo

- [ ] **RELEASE-01** The first tag, v0.1.0
      status: todo | owner: — | added: 2026-10-11
      notes: The org's versioning policy (draft): packages use semver. On
      the maintainer's go-ahead, tag myelinkey/v0.1.0, move the root's
      myelinkey requirement to it, and tag v0.1.0 there (no public version
      above 0.1.0). Then tell kubecom, so it can require tags instead of
      pseudo-versions.

- [ ] **DOCS-CHECK** Add `docs check` to the gate
      status: todo | owner: — | added: 2026-10-11
      notes: When the Docs Agent says the shared docs module has a
      repository and a version.
