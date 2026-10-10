#!/usr/bin/env bash
# myelin's gate (make check, and CI): gofmt, vet and test (the root module,
# key and myelintea), key under TinyGo, msgp's generated codecs current, and
# HOTTY's conformance vectors and hottytea from the hotty-go the root pins.
#
# A plain script rather than Makefile recipes, so that CI runs the same gate
# on a runner image without make.
set -euo pipefail
cd "$(dirname "$0")/.."
GO=${GO:-go}

echo "== gofmt"
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
	echo "gofmt: not formatted:"
	echo "$unformatted"
	exit 1
fi

echo "== vet"
"$GO" vet ./...

echo "== test"
"$GO" test ./...

# Re-run msgp and compare with the files as they were, so the check holds
# for uncommitted work too. A stale file is left regenerated: that is the
# fix.
# The key module needs nothing but the standard library and builds with
# TinyGo, whose web builds run it as 32-bit wasm with 64 KB stacks.
echo "== key"
"$GO" -C key vet ./...
"$GO" -C key test ./...
if ! command -v tinygo >/dev/null; then
	echo "key: tinygo is not on PATH; run the gate through mise: mise exec -- make check"
	exit 1
fi
(cd key && tinygo test -target wasm -stack-size=64KB .)
# Its vectors are HOTTY's, from the hotty repository at hotty-commit.
commit=$(cat key/testdata/conformance/hotty-commit)
if ! curl -fsSL "https://raw.githubusercontent.com/neuroplastio/hotty/$commit/conformance/vectors.json" |
	cmp -s - key/testdata/conformance/vectors.json; then
	echo "key: testdata/conformance/vectors.json is not hotty $commit's; run make vectors"
	exit 1
fi

echo "== myelintea"
"$GO" -C myelintea vet ./...
"$GO" -C myelintea test ./...
# Its hottytea is the hotty-go commit the root pins: one HOTTY key naming.
root=$("$GO" list -m -f '{{.Version}}' github.com/neuroplastio/hotty-go)
tea=$("$GO" -C myelintea list -m -f '{{.Version}}' github.com/neuroplastio/hotty-go/hottytea)
if [ "${root##*-}" != "${tea##*-}" ]; then
	echo "myelintea: hottytea is $tea, but the root pins hotty-go $root; pin both to one commit"
	exit 1
fi

echo "== msgp"
# msgp is pinned in mise.toml, as Go is.
if ! command -v msgp >/dev/null; then
	echo "msgp: not on PATH; run the gate through mise: mise exec -- make check"
	exit 1
fi
before=$(mktemp -d)
trap 'rm -rf "$before"' EXIT
gens=$(find . -name '*_gen.go')
for f in $gens; do
	mkdir -p "$before/$(dirname "$f")"
	cp "$f" "$before/$f"
done
"$GO" generate -run msgp ./...
stale=""
for f in $(find . -name '*_gen.go'); do
	cmp -s "$f" "$before/$f" || stale="$stale $f"
done
if [ -n "$stale" ]; then
	echo "msgp: generated code was not current; regenerated:$stale"
	exit 1
fi

echo "== vectors"
"$GO" mod download github.com/neuroplastio/hotty-go
hotty=$("$GO" list -m -f '{{.Dir}}' github.com/neuroplastio/hotty-go)
if ! cmp -s "$hotty/testdata/conformance/vectors.json" testdata/conformance/vectors.json; then
	echo "vectors: testdata/conformance/vectors.json is not the pinned hotty-go's; run make vectors"
	exit 1
fi

echo "check: ok"
