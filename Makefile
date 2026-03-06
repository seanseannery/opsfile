.PHONY: build release run clean lint help setup-local-dev

## help: show this help message
help:
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | awk -F: '{desc=$$2; for(i=3;i<=NF;i++) desc=desc":"$$i; printf "  %-10s %s\n", $$1, desc}'

# Go parameters
BINARY_NAME=ops
VERSION     ?= $(shell git describe --tags --always --dirty)
COMMIT      ?= $(shell git rev-parse --short HEAD)
LD_FLAGS    := -w -s -X sean_seannery/opsfile/internal.Version=$(VERSION) -X sean_seannery/opsfile/internal.Commit=$(COMMIT)

## install-hooks: install go, and any other dependencies and githooks from .githooks/ 
setup-local-dev:
	@if ! command -v go &>/dev/null; then \
		brew install go@1.25.5; \
		echo "Installed golang"; \
	fi
	make deps
	git config core.hooksPath .githooks
	@echo "Git hooks installed from .githooks/"
	

## build: build binary to bin/ops
build: clean
	go build -ldflags="$(LD_FLAGS)" -o bin/$(BINARY_NAME) ./cmd/ops

## release: build release binaries for all platforms (VERSION and COMMIT set via git or overridden externally)
release: clean
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -a -ldflags="$(LD_FLAGS)" -o bin/$(BINARY_NAME)_unix_$(VERSION)    ./cmd/ops
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -a -ldflags="$(LD_FLAGS)" -o bin/$(BINARY_NAME)_darwin_$(VERSION)  ./cmd/ops
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -a -ldflags="$(LD_FLAGS)" -o bin/$(BINARY_NAME)_$(VERSION).exe     ./cmd/ops
	@echo "Built: bin/$(BINARY_NAME)_unix_$(VERSION)  bin/$(BINARY_NAME)_darwin_$(VERSION)  bin/$(BINARY_NAME)_$(VERSION).exe"

## run: build and run the binary
run:
	go build -o bin/$(BINARY_NAME) ./cmd/ops
	./bin/$(BINARY_NAME) --version
	./bin/$(BINARY_NAME) --help

## clean: remove build artifacts
clean:
	go clean
	rm -f bin/$(BINARY_NAME)*

## deps: download and tidy Go module dependencies
deps:
	go mod download
	go mod tidy

## test: run all tests
test:
	go test -v ./...

## lint: check formatting (gofmt) and run static analysis (go vet)
lint:
	@echo "--- gofmt ---"; \
	unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "FAIL: the following files need formatting:"; \
		echo "$$unformatted"; \
		echo "Run: gofmt -w <file> to fix"; \
		exit 1; \
	fi; 
	@echo "--- go vet ---"
	go vet ./... 
