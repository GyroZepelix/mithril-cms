# Mithril CMS

A lightweight, headless Content Management System built with Go, designed for developers who need a simple yet powerful backend for content management without the overhead of a traditional CMS frontend.

## Overview
Mithril CMS is a headless CMS that provides a clean REST API for managing content, users, and permissions mainly built for the purpose of having a simple and light backend for a blogging website. Built with Go and PostgreSQL, it offers a minimalist approach to content management while maintaining flexibility and security through a robust role-based permission system.

> [!NOTE]
> This app is still in heavy development, as its a project I tackled for the purpose of trying Golang for the first time and as such many features are still missing like:
> - Completely configurable role and authentication system allowing adding custom roles and permissions through config files
> - Flexible configurable tables that can be used for many scenarios ( Currently only users, content and their comments are implemented )
> - A possible WebUI for monitoring and adding objects and tables
> ...and many many more!

## Development

### Prerequisites

- Go 1.23+
- PostgreSQL 13+
- Docker (for database setup)
- migrate CLI tool
- sqlc (for code generation)

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

### Running the Application

1. Start the database:
   ```bash
   just run  # or: go run cmd/mithril-cms/main.go
   ```

2. The server will start on `http://localhost:8080` by default.

### API Endpoints

#### Authentication
- `GET /api/login` - User login
- `POST /api/register` - User registration

#### Content Management
- `GET /api/contents` - List all posts
- `GET /api/contents/{id}` - Get specific post
- `POST /api/contents` - Create new post (Author/Admin)
- `PUT /api/contents/{id}` - Update post (Owner/Admin)

#### User Management
- `GET /api/users` - List users (Admin only)
- `GET /api/users/{id}` - Get user profile (Owner/Admin)

### Justfile

The project includes a Justfile for common tasks. Run `just -l` to see available recipes.
