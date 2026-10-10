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

# Copy HOTTY's conformance vectors: key's from the hotty repository at
# key/testdata/conformance/hotty-commit (`make vectors HOTTY=main` moves that
# to a branch's head first), the root's from the hotty-go go.mod pins.
VECTORS = key/testdata/conformance
vectors:
	@if [ -n "$(HOTTY)" ]; then git ls-remote --exit-code https://github.com/neuroplastio/hotty "refs/heads/$(HOTTY)" | cut -f1 > $(VECTORS)/hotty-commit; fi
	curl -fsSL "https://raw.githubusercontent.com/neuroplastio/hotty/$$(cat $(VECTORS)/hotty-commit)/conformance/vectors.json" -o $(VECTORS)/vectors.json
	$(GO) mod download github.com/neuroplastio/hotty-go
	install -m 644 "$$($(GO) list -m -f '{{.Dir}}' github.com/neuroplastio/hotty-go)/testdata/conformance/vectors.json" testdata/conformance/vectors.json

# Fuzz the short spelling's parser.
fuzz:
	$(GO) test -run '^$$' -fuzz FuzzParseKey -fuzztime 60s .
	$(GO) -C key test -run '^$$' -fuzz FuzzParse -fuzztime 60s .
	$(GO) -C key test -run '^$$' -fuzz FuzzHottyName -fuzztime 60s .
