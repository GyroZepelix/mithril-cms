# Mithril CMS — Project Specification

## Overview

Mithril CMS is an open-source, self-hostable, schema-first headless CMS. Content types are defined in YAML files, editors manage content through a React admin UI, and published content is consumed via a REST API. The system ships as a single Go binary that serves both the React SPA and the API, with PostgreSQL for storage.

## Definition of Done

The MVP is complete when:
1. `docker compose up` starts Mithril + PostgreSQL
2. Admin UI is accessible at `localhost:8080/admin` with login
3. Content types are loaded from `schema/*.yaml` and displayed in the sidebar
4. Full content lifecycle works: create draft → edit → publish → visible on public API
5. Media upload produces image variants; media library works
6. Audit log captures all actions and is viewable in the admin UI
7. Schema refresh (add fields, new content types) works without data loss
8. Full-text search returns ranked results with highlights
9. A single `make build` produces one binary that serves everything

## Constraints

- Single Go binary (no external runtime dependencies beyond PostgreSQL)
- No ORM — raw SQL with parameterized queries via pgx
- All user input validated and sanitized; no SQL injection possible
- JWT auth with short-lived access tokens + rotating refresh tokens
- Image processing in pure Go (no CGo / ImageMagick dependency)
- The admin SPA is embedded into the Go binary for production

## Architectural Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Module path | `github.com/GyroZepelix/mithril-cms` | Owner's GitHub namespace |
| Content storage | Dynamic PostgreSQL tables per content type (`ct_{name}`) | Native filtering/sorting/pagination, clean full-text search |
| Router | `go-chi/chi/v5` | stdlib-compatible, good middleware ecosystem |
| Database driver | `jackc/pgx/v5` | Best PostgreSQL driver for Go, connection pooling built-in |
| System migrations | `golang-migrate/migrate/v4` | Embedded SQL migrations for system tables |
| Content type DDL | Custom schema engine | Generates CREATE/ALTER TABLE from YAML diffs |
| YAML parsing | `gopkg.in/yaml.v3` | Standard Go YAML library |
| Password hashing | `alexedwards/argon2id` | Argon2id, OWASP recommended |
| Auth | JWT access (15min) + refresh tokens (httpOnly cookie, 7d, rotation) | Secure, stateless access with server-side refresh |
| Image processing | `disintegration/imaging` | Pure Go, no C deps |
| Logging | `log/slog` (stdlib) | Structured logging, no external dependency |
| Frontend | React 19 + TypeScript + Vite + Tailwind CSS + shadcn/ui | Modern stack, great DX |
| Embedding | `go:embed` for production; Vite dev proxy for development | Single binary in prod, hot reload in dev |

## Schema Format

Each content type is defined in a single YAML file in the `schema/` directory:

```yaml
name: blog_posts                    # Internal name (snake_case, used in table name and API routes)
display_name: Blog Posts            # Human-readable name for the admin UI
public_read: true                   # Whether entries are readable via the public API
fields:
  - name: title                     # Field name (snake_case)
    type: string                    # Field type (see supported types below)
    required: true                  # NOT NULL constraint
    searchable: true                # Included in full-text search vector
    max_length: 200                 # Validation constraint
  - name: slug
    type: string
    required: true
    unique: true                    # UNIQUE constraint
    regex: "^[a-z0-9]+(?:-[a-z0-9]+)*$"
  - name: body
    type: richtext
    required: true
    searchable: true
  - name: category
    type: enum
    values: [tech, design, business]
  - name: featured
    type: boolean
  - name: author
    type: relation
    relates_to: authors             # Target content type name
    relation_type: one              # "one" = FK column, "many" = junction table
```

### Supported Field Types

| Type | SQL Type | Notes |
|---|---|---|
| `string` | `VARCHAR(n)` or `TEXT` | `max_length` sets VARCHAR limit |
| `text` | `TEXT` | Long-form plain text |
| `richtext` | `TEXT` | Rich text / markdown |
| `int` | `INTEGER` | Supports `min`, `max` validation |
| `float` | `DOUBLE PRECISION` | Supports `min`, `max` validation |
| `boolean` | `BOOLEAN` | Defaults to `false` |
| `date` | `DATE` | ISO 8601 date |
| `time` | `TIME` | ISO 8601 time |
| `enum` | `TEXT` with CHECK | Requires `values` list |
| `json` | `JSONB` | Arbitrary JSON |
| `media` | `UUID` | FK to `media` table |
| `relation` | `UUID` or junction table | Requires `relates_to` and `relation_type` |

### Supported Validations

| Validation | Applies to | Effect |
|---|---|---|
| `required` | All types | NOT NULL + application-level check on create |
| `unique` | All types | UNIQUE constraint |
| `min_length` | string, text, richtext | Application-level validation |
| `max_length` | string, text, richtext | VARCHAR limit + application validation |
| `min` | int, float | Application-level validation |
| `max` | int, float | Application-level validation |
| `regex` | string | Application-level regex match |
| `searchable` | string, text, richtext | Included in tsvector for full-text search |

## Generated Table Structure

For each content type, a dynamic table is created:

