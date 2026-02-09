.PHONY: build run test lint clean

# Binary output name
BINARY := mithril

# Build the Go binary
build:
	go build -o $(BINARY) ./cmd/mithril/

# Run the server
run: build
	./$(BINARY)

# Run all tests
test:
	go test -race -count=1 ./...

# Run linters (requires golangci-lint to be installed)
lint:
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi

# Remove build artifacts
clean:
	rm -f $(BINARY)
	go clean -cache -testcache
