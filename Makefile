BINARY_NAME ?= mithril-cms
POSTGRESQL_URL ?= "postgresql://mithril:S3cret@localhost:5432/mithrildb?sslmode=disable"

build:
	@go build -o bin/$(BINARY_NAME) cmd/mithril-cms/main.go

run:
	@go run cmd/mithril-cms/main.go

test:
	@go test ./...

testsum:
	@gotestsum --format-hide-empty-pkg --format testdox ./...

testsum-watch:
	@gotestsum --format-hide-empty-pkg --format testdox --watch ./...

migration-up:
	migrate -database ${POSTGRESQL_URL} -path db/migration up

migration-down:
	migrate -database ${POSTGRESQL_URL} -path db/migration down

clean:
	@rm -rf bin/

help:
	@echo "Available commands:"
	@echo "  build            - Build the application"
	@echo "  run              - Run the application"
	@echo "  test             - Run all tests"
	@echo "  testsum          - Run tests with better formatting"
	@echo "  testsum-watch    - Run tests in watch mode"
	@echo "  migration-up     - Run database migrations up"
	@echo "  migration-down   - Run database migrations down"
	@echo "  clean            - Remove build artifacts"
	@echo "  help             - Show this help message"


