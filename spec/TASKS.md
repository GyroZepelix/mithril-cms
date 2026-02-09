# Mithril CMS — Implementation Tasks

## Task Dependency Graph

```
Task 0 (spec)
  │
Task 1 ─┬─ Task 2 ─┬─ Task 4 ─┬─ Task 7 ─── Task 8
        │          │          │
        ├─ Task 3 ─┘          ├─ Task 11
        │                     │
        │  Task 5 ────────────┤
        │                     │
        │  Task 6 ────────────┤
        │                     │
        │                     ├─ Task 9
        │                     ├─ Task 10
        │                     └─ Task 12
        │
        └─ Task 13 ─┬─ Task 14 (needs 7, 12)
                     ├─ Task 15 (needs 9)
                     └─ Task 16 (needs 10, 11)

Task 17 (needs 7, 8, 9, 10, 11, 14)
Task 18 (needs 17)
```

**Parallelizable groups:**
- Tasks 2 + 3 (after 1)
- Tasks 5 + 13 (after 1)
- Tasks 9 + 10 + 11 + 12 (after 5, 6)
- Tasks 14 + 15 + 16 (after 13 + their backend deps)

---

## Phase 0: Documentation

### Task 0: Write project spec to `spec/` directory
- Write `spec/SPEC.md`: Full project specification
- Write `spec/TASKS.md`: All implementation tasks (this file)
- **Depends on**: Nothing
- **Verify**: `spec/SPEC.md` and `spec/TASKS.md` exist with complete content

---

## Phase 1: Foundation

### Task 1: Initialize Go project scaffold

**Description**: Set up the Go module, directory structure, dependencies, config loading, and build tooling.

**Deliverables**:
- `go.mod` with module `github.com/GyroZepelix/mithril-cms`
- Directory structure: `cmd/mithril/`, `internal/{config,server,schema,database,content,auth,media,audit,search}/`, `migrations/`, `schema/`
- Dependencies installed: chi, pgx, golang-migrate, yaml.v3, argon2id, jwt/v5, imaging
- `internal/config/config.go`: Env-based config struct with fields: Port, DatabaseURL, SchemaDir, MediaDir, JWTSecret, DevMode, AdminEmail, AdminPassword
- `cmd/mithril/main.go`: Minimal entrypoint that loads config and prints startup message
- `Makefile` with targets: `build`, `run`, `test`, `lint`, `clean`
- `.gitignore` for Go + Node artifacts
- `schema/blog_posts.yaml`: Example schema file

**Depends on**: Task 0
**Verify**: `go build ./cmd/mithril/` compiles successfully

---

### Task 2: Database connection and system table migrations

**Description**: Implement PostgreSQL connection pool management and system table migrations.

**Deliverables**:
- `internal/database/database.go`: pgxpool connection with `New(databaseURL)`, `Close()`, `Health()` methods
- `internal/database/migrate.go`: golang-migrate integration using embedded SQL files
- `migrations/000001_initial_schema.up.sql`: Create tables:
  - `content_types` (id UUID PK, name TEXT UNIQUE, display_name TEXT, schema_hash TEXT, fields JSONB, public_read BOOLEAN, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ)
  - `admins` (id UUID PK, email TEXT UNIQUE, password_hash TEXT, created_at TIMESTAMPTZ)
  - `refresh_tokens` (id UUID PK, admin_id UUID FK, token_hash TEXT, expires_at TIMESTAMPTZ, created_at TIMESTAMPTZ)
  - `media` (id UUID PK, filename TEXT UNIQUE, original_name TEXT, mime_type TEXT, size BIGINT, width INT, height INT, variants JSONB, uploaded_by UUID FK, created_at TIMESTAMPTZ)
  - `audit_log` (id UUID PK, action TEXT, actor_id UUID, resource TEXT, resource_id UUID, payload JSONB, created_at TIMESTAMPTZ)
  - Appropriate indexes on all tables
- `migrations/000001_initial_schema.down.sql`: Drop all tables

**Depends on**: Task 1
**Verify**: Migrations run up/down cleanly against a test PostgreSQL instance

---

