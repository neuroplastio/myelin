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

_Last updated: 2026-10-11. The engine, the docs and the godoc are done,
and the history starts fresh (D10). Going public waits for the
maintainer's final review._

## In progress

## Blocked

- [ ] **PUBLIC-01** myelin goes public
      status: blocked | owner: — | added: 2026-10-10
      notes: Waits for the maintainer's final review (D8). Then flip the
      visibility, so that CI starts, and tell kubecom the public commit to
      require. It holds its switch until then and until its own go-ahead.

- [ ] **KEYS-01** myelin owns the keys; hotty-go depends on myelin (D12)
      status: blocked | owner: — | added: 2026-10-11
      notes: After PUBLIC-01. Build the dependency-free keys module and
      take the root off hotty-go; then the hotty agent moves hotty-go onto
      it, and each consumer bumps. What stays in hotty-go is agreed with the
      hotty agent first.

## Todo

- [ ] **RELEASE-01** The first tag, v0.1.0
      status: todo | owner: — | added: 2026-10-11
      notes: gov's P-5 (draft): packages use semver. After PUBLIC-01 and on
      the maintainer's go-ahead, tag v0.1.0 (no public version above
      0.1.0), and myelintea/v0.1.0 with its myelin requirement moved to
      v0.1.0. Then tell kubecom, so it can require tags instead of
      pseudo-versions.

- [ ] **DOCS-CHECK** Add `docs check` to the gate
      status: todo | owner: — | added: 2026-10-11
      notes: When the Docs Agent says the shared docs module has a
      repository and a version.
