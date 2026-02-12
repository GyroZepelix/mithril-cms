# Mithril CMS

An open-source, self-hostable, schema-first headless CMS. Content types are defined in YAML files, editors manage content through a React admin UI, and published content is consumed via a REST API. Ships as a single Go binary serving both the React SPA and the API, with PostgreSQL for storage.

## Quick Start with Docker

```bash
docker compose up
```

Navigate to [localhost:8080/admin](http://localhost:8080/admin) and log in with:
- Email: `admin@example.com`
- Password: `admin123456`

The default `docker-compose.yml` starts both Mithril and PostgreSQL with sample schemas.

## Features

- Schema-first: define content types in YAML, Mithril generates database tables
- 12 field types: string, text, integer, float, boolean, date, time, datetime, enum, media, relation-one, relation-many
- Full-text search with PostgreSQL tsvector (ranked results with highlights)
- JWT authentication with refresh token rotation and Argon2id password hashing
- Media upload with automatic image variant generation (thumbnail, medium, large)
- Audit logging for all admin actions
- Content type introspection API
- Hot schema refresh (apply schema changes without restarting)
- CLI for schema diff/apply operations
- React admin UI (embedded in the binary for production)
- Single binary deployment

## Tech Stack

| Layer    | Technology                                        |
|----------|---------------------------------------------------|
| Backend  | Go 1.24+                                          |
| Database | PostgreSQL 16+                                    |
| Router   | go-chi/chi v5                                     |
| DB driver| jackc/pgx v5                                      |
| Auth     | JWT (golang-jwt) + Argon2id password hashing      |
| Frontend | React 19 + TypeScript + Vite + Tailwind CSS + shadcn/ui |

## Development Setup

### Prerequisites

- **Go 1.24+**
- **PostgreSQL 16+**
- **Node.js 20+** (for the admin UI)

### Backend

```bash
export MITHRIL_DATABASE_URL="postgres://user:pass@localhost:5432/mithril?sslmode=disable"
export MITHRIL_JWT_SECRET="dev-secret"
export MITHRIL_ADMIN_EMAIL="admin@example.com"
export MITHRIL_ADMIN_PASSWORD="admin123456"
make run
```

The server will connect to PostgreSQL, run migrations, load YAML schemas, apply schema changes, and start listening on port 8080.

### Frontend

```bash
cd admin && npm install && npm run dev
```

The Vite dev server starts on port 5173. In dev mode (`MITHRIL_DEV_MODE=true`), the Go server proxies non-API requests to Vite.

### Full Build

```bash
make build-all
```

This builds the React admin, then compiles the Go binary with the SPA embedded (using the `embed_admin` build tag).

### Other Make Targets

```bash
make test    # Run all tests with race detection
make lint    # Run go vet (and golangci-lint if installed)
make clean   # Remove build artifacts
```

## Configuration

All configuration is via environment variables:

| Variable                 | Default     | Description                                                        |
|--------------------------|-------------|--------------------------------------------------------------------|
| `MITHRIL_PORT`           | `8080`      | HTTP server port                                                   |
| `MITHRIL_DATABASE_URL`   | *(required)* | PostgreSQL connection string                                      |
| `MITHRIL_SCHEMA_DIR`    | `./schema`  | Path to YAML schema files                                          |
| `MITHRIL_MEDIA_DIR`     | `./media`   | Path to media storage directory                                    |
| `MITHRIL_JWT_SECRET`    | *(required)* | Secret key for JWT signing                                        |
| `MITHRIL_DEV_MODE`      | `false`     | Enable dev mode (verbose logging, auto-apply breaking schema changes) |
| `MITHRIL_ADMIN_EMAIL`   | *(optional)* | Initial admin email (used on first run)                           |
| `MITHRIL_ADMIN_PASSWORD`| *(optional)* | Initial admin password (used on first run)                        |

## Schema Format

Content types are defined as YAML files in the schema directory. Example:

```yaml
name: blog_posts
display_name: Blog Posts
fields:
  - name: title
    type: string
    required: true
    searchable: true
    max_length: 200
  - name: body
    type: text
    searchable: true
  - name: author
    type: relation-one
    relation:
      content_type: authors
```

Supported field types: `string`, `text`, `integer`, `float`, `boolean`, `date`, `time`, `datetime`, `enum`, `media`, `relation-one`, `relation-many`.

See `spec/SPEC.md` for the full schema specification.

## API Reference

> For full API documentation with request/response examples, query parameters, field types, and error codes, see **[USAGE.md](USAGE.md)**.

### Health

| Method | Path      | Description        |
|--------|-----------|--------------------|
| GET    | `/health` | Health check (includes DB connectivity) |

### Public Content API

| Method | Path                    | Description                          |
|--------|-------------------------|--------------------------------------|
| GET    | `/api/{type}`           | List published entries (paginated, filterable, searchable) |
| GET    | `/api/{type}/{id}`      | Get a single published entry         |

### Authentication

| Method | Path                       | Description                |
|--------|----------------------------|----------------------------|
| POST   | `/admin/api/auth/login`    | Login (returns JWT + refresh token) |
| POST   | `/admin/api/auth/refresh`  | Refresh access token       |
| POST   | `/admin/api/auth/logout`   | Logout (revoke refresh token) |
| GET    | `/admin/api/auth/me`       | Get current admin profile  |

### Admin Content API (requires JWT)

| Method | Path                                        | Description          |
|--------|---------------------------------------------|----------------------|
| GET    | `/admin/api/content/{type}`                 | List all entries     |
| POST   | `/admin/api/content/{type}`                 | Create entry         |
| GET    | `/admin/api/content/{type}/{id}`            | Get entry            |
| PUT    | `/admin/api/content/{type}/{id}`            | Update entry         |
| POST   | `/admin/api/content/{type}/{id}/publish`    | Publish entry        |

### Media (requires JWT for management)

| Method | Path                       | Description                |
|--------|----------------------------|----------------------------|
| POST   | `/admin/api/media`         | Upload file                |
| GET    | `/admin/api/media`         | List media (paginated)     |
| DELETE | `/admin/api/media/{id}`    | Delete media and variants  |
| GET    | `/media/{filename}`        | Serve file (public)        |

### Admin Utilities (requires JWT)

| Method | Path                           | Description                    |
|--------|--------------------------------|--------------------------------|
| GET    | `/admin/api/content-types`     | List all content types         |
| GET    | `/admin/api/content-types/{name}` | Get content type details    |
| GET    | `/admin/api/audit-log`         | Query audit log (filterable)   |
| POST   | `/admin/api/schema/refresh`    | Reload and apply schema changes |

## CLI

```
mithril                    Start the HTTP server (default)
mithril serve              Start the HTTP server (explicit)
mithril schema diff        Show pending schema changes
mithril schema apply       Apply safe schema changes
mithril schema apply --force   Apply ALL schema changes (including breaking)
```

## Production Deployment

### Single Binary

```bash
make build-all    # Produces ./mithril with embedded admin UI
./mithril         # Requires MITHRIL_DATABASE_URL and MITHRIL_JWT_SECRET
```

### Docker

```bash
docker compose up -d
```

Or build and run the image directly:

```bash
docker build -t mithril-cms .
docker run -p 8080:8080 \
  -e MITHRIL_DATABASE_URL="postgres://..." \
  -e MITHRIL_JWT_SECRET="your-secret" \
  -e MITHRIL_ADMIN_EMAIL="admin@example.com" \
  -e MITHRIL_ADMIN_PASSWORD="admin123456" \
  -v ./schema:/schema:ro \
  -v media_data:/data/media \
  mithril-cms
```

**Note:** `MITHRIL_ADMIN_EMAIL` and `MITHRIL_ADMIN_PASSWORD` are only needed on first run to create the initial admin account.

### Security Notes

- **Change the JWT secret** -- use a long random string (32+ characters).
- **Use a strong admin password** -- the default `admin123456` is for development only.
- **Run behind HTTPS** -- use a reverse proxy (nginx, Caddy, Traefik) with TLS termination.
- **Restrict database access** -- do not expose PostgreSQL to the public internet.
- **Set `MITHRIL_DEV_MODE=false`** in production (this is the default).

## Project Structure

```
mithril-cms/
├── cmd/
│   └── mithril/          # Application entrypoint (main.go)
├── internal/
│   ├── config/           # Environment-based configuration
│   ├── database/         # PostgreSQL connection pool and migrations
│   ├── schema/           # YAML schema loader, validator, and DDL engine
│   ├── server/           # HTTP server, router, middleware, response helpers
│   ├── auth/             # JWT authentication, Argon2id hashing, middleware
│   ├── content/          # Dynamic content CRUD, validation, query builder
│   ├── search/           # Full-text search with PostgreSQL tsvector
│   ├── media/            # Media upload, image processing, file serving
│   ├── contenttypes/     # Content type introspection API
│   ├── schemaapi/        # Schema refresh API
│   └── audit/            # Audit logging system
├── migrations/           # SQL migration files (system tables)
├── schema/               # YAML content type definitions
├── admin/                # React admin SPA
├── spec/                 # Project specification and task planning
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
└── tools.go
```

## License

This project will be released under an open-source license. License selection is pending.