### Task 3: YAML schema loader and validator

**Description**: Parse YAML schema files, validate their structure, and produce structured Go types.

**Deliverables**:
- `internal/schema/types.go`: Go types: `ContentType`, `Field`, `FieldType` enum constants for all 12 field types, `RelationType` constants
- `internal/schema/loader.go`: `LoadSchemas(dir string) ([]ContentType, error)` — reads all `*.yaml` files, parses them
- `internal/schema/validator.go`: Validates:
  - Content type names: snake_case, no SQL reserved words, no `ct_` prefix collisions
  - Field names: snake_case, no reserved column names (`id`, `status`, `search_vector`, `created_by`, `updated_by`, `created_at`, `updated_at`, `published_at`)
  - Field types: must be one of the supported types
  - Enum fields: must have non-empty `values` list
  - Relation fields: must have `relates_to` and `relation_type`
  - `searchable`: only on string, text, richtext types
  - `min`/`max`: only on int, float types
  - `min_length`/`max_length`: only on string, text, richtext types
  - Relation targets: validated that referenced content types exist in the loaded set
  - Computes SHA256 hash of each file for change detection
- `internal/schema/loader_test.go`: Tests for valid schemas, each invalid case, and error messages

**Depends on**: Task 1
**Verify**: Valid schemas parse correctly; invalid schemas produce actionable error messages

---

### Task 4: Schema-to-SQL migration engine

**Description**: Generate DDL from content type definitions and handle schema evolution.

**Deliverables**:
- `internal/schema/ddl.go`:
  - Field type → SQL type mapping (see spec)
  - `GenerateCreateTable(ct ContentType) string` — full CREATE TABLE with constraints, indexes, search_vector, trigger
  - Constraint generation: NOT NULL, UNIQUE, CHECK (enum), FK (relation/media)
  - Index generation: status, created_at, unique fields, GIN on search_vector
  - Trigger: auto-update search_vector on INSERT/UPDATE using `to_tsvector('english', coalesce(field1,'') || ' ' || coalesce(field2,''))` for searchable fields
- `internal/schema/diff.go`:
  - `DiffSchema(loaded ContentType, existing *ContentType) []Change`
  - Change types: `AddColumn`, `DropColumn`, `AlterColumn`, `AddIndex`, `DropIndex`
  - Safety classification: Safe (add nullable column, add index) vs Breaking (drop column, type change, add NOT NULL to existing column)
- `internal/schema/engine.go`:
  - `Engine` struct with `Apply(ctx, db, schemas)` method
  - Orchestrates: load from DB → diff → classify → apply safe changes → block breaking changes (unless dev mode)
  - Registers new content types in `content_types` table
  - Updates `schema_hash` and `fields` in `content_types` on successful apply
- `internal/schema/ddl_test.go`: Tests for SQL generation
- `internal/schema/diff_test.go`: Tests for change detection and safety classification
- Relation handling:
  - `relation_type: "one"` → `UUID` column with `REFERENCES ct_{target}(id)`
  - `relation_type: "many"` → junction table `ct_{source}_{field}_rel(source_id UUID, target_id UUID, PRIMARY KEY(source_id, target_id))`

**Depends on**: Task 2, Task 3
**Verify**: Schema application creates correct tables; schema changes produce correct ALTER TABLE; breaking changes are detected and blocked (or applied in dev mode)

---

## Phase 2: Core API

### Task 5: HTTP server, router, and middleware

**Description**: Set up the HTTP server with full route tree, middleware stack, and SPA serving.

**Deliverables**:
- `internal/server/server.go`: HTTP server with configurable addr, graceful shutdown (30s timeout), `Start()` and `Shutdown()` methods
- `internal/server/router.go`:
  - Full route tree matching the API routes in the spec
  - Middleware stack: RequestID, RealIP, slog-based request logger, Recoverer, CORS (configurable origins), content-type enforcement (JSON for API routes)
  - `/health` endpoint returning `{"status": "ok"}`
  - Route groups: public API, admin API (auth required), media serving, SPA catch-all
