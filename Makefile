# myelin. Go, msgp and TinyGo are pinned in mise.toml: `mise exec -- make check`.
GO ?= go
export GO

.PHONY: check generate vectors fuzz

# The gate: gofmt, vet, test (myelinkey under TinyGo too), msgp's codecs and the
# vectors current.
check:
	./scripts/check.sh

# Regenerate the data types' msgp codec (data_gen.go).
generate:
	$(GO) generate ./...

# Copy HOTTY's conformance vectors from the hotty repository, at
# myelinkey/testdata/conformance/hotty-commit. `make vectors HOTTY=main` moves that
# pin to a branch's head first.
VECTORS = myelinkey/testdata/conformance
vectors:
	@if [ -n "$(HOTTY)" ]; then git ls-remote --exit-code https://github.com/neuroplastio/hotty "refs/heads/$(HOTTY)" | cut -f1 > $(VECTORS)/hotty-commit; fi
	curl -fsSL "https://raw.githubusercontent.com/neuroplastio/hotty/$$(cat $(VECTORS)/hotty-commit)/conformance/vectors.json" -o $(VECTORS)/vectors.json

# Fuzz the short spelling's parser.
fuzz:
	$(GO) -C myelinkey test -run '^$$' -fuzz FuzzParse -fuzztime 60s .
	$(GO) -C myelinkey test -run '^$$' -fuzz FuzzHottyName -fuzztime 60s .
