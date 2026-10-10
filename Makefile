# myelin. Go and msgp are pinned in mise.toml: `mise exec -- make check`.
GO ?= go
export GO

.PHONY: check generate vectors fuzz

# The gate: gofmt, vet, test, msgp's codecs and the vectors current.
check:
	./scripts/check.sh

# Regenerate the data types' msgp codec (data_gen.go).
generate:
	$(GO) generate ./...

# Copy HOTTY's conformance vectors from the hotty-go that go.mod pins.
vectors:
	$(GO) mod download github.com/neuroplastio/hotty-go
	install -m 644 "$$($(GO) list -m -f '{{.Dir}}' github.com/neuroplastio/hotty-go)/testdata/conformance/vectors.json" testdata/conformance/vectors.json

# Fuzz the short spelling's parser.
fuzz:
	$(GO) test -run '^$$' -fuzz FuzzParseKey -fuzztime 60s .
