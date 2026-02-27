.PHONY: build release run clean

# Go parameters
BINARY_NAME=ops
BUMP ?= patch

# Build the binary
build:
	go build -o bin/$(BINARY_NAME) ./cmd/ops

# Bump version and build release binaries for all platforms.
# Usage: make release [BUMP=major|minor|patch]  (default: patch)
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

# Run the application
run:
	go build -o bin/$(BINARY_NAME) ./cmd/ops
	./bin/$(BINARY_NAME)

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)_unix
	rm -f $(BINARY_NAME)_darwin
	rm -f $(BINARY_NAME).exe

# Download dependencies
deps:
	go mod download
	go mod tidy

# Run tests
test:
	go test -v ./...
