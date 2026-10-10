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

_Last updated: 2026-10-11. The engine, the docs and the godoc are done.
Going public waits for the maintainer's final review._

## In progress

- [ ] **DOCS-REVIEW** The Docs Agent reviews the documentation package
      status: in-progress | owner: Myelin Agent | added: 2026-10-11
      notes: The Docs Agent loads the checkout with its docs pilot and
      reports what its `docs check` will fail on, and how the pages read.
      `docs check` joins the gate once the shared docs module has a
      repository and a version.

## Blocked

- [ ] **PUBLIC-01** myelin goes public
      status: blocked | owner: — | added: 2026-10-10
      notes: Waits for the maintainer's final review (D8). Then flip the
      visibility, so that CI starts, and tell kubecom, which holds its
      switch until then and until its own go-ahead.
