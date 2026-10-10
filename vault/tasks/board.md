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
      Its local switch is green on myelin 2ebaa0e with `key` 5790483
      (kubecom 5d1b2f1, unpushed): live keys go through
      `hottytea.KeyName` and `key.ParseHotty` (D13).

- [ ] **KEYS-02** hotty-go takes its key names from `key` (D12 step 2)
      status: blocked | owner: — | added: 2026-10-11
      notes: myelin's step is done (D13). The hotty agent makes
      `hotty.ParseKey` delegate to `key.ParseHotty` after HOTTY's msgpack
      release; it reports changes to its key code or vectors before then,
      and `make vectors HOTTY=main` follows a vectors bump. When it lands,
      send gov the commits (NEIO-13's "until the switch" lines).

## Todo

- [ ] **RELEASE-01** The first tag, v0.1.0
      status: todo | owner: — | added: 2026-10-11
      notes: gov's P-5 (draft): packages use semver. After PUBLIC-01 and on
      the maintainer's go-ahead, tag key/v0.1.0, move the root's key
      requirement to it, and tag v0.1.0 there (no public version above
      0.1.0). Then tell kubecom, so it can require tags instead of
      pseudo-versions.

- [ ] **DOCS-CHECK** Add `docs check` to the gate
      status: todo | owner: — | added: 2026-10-11
      notes: When the Docs Agent says the shared docs module has a
      repository and a version.
