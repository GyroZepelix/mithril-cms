BINARY_NAME := "mithril-cms"
POSTGRESQL_URL := "postgresql://mithril:S3cret@localhost:5432/mithrildb?sslmode=disable"

_default:
    @just --list

# Build the application
build:
    go build -o bin/{{BINARY_NAME}} cmd/mithril-cms/main.go

# Run the application
run:
    go run cmd/mithril-cms/main.go

# Run all tests
test:
    go test ./...

# Run tests with improved formatting
testsum:
    gotestsum --format-hide-empty-pkg --format testdox ./...

# Run tests in watch mode
testsum-watch:
    gotestsum --format-hide-empty-pkg --format testdox --watch ./...

# Run database migrations up
migration-up:
    migrate -database {{POSTGRESQL_URL}} -path db/migration up

# Run database migrations down
migration-down:
    migrate -database {{POSTGRESQL_URL}} -path db/migration down

# Drop database migrations
migration-drop:
    migrate -database {{POSTGRESQL_URL}} -path db/migration drop

# Generate mock files
generate-mock:
    go generate ./...

# Clean build artifacts
clean:
    rm -rf bin/
