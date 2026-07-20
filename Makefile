SHELL := /bin/sh

README := README.md
DOCS := docs
GOMARKDOC ?= $(shell go env GOPATH)/bin/gomarkdoc

.PHONY: docs docs-check docs-generate generate generate-check integration-test pages-help parity-check

generate:
	@go run ./cmd/contractsync
	@go run ./cmd/wrappergen

generate-check:
	@go run ./cmd/contractsync -check
	@go run ./cmd/wrappergen -check

parity-check:
	@test -n "$(PYTHON_CONTRACT_SNAPSHOT)" || (printf '%s\n' "Set PYTHON_CONTRACT_SNAPSHOT to the committed Python contract_snapshot.json path." >&2; exit 2)
	@go run ./cmd/contractparity -python "$(PYTHON_CONTRACT_SNAPSHOT)"

docs:
	@mkdir -p $(DOCS)
	@sed 's#](docs/#](#g' $(README) > $(DOCS)/index.md
	@printf '%s\n' "Docs home refreshed from README.md."

docs-check:
	@tmp="$$(mktemp)"; sed 's#](docs/#](#g' $(README) > "$$tmp" && cmp -s "$$tmp" $(DOCS)/index.md; status=$$?; rm -f "$$tmp"; exit $$status

docs-generate:
	@mkdir -p $(DOCS)/reference
	@$(GOMARKDOC) -u -o $(DOCS)/reference/aeries.md .
	@$(GOMARKDOC) -u -o $(DOCS)/reference/contract.md ./contract
	@$(GOMARKDOC) -u -o $(DOCS)/reference/internal.md ./internal/contract
	@printf '%s\n' "# Generated API Reference" "" "- [Public Package](aeries.md)" "- [Contract Package](contract.md)" "- [Internal Packages](internal.md)" > $(DOCS)/reference/index.md

integration-test:
	@go test -tags=integration ./...

pages-help:
	@printf '%s\n' "Use 'make generate' for contract and wrapper artifacts, 'make generate-check' to detect drift, 'make parity-check PYTHON_CONTRACT_SNAPSHOT=/path/to/contract_snapshot.json' for cross-SDK parity, 'make docs' to mirror README.md, 'make docs-generate' for API docs, 'make integration-test' for live read-only smoke checks, and mkdocs build after installing MkDocs Material and gomarkdoc."
