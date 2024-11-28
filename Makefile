build:
	go build -o bin/main cmd/mithril-cms/main.go

run:
	go run cmd/mithril-cms/main.go

migration-up:
	migrate -database ${POSTGRESQL_URL} -path db/migration up

migration-down:
	migrate -database ${POSTGRESQL_URL} -path db/migration down