- `internal/server/spa.go`:
  - Production: `go:embed` directive for `admin/dist/*`, serves index.html for non-API/non-media routes
  - Development: reverse proxy to Vite dev server (localhost:5173)
  - Build tag or config flag to switch modes
- `internal/server/response.go`: Helper functions for JSON responses: `JSON(w, status, data)`, `Error(w, status, code, message, details)`, `Paginated(w, data, meta)`
- Update `cmd/mithril/main.go`: Wire config → DB → migrations → schema engine → router → server with graceful shutdown on SIGINT/SIGTERM

**Depends on**: Task 1, Task 2
**Verify**: Server starts, `/health` returns 200 JSON, unknown API routes return 404 JSON, non-API paths would serve SPA (placeholder OK for now)

---

### Task 6: Authentication system

**Description**: Implement admin authentication with JWT access tokens and rotating refresh tokens.

**Deliverables**:
- `internal/auth/service.go`:
  - `EnsureAdmin(ctx, db, email, password)`: Creates initial admin if none exist (used on startup)
  - `HashPassword(password) (string, error)`: Argon2id hashing
  - `VerifyPassword(hash, password) (bool, error)`: Argon2id verification
  - Password policy: minimum 8 characters, maximum 64 characters
- `internal/auth/jwt.go`:
  - `CreateAccessToken(adminID, email, secret) (string, error)`: JWT with `sub` (admin_id), `email` claims, 15min expiry
  - `ValidateAccessToken(tokenString, secret) (*Claims, error)`: Parse and validate
- `internal/auth/handler.go`:
  - `POST /admin/api/auth/login`: Accepts `{email, password}`, returns `{access_token}` + sets `refresh_token` httpOnly cookie (7d, Secure, SameSite=Strict)
  - `POST /admin/api/auth/refresh`: Reads refresh cookie, validates, rotates (delete old + create new), returns new access token + new cookie
  - `POST /admin/api/auth/logout`: Deletes refresh token from DB, clears cookie
  - `GET /admin/api/auth/me`: Returns `{id, email}` of authenticated admin
- `internal/auth/middleware.go`:
  - Extracts `Authorization: Bearer <token>` header
  - Validates JWT, extracts claims
  - Sets admin ID and email in request context
  - Returns 401 JSON on missing/invalid/expired token
- `internal/auth/repository.go`:
  - Admin CRUD: `GetByEmail`, `Create`
  - Refresh tokens: `Create`, `GetByHash`, `Delete`, `DeleteAllForAdmin`
  - Token stored as SHA256 hash of the actual token value

**Depends on**: Task 2, Task 5
**Verify**: Login → access token → protected endpoint works. Wrong password → 401. Expired token → 401 → refresh → new token. Password <8 or >64 → 400.

---

### Task 7: Content CRUD API

**Description**: Implement full content CRUD with dynamic SQL, validation, and pagination.

**Deliverables**:
- `internal/content/handler.go`:
  - `POST /admin/api/content/{content-type}`: Create entry (draft)
  - `GET /admin/api/content/{content-type}`: List entries (admin, all statuses)
  - `GET /admin/api/content/{content-type}/{id}`: Get single entry (admin)
  - `PUT /admin/api/content/{content-type}/{id}`: Update entry
  - `POST /admin/api/content/{content-type}/{id}/publish`: Set status=published, published_at=now()
  - `GET /api/{content-type}`: Public list (published only, public_read types only)
  - `GET /api/{content-type}/{id}`: Public get (published only)
- `internal/content/service.go`:
  - Business logic layer between handlers and repository
  - Validates content type exists and is accessible
  - Delegates to validation, repository, and audit
- `internal/content/repository.go`:
  - Dynamic SQL generation for INSERT, UPDATE, SELECT with parameterized queries
  - Column names validated against schema field whitelist (prevents SQL injection)
  - All queries use `$1, $2, ...` placeholders
- `internal/content/validation.go`:
  - `ValidateEntry(schema, data map[string]any, isUpdate bool) []FieldError`
  - Checks: required (on create), type coercion, min/max, min_length/max_length, regex, enum values
