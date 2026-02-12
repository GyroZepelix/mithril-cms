# Mithril CMS - API Reference

This document covers the full HTTP API for Neo-Mithril, a headless CMS. All endpoints return JSON (`application/json; charset=utf-8`). The base URL depends on your deployment (default: `http://localhost:8080`).

---

## Table of Contents

- [Authentication](#authentication)
- [Public Content API](#public-content-api)
- [Admin API](#admin-api)
  - [Auth](#auth)
  - [Content CRUD](#content-crud)
  - [Media Management](#media-management)
  - [Content Types (Introspection)](#content-types-introspection)
  - [Audit Log](#audit-log)
  - [Schema Refresh](#schema-refresh)
- [Public Media Serving](#public-media-serving)
- [Health Check](#health-check)
- [Response Formats](#response-formats)
- [Query Parameters Reference](#query-parameters-reference)
- [Field Types Reference](#field-types-reference)
- [Media Variants](#media-variants)

---

## Authentication

Mithril uses **JWT-based authentication** with short-lived access tokens and long-lived refresh tokens.

- **Access token**: Passed as `Authorization: Bearer <token>` header. Short-lived.
- **Refresh token**: Stored as an `httpOnly` cookie (`refresh_token`), scoped to `/admin/api/auth`. Valid for 7 days. Rotated on each refresh.

### Login Flow

1. `POST /admin/api/auth/login` with email + password.
2. Receive `access_token` in the response body. A `refresh_token` cookie is set automatically.
3. Use the access token in the `Authorization` header for all protected endpoints.
4. When the access token expires, call `POST /admin/api/auth/refresh` (the browser sends the cookie automatically).
5. On logout, call `POST /admin/api/auth/logout` to invalidate the refresh token.

---

## Public Content API

These endpoints require **no authentication**. They only return **published** entries for content types that have `public_read: true` in their schema.

All requests to `/api/*` must include the `Content-Type: application/json` header.

### List Published Entries

```
GET /api/{contentType}
```

Returns a paginated list of published entries.

**Query Parameters**: See [Query Parameters Reference](#query-parameters-reference).

**Response** `200 OK`:

```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "Hello World",
      "status": "published",
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-15T12:00:00Z",
      "published_at": "2025-01-15T12:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 42,
    "total_pages": 3
  }
}
```

**Errors**:

| Status | Code | Condition |
|--------|------|-----------|
| 404 | `NOT_FOUND` | Content type does not exist or `public_read` is false |
| 400 | `INVALID_PARAMS` | Invalid query parameters |

### Get Single Published Entry

```
GET /api/{contentType}/{id}
```

**Path Parameters**:

- `id` (UUID) - Entry ID.

**Response** `200 OK`:

```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "Hello World",
    "body": "<p>Content here...</p>",
    "status": "published",
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T12:00:00Z",
    "published_at": "2025-01-15T12:00:00Z"
  }
}
```

**Errors**:

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `INVALID_ID` | ID is not a valid UUID |
| 404 | `NOT_FOUND` | Entry not found, not published, or content type not public |

---

## Admin API

All admin endpoints are under `/admin/api`. Requests must include the `Content-Type: application/json` header.

Protected endpoints (all except login, refresh, logout) require:

```
Authorization: Bearer <access_token>
```

### Auth

#### Login

```
POST /admin/api/auth/login
```

**Request Body**:

```json
{
  "email": "admin@example.com",
  "password": "your-password"
}
```

**Response** `200 OK`:

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

A `refresh_token` httpOnly cookie is also set.

**Errors**:

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `VALIDATION_ERROR` | Email or password missing |
| 401 | `UNAUTHORIZED` | Invalid credentials |

#### Refresh Token

```
POST /admin/api/auth/refresh
```

No request body. The refresh token is read from the `refresh_token` cookie.

**Response** `200 OK`:

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

A new `refresh_token` cookie replaces the old one (token rotation).

**Errors**:

| Status | Code | Condition |
|--------|------|-----------|
| 401 | `UNAUTHORIZED` | Missing, empty, invalid, or expired refresh token |

#### Logout

```
POST /admin/api/auth/logout
```

No request body. Invalidates the refresh token and clears the cookie.

**Response** `200 OK`:

```json
{
  "data": {
    "message": "logged out"
  }
}
```

#### Get Current User

```
GET /admin/api/auth/me
```

**Auth**: Required.

**Response** `200 OK`:

```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "admin@example.com"
  }
}
```

### Content CRUD

All content endpoints are protected and operate on entries of a specific content type.

#### List Entries (Admin)

```
GET /admin/api/content/{contentType}
```

Returns all entries (draft + published) with pagination.

**Query Parameters**: See [Query Parameters Reference](#query-parameters-reference).

**Response** `200 OK`: Same paginated format as [public list](#list-published-entries), but includes draft entries.

#### Get Entry (Admin)

```
GET /admin/api/content/{contentType}/{id}
```

Returns a single entry regardless of publish status.

**Response** `200 OK`:

```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "Draft Post",
    "status": "draft",
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T10:30:00Z",
    "published_at": null,
    "created_by": "admin-uuid",
    "updated_by": "admin-uuid"
  }
}
```

**Errors**:

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `INVALID_ID` | ID is not a valid UUID |
| 404 | `NOT_FOUND` | Entry or content type not found |

#### Create Entry

```
POST /admin/api/content/{contentType}
```

**Request Body**: A JSON object with field values matching the content type schema.

```json
{
  "title": "My New Post",
  "body": "<p>Hello world</p>",
  "category": "tech"
}
```

**Response** `201 Created`:

```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "My New Post",
    "body": "<p>Hello world</p>",
    "category": "tech",
    "status": "draft",
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T10:30:00Z",
    "created_by": "admin-uuid",
    "updated_by": "admin-uuid"
  }
}
```

**Errors**:

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `INVALID_JSON` | Malformed or too-large request body (max 1 MiB) |
| 400 | `VALIDATION_ERROR` | Field validation failed (see `error.details`) |
| 404 | `NOT_FOUND` | Content type not found |

#### Update Entry

```
PUT /admin/api/content/{contentType}/{id}
```

**Request Body**: A JSON object with updated field values. Only include fields you want to change.

```json
{
  "title": "Updated Title"
}
```

**Response** `200 OK`: Returns the full updated entry (same shape as create).

**Errors**: Same as [Create Entry](#create-entry), plus `INVALID_ID` for bad UUIDs.

#### Publish Entry

```
POST /admin/api/content/{contentType}/{id}/publish
```

No request body. Sets the entry status to `published` and records `published_at`.

**Response** `200 OK`: Returns the full published entry.

**Errors**:

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `INVALID_ID` | ID is not a valid UUID |
| 404 | `NOT_FOUND` | Entry or content type not found |

### Media Management

#### Upload Media

```
POST /admin/api/media
```

**Content-Type**: `multipart/form-data` (not JSON).

**Form Fields**:

- `file` (required) - The file to upload. Max 10 MiB.

**Allowed MIME types**: `image/jpeg`, `image/png`, `image/gif`, `image/webp`, `application/pdf`, `text/plain`, `text/csv`, `application/json`.

For image uploads, resized variants (sm, md, lg) are generated automatically.

**Response** `201 Created`:

```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "filename": "a1b2c3d4e5f6.jpg",
    "original_name": "photo.jpg",
    "mime_type": "image/jpeg",
    "size": 245760,
    "width": 3000,
    "height": 2000,
    "variants": {
      "sm": "sm/a1b2c3d4e5f6.jpg",
      "md": "md/a1b2c3d4e5f6.jpg",
      "lg": "lg/a1b2c3d4e5f6.jpg"
    },
    "uploaded_by": "admin-uuid",
    "created_at": "2025-01-15T10:30:00Z"
  }
}
```

**Errors**:

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `INVALID_UPLOAD` | Failed to parse multipart form or file too large |
| 400 | `MISSING_FILE` | No `file` field in the form |
| 400 | `UPLOAD_ERROR` | Disallowed MIME type or other validation failure |

#### List Media

```
GET /admin/api/media
```

**Query Parameters**:

| Param | Default | Description |
|-------|---------|-------------|
| `page` | `1` | Page number (positive integer) |
| `per_page` | `20` | Items per page (1-100) |

**Response** `200 OK`:

```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "filename": "a1b2c3d4e5f6.jpg",
      "original_name": "photo.jpg",
      "mime_type": "image/jpeg",
      "size": 245760,
      "width": 3000,
      "height": 2000,
      "variants": {
        "sm": "sm/a1b2c3d4e5f6.jpg",
        "md": "md/a1b2c3d4e5f6.jpg",
        "lg": "lg/a1b2c3d4e5f6.jpg"
      },
      "uploaded_by": "admin-uuid",
      "created_at": "2025-01-15T10:30:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 5,
    "total_pages": 1
  }
}
```

#### Delete Media

```
DELETE /admin/api/media/{id}
```

**Path Parameters**:

- `id` (UUID) - Media record ID.

**Response** `200 OK`:

```json
{
  "data": {
    "message": "deleted"
  }
}
```

**Errors**:

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `INVALID_ID` | ID is not a valid UUID |
| 404 | `NOT_FOUND` | Media not found |

### Content Types (Introspection)

Discover available content types and their field schemas.

#### List All Content Types

```
GET /admin/api/content-types
```

**Response** `200 OK`:

```json
{
  "data": [
    {
      "name": "posts",
      "display_name": "Blog Posts",
      "public_read": true,
      "fields": [
        {
          "name": "title",
          "type": "string",
          "required": true,
          "unique": false,
          "searchable": true,
          "max_length": 200
        },
        {
          "name": "body",
          "type": "richtext",
          "required": true,
          "unique": false,
          "searchable": true
        },
        {
          "name": "category",
          "type": "enum",
          "required": false,
          "unique": false,
          "searchable": false,
          "values": ["tech", "design", "news"]
        }
      ],
      "entry_count": 42
    }
  ]
}
```

#### Get Single Content Type

```
GET /admin/api/content-types/{name}
```

**Path Parameters**:

- `name` - Content type name (e.g., `posts`).

**Response** `200 OK`: Same shape as a single item from the list.

**Errors**:

| Status | Code | Condition |
|--------|------|-----------|
| 404 | `NOT_FOUND` | Content type not found |

### Audit Log

```
GET /admin/api/audit-log
```

Returns a paginated list of audit log entries.

**Query Parameters**:

| Param | Default | Description |
|-------|---------|-------------|
| `page` | `1` | Page number (positive integer) |
| `per_page` | `20` | Items per page (1-100) |
| `action` | - | Filter by action (exact match, e.g., `admin.login.success`) |
| `resource` | - | Filter by resource (exact match, e.g., `posts`) |

**Response** `200 OK`:

```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "action": "content.create",
      "actor_id": "admin-uuid",
      "resource": "posts",
      "resource_id": "entry-uuid",
      "payload": {
        "title": "New Post"
      },
      "created_at": "2025-01-15T10:30:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

### Schema Refresh

```
POST /admin/api/schema/refresh
```

Reloads content type schemas from YAML files on disk, diffs against the database, and applies non-breaking changes. Breaking changes block the refresh.

No request body.

**Response** `200 OK`:

```json
{
  "data": {
    "applied": [
      {
        "type": "add_column",
        "table": "ct_posts",
        "column": "subtitle",
        "detail": "added column subtitle (VARCHAR(255))",
        "safe": true
      }
    ],
    "new_types": ["faqs"],
    "updated_types": ["posts"]
  }
}
```

**Response** `409 Conflict` (breaking changes detected):

```json
{
  "error": {
    "code": "BREAKING_CHANGES",
    "message": "schema refresh blocked due to breaking changes",
    "details": [
      {
        "field": "ct_posts.category",
        "message": "cannot remove column that contains data"
      }
    ]
  }
}
```

---

## Public Media Serving

```
GET /media/{filename}
```

Serves a media file by its generated filename. No authentication required.

**Query Parameters**:

| Param | Default | Description |
|-------|---------|-------------|
| `v` | `original` | Variant to serve: `sm`, `md`, `lg`, or `original` |

If the requested variant does not exist (e.g., for non-image files), the original is served as a fallback.

Images are served inline. Non-image files are served as downloads (`Content-Disposition: attachment`).

**Cache**: `Cache-Control: public, max-age=31536000, immutable`.

**Errors**:

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `INVALID_VARIANT` | Variant is not one of: `original`, `sm`, `md`, `lg` |
| 404 | `NOT_FOUND` | File not found |

---

## Health Check

```
GET /health
```

No authentication required.

**Response** `200 OK`:

```json
{
  "status": "ok"
}
```

**Response** `503 Service Unavailable`:

```json
{
  "error": {
    "code": "DB_UNHEALTHY",
    "message": "database health check failed"
  }
}
```

> Note: The health endpoint does **not** use the `{"data": ...}` envelope.

---

## Response Formats

### Success (single item)

```json
{
  "data": { ... }
}
```

### Success (paginated list)

```json
{
  "data": [ ... ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 42,
    "total_pages": 3
  }
}
```

### Error

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed",
    "details": [
      {
        "field": "title",
        "message": "is required"
      }
    ]
  }
}
```

The `details` array is only present for validation errors that have field-level information.

### Common Error Codes

| Code | Meaning |
|------|---------|
| `INVALID_JSON` | Malformed or too-large request body |
| `INVALID_PARAMS` | Bad query parameters |
| `INVALID_ID` | ID is not a valid UUID |
| `VALIDATION_ERROR` | Field validation failed (check `details`) |
| `UNAUTHORIZED` | Missing or invalid authentication |
| `NOT_FOUND` | Resource not found |
| `NOT_IMPLEMENTED` | Endpoint not yet available |
| `INTERNAL_ERROR` | Unexpected server error |
| `BREAKING_CHANGES` | Schema refresh blocked (409) |
| `DB_UNHEALTHY` | Database health check failed (503) |

---

## Query Parameters Reference

These parameters apply to content list endpoints (`GET /api/{contentType}` and `GET /admin/api/content/{contentType}`).

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | `1` | Page number (must be >= 1) |
| `per_page` | int | `20` | Items per page (1-100, clamped at 100) |
| `sort` | string | `created_at` | Field to sort by. Must be a user-defined field or a system column: `id`, `status`, `created_at`, `updated_at`, `published_at`, `created_by`, `updated_by` |
| `order` | string | `desc` | Sort direction: `asc` or `desc` |
| `filter[field]` | string | - | Exact-match filter on a field. Example: `filter[status]=published` |
| `q` | string | - | Full-text search across fields marked as `searchable` |

**Filter example**:

```
GET /api/posts?filter[category]=tech&sort=title&order=asc&page=2&per_page=10
```

---

## Field Types Reference

Content type fields are defined in YAML schemas. Each type maps to a specific SQL type and has its own validation rules.

| Type | SQL Type | Description | Validation Options |
|------|----------|-------------|-------------------|
| `string` | `VARCHAR(n)` | Short text | `required`, `unique`, `searchable`, `min_length`, `max_length`, `regex` |
| `text` | `TEXT` | Long plain text | `required`, `unique`, `searchable`, `min_length`, `max_length` |
| `richtext` | `TEXT` | HTML rich text | `required`, `unique`, `searchable`, `min_length`, `max_length` |
| `int` | `INTEGER` | Integer number | `required`, `unique`, `min`, `max` |
| `float` | `DOUBLE PRECISION` | Decimal number | `required`, `unique`, `min`, `max` |
| `boolean` | `BOOLEAN` | True/false | `required` |
| `date` | `DATE` | Calendar date | `required`, `unique` |
| `time` | `TIMESTAMPTZ` | Timestamp | `required`, `unique` |
| `enum` | `VARCHAR(255)` | Predefined values | `required`, `values` (list of allowed strings) |
| `json` | `JSONB` | Arbitrary JSON | `required` |
| `media` | `UUID` (FK) | Reference to media | `required` |
| `relation` | `UUID` / `UUID[]` | Reference to another content type | `required`, `relates_to`, `relation_type` (`one` or `many`) |

---

## Media Variants

When an image is uploaded (`image/jpeg`, `image/png`, `image/gif`, `image/webp`), the server automatically generates resized variants:

| Variant | Max Width | Usage |
|---------|-----------|-------|
| `sm` | 480px | Thumbnails, mobile |
| `md` | 1024px | Cards, content previews |
| `lg` | 1920px | Full-width, hero images |
| `original` | - | Unmodified upload |

Aspect ratio is preserved. If the original image is smaller than a variant's max width, that variant is not generated.

To request a specific variant, use the `v` query parameter:

```
GET /media/a1b2c3d4e5f6.jpg?v=md
```

Non-image files (PDF, CSV, etc.) do not have variants; the `v` parameter defaults to `original`.
