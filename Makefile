.PHONY: build build-admin build-all run test testsum dev-up dev-down lint clean

# Auto-load .env if present
ifneq (,$(wildcard .env))
include .env
export
endif

# Binary output name
BINARY := mithril

# Build the Go binary (without embedded admin SPA)
build:
	go build -o $(BINARY) ./cmd/mithril/

# Build the admin React SPA
build-admin:
	cd admin && npm ci && npm run build

# Build everything: admin SPA + Go binary with embedded admin
build-all: build-admin
	go build -tags embed_admin -o $(BINARY) ./cmd/mithril/

# Run the server (dev mode, no embed)
run: build
	./$(BINARY)

# Run all tests
test:
	go test -race -count=1 ./...

# Run tests using gotestsum
testsum:
	gotestsum

# Spin up containers needed to run dev mode
dev-up:
	docker compose -f docker-compose.dev.yml up -d

# Spin down dev containers
dev-down:
	docker compose -f docker-compose.dev.yml down

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
	rm -rf admin/dist
	go clean -cache -testcache