- `internal/content/query.go`:
  - Query builder for list endpoints
  - Supports: `?page=&per_page=&sort=&order=&filter[field]=value&q=search`
  - `sort` validated against schema fields
  - `filter[field]` validated against schema fields
  - Default: `page=1, per_page=20, sort=created_at, order=desc`
  - Max per_page: 100
- Response format per spec: `{"data": [...], "meta": {"page", "per_page", "total", "total_pages"}}`

**Depends on**: Task 4, Task 5, Task 6
**Verify**: Full CRUD cycle: create draft → update → publish → visible on public API. Validation errors return 400 with field-level details. Pagination, filtering, sorting work correctly.

---

### Task 8: Full-text search integration

**Description**: Integrate PostgreSQL full-text search into content listing endpoints.

**Deliverables**:
- `internal/search/search.go`:
  - `BuildSearchClause(query string, searchableFields []string) (whereClause, orderClause string, args []any)`
  - Uses `plainto_tsquery('english', $N)` for the query
  - Ranks results with `ts_rank(search_vector, query)`
  - Generates headline snippets with `ts_headline('english', field, query)` for the first searchable field
- Integration into `internal/content/query.go`:
  - When `?q=` is present, adds search WHERE clause and ORDER BY rank
  - Returns `_search_headline` field in results
- Works on both public and admin list endpoints

**Depends on**: Task 4, Task 7
**Verify**: Insert entries → search by keyword → ranked results returned with highlights

---

## Phase 3: Supporting Features

### Task 9: Media upload, processing, and serving

**Description**: Handle file uploads, generate image variants, and serve media files.

**Deliverables**:
- `internal/media/handler.go`:
  - `POST /admin/api/media`: Multipart upload, returns media object with URLs
  - `GET /admin/api/media`: List media (paginated)
  - `DELETE /admin/api/media/{id}`: Delete media + all variant files
  - `GET /media/{filename}`: Serve file with `?v=sm|md|lg` variant support
- `internal/media/service.go`:
  - Validates: file size ≤ 10MB, MIME type in allowlist (image/jpeg, image/png, image/gif, image/webp, application/pdf, text/plain, etc.)
  - Saves original file
  - For images: generates 3 variants using `disintegration/imaging`:
    - `sm`: max width 480px (proportional)
    - `md`: max width 1024px (proportional)
    - `lg`: max width 1920px (proportional)
  - Records metadata: filename, original_name, mime_type, size, width, height, variants JSON
- `internal/media/repository.go`: CRUD for media records
- `internal/media/storage.go`:
  - Local filesystem storage: `{media_dir}/original/{uuid}.{ext}`, `{media_dir}/{variant}/{uuid}.{ext}`
  - File serving with correct Content-Type and Cache-Control headers (1 year for immutable content-addressed files)

**Depends on**: Task 2, Task 5, Task 6
**Verify**: Upload image → 3 variants created on disk. Serve via URL with correct Content-Type. Non-image files: original only, no variants.

---

### Task 10: Audit logging system

**Description**: Record and expose all significant admin actions.

**Deliverables**:
- `internal/audit/service.go`:
  - `Log(ctx, Event)` — Event has: Action, ActorID, Resource, ResourceID, Payload (map[string]any)
  - Uses buffered channel (size 100) for async writes
  - Background goroutine flushes buffer to DB
  - Graceful shutdown: drain channel before exit
- `internal/audit/handler.go`:
  - `GET /admin/api/audit-log`: Paginated list with filters `?action=&resource=&page=&per_page=`
- `internal/audit/repository.go`:
  - `Insert(ctx, event)` and `List(ctx, filters, page, perPage)` with total count
- Tracked events:
  - `entry.create`, `entry.update`, `entry.publish`
  - `schema.refresh`
  - `admin.login.success`, `admin.login.failure`
  - `media.upload`, `media.delete`

**Depends on**: Task 2, Task 5, Task 6
**Verify**: Actions generate audit entries. Audit API returns filtered, paginated results.

---

### Task 11: Schema refresh endpoint and CLI

**Description**: Enable runtime schema reloading and provide CLI commands for schema management.

