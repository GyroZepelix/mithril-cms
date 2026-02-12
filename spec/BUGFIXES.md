# Bug Fixes — Admin UI Manual Testing

## Bug 1: White screen after login redirect

**Symptom**: After logging in, the app redirects to `/admin/content` but shows a white screen. Refreshing the page fixes it.

**Root cause**: `Header.tsx:17` accesses `state.admin.email` without a null guard. During the login-to-authenticated transition, there is a React render cycle where `state.status` is `"authenticated"` but `state.admin` hasn't propagated to all components yet. This crashes the Header component, killing the entire component tree.

**Error**: `Uncaught TypeError: Cannot read properties of undefined (reading 'email')`

**Fix**:
- `admin/src/lib/auth.tsx`: The login function now fetches `/admin/api/auth/me` after obtaining the access token to populate admin data (the backend login endpoint only returns `access_token`, not admin info)
- `admin/src/components/layout/Header.tsx`: Change `state.admin.email` → `state.admin?.email ?? ""` (defensive)
- `admin/src/components/layout/AppLayout.tsx`: Add guard — if `status === "authenticated"` but `admin` is missing, show a loading spinner (defensive fallback)

---

## Bug 2: UUID serialized as byte array (Critical)

**Symptom**: Creating a content entry and clicking publish produces a URL like `/admin/content/authors/206,7,190,224,6,59,69,190,...` and shows "id must be a valid UUID".

**Root cause**: `pgx.RowToMap` scans PostgreSQL UUID columns as `[16]byte` instead of strings. Go's `encoding/json` then serializes these as integer arrays. The frontend receives `[206,7,190,...]` instead of `"ce07bee0-063b-45be-..."`.

**Fix**:
- `internal/content/repository.go`: Add `normalizeRow`/`normalizeRows` helpers that convert `[16]byte` and `pgtype.UUID` values to UUID string format. Applied at all 5 `pgx.RowToMap` call sites (`List`, `GetByID`, `Insert`, `Update`, `Publish`).

**Bonus**: This also fixes audit logging — `service.go:114` does `entry["id"].(string)` which silently fails when ID is `[16]byte`, meaning audit events for `entry.create` currently don't record the resource ID.

---

## Bug 3: Media copy URL buttons missing padding

**Symptom**: In the media detail dialog, the "Original" and "Small (480px)" copy URL buttons appear cramped/missing padding.

**Root cause**: The `CopyButton` in `MediaDetail.tsx` has `className="h-7 gap-1.5 text-xs"` which overrides the shadcn Button's `size="sm"` height but doesn't explicitly set horizontal padding.

**Fix**:
- `admin/src/components/media/MediaDetail.tsx`: Change `className="h-7 gap-1.5 text-xs"` → `className="h-7 gap-1.5 px-3 text-xs"`
