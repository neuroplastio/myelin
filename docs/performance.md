---
group: Reference
order: 30
nav: Performance
description: what a key, a match, a menu and a context cost, how they were measured, and how to run the benchmarks
path: performance
---

# Performance

A key no binding starts with costs one map lookup. Matching a key against
bindings with contexts takes under half a microsecond and allocates
nothing, and compiling a context, once when the keymap loads, takes about
two microseconds. A program can look a key up on every keystroke, and
draw a menu on every frame, without counting.

## The numbers

| Operation | Time | Allocations |
|-----------|------|-------------|
| A key no binding starts with (`Feed`) | 45 ns | 1 |
| A key matched against bindings with contexts (`Match`) | 0.44 µs | 0 |
| The leader's menu (`Menu`) | 17 µs | 60 |
| A palette row's key (`KeyFor`) | 0.41 µs | 0 |
| Compiling a context, once, at load (`Compile`) | 2 µs | 22 |

A query evaluates each distinct context once, on an activation the keymap
pools, and evaluating a context doesn't allocate, so a match allocates
nothing. A key no binding starts with allocates only the slice that hands
it back to the program. A menu's allocations are its items.

## How they were measured

The keymap is the size of a program's defaults: a hundred bindings. For
each of 25 keys there is a binding under the leader key, a sequence that
continues it, a binding in the context
`sidebar && panel.ext == 'git' && !input`, and one in
`buffer.tags['format'] == 'gofmt'`, with a group labelling each leader
prefix. The path is a git panel in a window's sidebar, three nodes deep.

The times are the medians of six runs of `go test -bench . -benchmem`,
with Go 1.26 on linux/amd64, on a laptop with an AMD Ryzen AI 9 HX 370
that was doing other work. They move with the machine and its load; the
allocations don't.

## Size

myelin requires cel-lite, its own key names and msgp's runtime, which
need only Go's standard library. A small program that declares a schema,
builds a keymap with contexts and matches a key is 2.5 MB built for
linux/amd64, and 4.2 MB for js/wasm, with `-trimpath -ldflags=-s` and
Go 1.26. A program that prints hello is 1.6 MB, or 2.6 MB in js/wasm, so
most of that is Go's own runtime.

## What the gate holds

A test fails when an operation allocates more than it does now:

- a key no binding starts with: 1;
- a prefix of a longer binding, a match in contexts and a palette row's
  key: none;
- the leader's menu: at most 100;
- every key's binding on a path six nodes deep (`Effective`): at most 8.

The times aren't checked, since they move with the machine.

## Running them

```sh
mise exec -- make bench
```

runs every benchmark, myelinkey's included, with their allocations. To
compare two runs, save each one's output and give both to
[benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat).
