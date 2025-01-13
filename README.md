# Mithril CMS

## Development

### Database Setup

Mithril CMS uses PostgreSQL. The database configuration is defined in `docker-compose.yml`:

```yaml
db:
  image: postgres
  restart: always
  environment:
    POSTGRES_PASSWORD: S3cret
    POSTGRES_USER: mithril
    POSTGRES_DB: mithrildb
  volumes:
    - pgdata:/var/lib/postgresql/data 
  ports:
    - 5432:5432
```

To start the database:

```bash
docker compose up -d
```

### Using sqlc

sqlc generates type-safe Go code from SQL. Our project uses sqlc as follows:

1. Install sqlc:
   ```bash
   go install github.com/kyleconroy/sqlc/cmd/sqlc@latest
   ```

2. The `sqlc.yaml` configuration file is already in the project root.

3. SQL queries are located in `internal/constant/query/query.sql`.

4. Generate Go code:
   ```bash
   sqlc generate
   ```

   This will create or update files in `internal/storage/persistence/`.

### Using migrate

migrate manages database migrations:

1. Install migrate:
   ```bash
   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   ```

2. Migration files are located in `db/migration/`.

3. To create a new migration:
   ```bash
   migrate create -ext sql -dir db/migration -seq <migration_name>
   ```

4. To run migrations:
   ```bash
   migrate -database "postgresql://mithril:S3cret@localhost:5432/mithrildb?sslmode=disable" -path db/migration up
   ```

### Makefile

The project includes a Makefile for common tasks. Run `make help` to see available commands.
