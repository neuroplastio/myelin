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
godoc are done, and keys are their own module, myelinkey (D13, D14)._

## In progress

## Blocked

- [ ] **KEYS-02** hotty-go takes its key names from `myelinkey` (D12 step 2)
      status: blocked | owner: — | added: 2026-10-11
      notes: myelin's step is done (D13, D14). The hotty agent makes
      `hotty.ParseKey` delegate to `myelinkey.ParseHotty` after HOTTY's
      msgpack release; it reports changes to its key code or vectors
      before then, and `make vectors HOTTY=main` follows a vectors bump.
      When it lands, report the commits to the org's design record, which
      describes step 2 as pending.

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
