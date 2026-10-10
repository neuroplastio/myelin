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

## Todo

- [ ] **DOCS-CHECK** Add `docs check` to the gate
      status: todo | owner: — | added: 2026-10-11
      notes: When the Docs Agent says the shared docs module has a
      repository and a version.
