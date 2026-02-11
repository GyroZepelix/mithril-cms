# Mithril CMS

An open-source, self-hostable, schema-first headless CMS. Content types are defined in YAML files, editors manage content through a React admin UI, and published content is consumed via a REST API. Ships as a single Go binary serving both the React SPA and the API, with PostgreSQL for storage.

## Status

**Early development / Work in progress.**

Currently implemented:
- Configuration loading and structured logging
- Database connection pool with health checks
- System table migrations (golang-migrate)
- YAML schema loader and validator
- Schema-to-SQL migration engine with diff detection
- HTTP server with graceful shutdown and request timeouts
- Full API route tree with middleware (RequestID, RealIP, slog logger, Recoverer, CORS, JSON Content-Type enforcement)
- Response helpers (JSON, Error, Paginated) matching spec envelope format
- Health check endpoint (`/health`) with database connectivity check
- SPA handler (dev mode proxies to Vite, production serves placeholder)
- Authentication system (JWT access tokens + refresh token rotation, Argon2id password hashing)
- Content CRUD API (dynamic SQL generation, validation for all 12 field types, pagination/filtering/sorting)
- Full-text search integration (PostgreSQL tsvector, ranked results with highlights)
- Media upload system (image variant generation, MIME validation, security headers, path traversal protection)

Pending: Audit logging, schema refresh endpoint, content type introspection API, admin UI. See `spec/SPEC.md` for the full planned specification.

## Tech Stack

| Layer    | Technology                                        |
|----------|---------------------------------------------------|
| Backend  | Go 1.24+                                          |
| Database | PostgreSQL 16+                                    |
| Router   | go-chi/chi v5                                     |
| DB driver| jackc/pgx v5                                      |
| Auth     | JWT (golang-jwt) + Argon2id password hashing      |
| Frontend | React 19 + TypeScript + Vite + Tailwind CSS + shadcn/ui (coming soon) |

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
│   └── audit/            # Audit logging system (pending implementation)
├── migrations/           # SQL migration files (system tables)
├── schema/               # YAML content type definitions
│   ├── authors.yaml
│   └── blog_posts.yaml
├── spec/                 # Project specification and task planning
├── admin/                # React admin SPA (not yet initialized)
├── Makefile
├── go.mod
└── tools.go              # Tracks Go dependencies not directly imported
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
| `MITHRIL_ADMIN_EMAIL`   | *(required on first run)* | Initial admin email                               |
| `MITHRIL_ADMIN_PASSWORD`| *(required on first run)* | Initial admin password                            |

## Development Prerequisites

- **Go 1.24+**
- **PostgreSQL 16+**
- **Node.js 20+** (for the admin UI, coming soon)

## Getting Started

Clone the repository and build:

```bash
make build
```

Run the binary (requires PostgreSQL and `MITHRIL_DATABASE_URL` environment variable):

```bash
export MITHRIL_DATABASE_URL="postgres://user:pass@localhost:5432/mithril?sslmode=disable"
make run
```

The server will:
1. Connect to PostgreSQL and run system table migrations
2. Load and validate YAML schemas from `./schema`
3. Apply schema changes to the database (creating/altering content type tables)
4. Start the HTTP server on port 8080 (configurable via `MITHRIL_PORT`)

Check the health endpoint:

```bash
curl http://localhost:8080/health
```

Other available Make targets:

```bash
make test    # Run all tests with race detection
make lint    # Run go vet (and golangci-lint if installed)
make clean   # Remove build artifacts
```

**Note:** The following API routes are now functional: authentication (`/admin/api/auth/*`), content CRUD (`/api/{content-type}`, `/admin/api/content/{content-type}`), and media upload/serving (`/admin/api/media`, `/media/{filename}`). Pending routes: audit log, content type introspection, and schema refresh.

## License

This project will be released under an open-source license. License selection is pending.
