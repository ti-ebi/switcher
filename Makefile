GO ?= go
GOLANGCI_LINT ?= golangci-lint
GOFUMPT ?= gofumpt
GOIMPORTS ?= goimports

GOFILES := $(shell find . -type f -name '*.go' -not -path './vendor/*')

.PHONY: fmt fmt-check lint test build check

fmt:
	$(GOFUMPT) -w $(GOFILES)
	$(GOIMPORTS) -w $(GOFILES)

fmt-check:
	@test -z "$$($(GOFUMPT) -l $(GOFILES))" || (echo "gofumpt check failed. Run: make fmt" && exit 1)
	@test -z "$$($(GOIMPORTS) -l $(GOFILES))" || (echo "goimports check failed. Run: make fmt" && exit 1)

lint:
	$(GOLANGCI_LINT) run ./...

test:
	$(GO) test ./...

build:
	$(GO) build ./...

check: fmt-check lint test build
