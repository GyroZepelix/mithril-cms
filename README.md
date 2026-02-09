# Mithril CMS

An open-source, self-hostable, schema-first headless CMS. Content types are defined in YAML files, editors manage content through a React admin UI, and published content is consumed via a REST API. Ships as a single Go binary serving both the React SPA and the API, with PostgreSQL for storage.

## Status

**Early development / Work in progress.**

The project scaffold is in place with configuration loading and structured logging. The database layer, schema engine, HTTP server, API routes, and admin UI are not yet implemented. See `spec/SPEC.md` for the full planned specification.

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
│   └── config/           # Environment-based configuration
├── migrations/           # SQL migration files (golang-migrate, not yet populated)
├── schema/               # YAML content type definitions
│   ├── authors.yaml
│   └── blog_posts.yaml
├── spec/                 # Project specification and task planning
├── admin/                # React admin SPA (not yet initialized)
├── Makefile
├── go.mod
└── tools.go              # Tracks Go dependencies not yet directly imported
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

Run the binary:

```bash
make run
```

Other available Make targets:

```bash
make test    # Run all tests with race detection
make lint    # Run go vet (and golangci-lint if installed)
make clean   # Remove build artifacts
```

**Note:** The database connection and HTTP server are not yet wired up. Running the binary will load configuration, log startup info, and exit. This is expected at the current stage of development.

## License

This project will be released under an open-source license. License selection is pending.