```sql
CREATE TABLE ct_{name} (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status        TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','published')),
    -- user-defined fields with types, constraints, and indexes --
    search_vector TSVECTOR,
    created_by    UUID REFERENCES admins(id),
    updated_by    UUID REFERENCES admins(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at  TIMESTAMPTZ
);

CREATE INDEX idx_ct_{name}_status ON ct_{name}(status);
CREATE INDEX idx_ct_{name}_created_at ON ct_{name}(created_at);
CREATE INDEX idx_ct_{name}_search ON ct_{name} USING GIN(search_vector);
-- Plus UNIQUE indexes for unique fields
-- Plus trigger to update search_vector on INSERT/UPDATE
```

Relations:
- `relation_type: "one"` → `UUID` column with FK to `ct_{target}(id)`
- `relation_type: "many"` → junction table `ct_{source}_{field}_rel(source_id, target_id)`

## System Tables

Managed by golang-migrate:

- **`content_types`**: Registry of loaded schemas (name, display_name, schema_hash, fields JSON, created_at, updated_at)
- **`admins`**: Admin users (id, email, password_hash, created_at)
- **`refresh_tokens`**: Refresh token storage (id, admin_id, token_hash, expires_at, created_at)
- **`media`**: Media file records (id, filename, original_name, mime_type, size, width, height, variants JSON, uploaded_by, created_at)
- **`audit_log`**: Audit events (id, action, actor_id, resource, resource_id, payload JSONB, created_at)

## API Routes

### Public API (read-only, published entries, only for `public_read: true` content types)

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/{content-type}` | List entries (paginated, filterable, sortable, searchable) |
| `GET` | `/api/{content-type}/{id}` | Get single published entry |

### Query Parameters (Public List)

- `page` (int, default 1): Page number
- `per_page` (int, default 20, max 100): Items per page
- `sort` (string): Field name to sort by
- `order` (string, `asc`/`desc`, default `desc`): Sort order
- `filter[field]` (string): Filter by field value (exact match)
- `q` (string): Full-text search query

### Admin Auth

| Method | Path | Description |
|---|---|---|
| `POST` | `/admin/api/auth/login` | Login (returns access token + sets refresh cookie) |
| `POST` | `/admin/api/auth/refresh` | Refresh access token (reads httpOnly cookie) |
| `POST` | `/admin/api/auth/logout` | Logout (clears refresh cookie) |
| `GET` | `/admin/api/auth/me` | Get current admin info |

### Admin Content

| Method | Path | Description |
|---|---|---|
| `GET` | `/admin/api/content-types` | List all content types with entry counts |
| `GET` | `/admin/api/content-types/{name}` | Get single content type with full field definitions |
| `GET` | `/admin/api/content/{content-type}` | List entries (paginated, all statuses) |
| `POST` | `/admin/api/content/{content-type}` | Create entry (draft) |
| `GET` | `/admin/api/content/{content-type}/{id}` | Get single entry |
| `PUT` | `/admin/api/content/{content-type}/{id}` | Update entry |
| `POST` | `/admin/api/content/{content-type}/{id}/publish` | Publish entry |

### Admin Media

| Method | Path | Description |
|---|---|---|
| `POST` | `/admin/api/media` | Upload file (multipart) |
| `GET` | `/admin/api/media` | List media (paginated) |
| `DELETE` | `/admin/api/media/{id}` | Delete media file + variants |

### Media Serving (Public)

| Method | Path | Description |
|---|---|---|
| `GET` | `/media/{filename}` | Serve media file (`?v=sm\|md\|lg` for variants) |

### Admin Other

| Method | Path | Description |
|---|---|---|
| `GET` | `/admin/api/audit-log` | List audit events (paginated, filterable) |
| `POST` | `/admin/api/schema/refresh` | Reload schemas and apply safe changes |

### SPA Catch-All

| Method | Path | Description |
|---|---|---|
| `GET` | `/*` | Serves `admin/dist/index.html` for client-side routing |

## Response Formats

### Success (list)
```json
{
  "data": [...],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 42,
    "total_pages": 3
  }
}
```

### Success (single)
```json
{
  "data": { ... }
}
```

### Error
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed",
    "details": [
      { "field": "title", "message": "is required" },
      { "field": "slug", "message": "must match pattern ^[a-z0-9]+(?:-[a-z0-9]+)*$" }
    ]
  }
}
```

## Configuration

All configuration via environment variables:

| Variable | Default | Description |
|---|---|---|
| `MITHRIL_PORT` | `8080` | HTTP server port |
| `MITHRIL_DATABASE_URL` | (required) | PostgreSQL connection string |
| `MITHRIL_SCHEMA_DIR` | `./schema` | Path to YAML schema files |
| `MITHRIL_MEDIA_DIR` | `./media` | Path to media storage directory |
| `MITHRIL_JWT_SECRET` | (required) | Secret key for JWT signing |
| `MITHRIL_DEV_MODE` | `false` | Enable dev mode (auto-apply breaking schema changes, dev proxy) |
| `MITHRIL_ADMIN_EMAIL` | (required on first run) | Initial admin email |
| `MITHRIL_ADMIN_PASSWORD` | (required on first run) | Initial admin password |
