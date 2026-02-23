.PHONY: build release run clean

# Go parameters
BINARY_NAME=ops


# Build the binary
build:
	go build -o bin/$(BINARY_NAME) -v ./...

# Build for production with optimizations
release:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -a -installsuffix cgo -ldflags="-w -s" -o $(BINARY_NAME)_unix ./...
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -a -installsuffix cgo -ldflags="-w -s" -o $(BINARY_NAME)_darwin ./...
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -a -installsuffix cgo -ldflags="-w -s" -o $(BINARY_NAME).exe ./...

# Run the application
run:
	go build -o bin/$(BINARY_NAME) -v ./...
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