**Deliverables**:
- Extend `internal/schema/engine.go`:
  - `Refresh(ctx, db, schemaDir) (*RefreshResult, error)` — reload schemas, diff, apply safe, report breaking
  - `RefreshResult`: lists of applied changes, pending breaking changes, new content types
- Update `cmd/mithril/main.go` with subcommands:
  - `mithril serve` (default): Start the server
  - `mithril schema diff`: Load schemas, diff against DB, print changes (no apply)
  - `mithril schema apply`: Apply all safe changes
  - `mithril schema apply --force`: Apply all changes including breaking
- `POST /admin/api/schema/refresh` handler:
  - Calls `engine.Refresh()`
  - Returns 200 with applied changes on success
  - Returns 409 with breaking change details if any exist

**Depends on**: Task 4, Task 5, Task 6
**Verify**: `mithril schema diff` shows pending changes. Safe changes auto-apply. Breaking changes blocked with clear messages.

---

### Task 12: Content type introspection API

**Description**: Expose content type metadata for the admin UI to dynamically build forms and navigation.

**Deliverables**:
- `GET /admin/api/content-types` handler:
  - Returns all content types with: name, display_name, public_read, field count, entry count (via COUNT query)
- `GET /admin/api/content-types/{name}` handler:
  - Returns full content type definition with all field details (name, type, required, unique, searchable, validations, enum values, relation info)

**Depends on**: Task 3, Task 5, Task 6
**Verify**: Returns all registered content types matching YAML schemas with accurate entry counts.

---

## Phase 4: Admin UI

### Task 13: Initialize React admin project

**Description**: Bootstrap the React admin SPA with routing, auth, and layout.

**Deliverables**:
- `admin/` bootstrapped with Vite + React 19 + TypeScript + Tailwind CSS v4 + shadcn/ui
- `admin/src/lib/api.ts`: Fetch wrapper that:
  - Adds Authorization Bearer header from in-memory token
  - On 401 response: attempts token refresh, retries original request
  - Throws typed errors for UI consumption
- `admin/src/lib/auth.tsx`: AuthProvider + useAuth hook:
  - Stores access token in memory (not localStorage)
  - On mount: attempts silent refresh (POST /admin/api/auth/refresh)
  - Provides: login, logout, isAuthenticated, user
- `admin/src/components/layout/AppLayout.tsx`:
  - Sidebar: dynamic navigation from content types API, plus Media and Audit Log links
  - Header: breadcrumb, user email, logout button
- `admin/src/pages/LoginPage.tsx`: Email + password form, error display
- Routing (React Router):
  - `/admin/login` → LoginPage (public)
  - `/admin/content/:type` → ContentListPage (protected)
  - `/admin/content/:type/new` → ContentEditPage (protected)
  - `/admin/content/:type/:id` → ContentEditPage (protected)
  - `/admin/media` → MediaPage (protected)
  - `/admin/audit-log` → AuditLogPage (protected)
- `vite.config.ts`: Proxy `/api` and `/admin/api` and `/media` to `localhost:8080`

**Depends on**: Task 1
**Verify**: `npm run dev` starts, login page renders, auth flow works, sidebar loads content types from API.

---

### Task 14: Dynamic content list and edit pages

**Description**: Build the content management UI that dynamically adapts to schema definitions.

**Deliverables**:
- `ContentListPage.tsx`: Data table with:
  - Columns derived from schema fields
  - Sortable column headers
  - Pagination controls
  - Search input (triggers `?q=`)
  - Status badges (draft/published)
  - "New" button → navigates to create page
  - Row click → navigates to edit page
- `ContentEditPage.tsx`:
  - Loads schema for content type
  - Loads entry data (if editing)
  - Renders dynamic form based on schema fields
  - Save (draft) and Publish buttons
  - Inline validation error display
  - Breadcrumb navigation
- `ContentForm.tsx`: Maps schema field types to components
- Field components:
  - `StringField`: Text input with max length indicator
  - `TextField`: Textarea
  - `RichTextField`: Textarea with markdown preview toggle
  - `NumberField`: Number input with min/max
  - `BooleanField`: Switch/checkbox
  - `DateField`: Date picker
  - `TimeField`: Time picker
  - `EnumField`: Select dropdown from schema values
  - `JSONField`: Code editor textarea with JSON validation
  - `MediaField`: Upload button + preview thumbnail + clear
  - `RelationField`: Searchable select that queries related content type
