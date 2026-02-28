.PHONY: build release run clean lint help

## help: show this help message
help:
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | awk -F: '{desc=$$2; for(i=3;i<=NF;i++) desc=desc":"$$i; printf "  %-10s %s\n", $$1, desc}'

# Go parameters
BINARY_NAME=ops
BUMP ?= patch

## build: build binary to bin/ops
build:
	go build -o bin/$(BINARY_NAME) ./cmd/ops

## release: bump version and build release binaries (BUMP=major|minor|patch, default: patch)
release:
	@set -e; \
	current=$$(sed -n 's/.*Version = "\([0-9]*\.[0-9]*\.[0-9]*\)".*/\1/p' internal/version.go); \
	major=$$(echo $$current | cut -d. -f1); \
	minor=$$(echo $$current | cut -d. -f2); \
	patch=$$(echo $$current | cut -d. -f3); \
	case "$(BUMP)" in \
		major) major=$$((major + 1)); minor=0; patch=0 ;; \
		minor) minor=$$((minor + 1)); patch=0 ;; \
		patch) patch=$$((patch + 1)) ;; \
		*) echo "Error: BUMP must be 'major', 'minor', or 'patch'"; exit 1 ;; \
	esac; \
	new="$$major.$$minor.$$patch"; \
	echo "Bumping version: $$current -> $$new"; \
	perl -pi -e "s/\"$$current\"/\"$$new\"/" internal/version.go
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -a -ldflags="-w -s" -o $(BINARY_NAME)_unix    ./cmd/ops
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -a -ldflags="-w -s" -o $(BINARY_NAME)_darwin  ./cmd/ops
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -a -ldflags="-w -s" -o $(BINARY_NAME).exe     ./cmd/ops

## run: build and run the binary
run:
	go build -o bin/$(BINARY_NAME) ./cmd/ops
	./bin/$(BINARY_NAME)

## clean: remove build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)_unix
	rm -f $(BINARY_NAME)_darwin
	rm -f $(BINARY_NAME).exe

## deps: download and tidy Go module dependencies
deps:
	go mod download
	go mod tidy

## test: run all tests
test:
	go test -v ./...

## lint: check formatting (gofmt) and run static analysis (go vet)
lint:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt: the following files need formatting:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi
	go vet ./...