- Uses React Hook Form for form state management

**Depends on**: Task 7, Task 12, Task 13
**Verify**: List/create/edit/publish cycle works through UI. All field types render correctly. Validation errors display inline.

---

### Task 15: Media management page

**Description**: Build the media library UI for managing uploads.

**Deliverables**:
- `MediaPage.tsx`: Grid layout of uploaded files with:
  - Thumbnails for images
  - File icon for non-images
  - Filename, size, upload date
  - Pagination
- `MediaUploader.tsx`: Drag-and-drop upload zone with:
  - File type and size validation (client-side)
  - Upload progress indicator
  - Success/error feedback
- `MediaDetail.tsx`: Detail panel (sidebar or modal) with:
  - Full preview for images
  - Variant previews (sm, md, lg) with dimensions
  - Copyable URL for each variant
  - File metadata (name, type, size, dimensions, uploaded by, date)
  - Delete button with confirmation dialog

**Depends on**: Task 9, Task 13
**Verify**: Upload via drag-and-drop, grid displays thumbnails, detail panel shows variants, delete works.

---

### Task 16: Audit log page and schema refresh UI

**Description**: Build audit log viewer and schema management UI.

**Deliverables**:
- `AuditLogPage.tsx`: Data table with:
  - Columns: timestamp, action, actor (email), resource, resource ID
  - Filterable by action (dropdown) and resource (dropdown)
  - Expandable rows showing full payload JSON
  - Pagination
- Schema refresh UI:
  - Button in sidebar or settings area
  - Triggers `POST /admin/api/schema/refresh`
  - Shows success message with list of applied changes
  - Shows error/warning for breaking changes with details

**Depends on**: Task 10, Task 11, Task 13
**Verify**: Audit events display with filters. Schema refresh triggers correctly with success/failure feedback.

---

## Phase 5: Integration & Deployment

### Task 17: Embed React into Go binary + integration tests

**Description**: Finalize the production build pipeline and write integration tests.

**Deliverables**:
- Finalize `internal/server/spa.go`: `go:embed` for `admin/dist/*` with build tag `embed_admin`
- `Makefile` targets:
  - `build-admin`: `cd admin && npm ci && npm run build`
  - `build`: `build-admin` + `go build -tags embed_admin -o mithril ./cmd/mithril/`
  - `dev`: Run Go server + Vite dev server concurrently
- Integration test suite (`internal/integration/`):
  - Starts test PostgreSQL (testcontainers or expects running instance)
  - Runs migrations
  - Loads test schemas
  - Seeds admin user
  - Tests full flow: login → CRUD → publish → public API → media upload → search → audit log
- All tests pass

**Depends on**: Tasks 7, 8, 9, 10, 11, 14
**Verify**: `make build` produces single binary. Binary serves SPA + API. Integration tests pass.

---

### Task 18: Docker deployment and documentation

**Description**: Create Docker setup and comprehensive documentation.

**Deliverables**:
- `Dockerfile`: Multi-stage build:
  - Stage 1 (Node): Build admin SPA
  - Stage 2 (Go): Build Go binary with embedded SPA
  - Stage 3 (Alpine): Runtime with minimal image
- `docker-compose.yml`:
  - Mithril service: build from Dockerfile, env vars, ports, volumes (schema/, media/), depends_on postgres with health check
  - PostgreSQL service: postgres:16-alpine, health check, named volume for data
- `.dockerignore`: Exclude .git, node_modules, media/, etc.
- Update `README.md` with:
  - Project overview and features
  - Quick start (docker compose)
  - Development setup (Go + Node prerequisites, make dev)
  - Configuration reference (all env vars)
  - Schema format reference (all field types and validations)
  - API reference (all routes with examples)
  - Production deployment notes

**Depends on**: Task 17
**Verify**: `docker compose up` starts everything. Admin UI accessible at `:8080/admin`. Full workflow works end-to-end.
