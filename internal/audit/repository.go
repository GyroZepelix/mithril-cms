// Package audit provides audit logging for significant admin actions in
// the Mithril CMS. Events are written asynchronously to the audit_log table
// so that logging never blocks or fails API requests.
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/GyroZepelix/mithril-cms/internal/database"
)

// AuditEntry represents a single row in the audit_log table.
type AuditEntry struct {
	ID         string         `json:"id"`
	Action     string         `json:"action"`
	ActorID    *string        `json:"actor_id,omitempty"`
	Resource   *string        `json:"resource,omitempty"`
	ResourceID *string        `json:"resource_id,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// AuditFilters holds optional filter parameters for listing audit entries.
type AuditFilters struct {
	Action   string // filter by action (exact match)
	Resource string // filter by resource (exact match)
}

// Repository provides database operations for the audit_log table.
type Repository struct {
	db *database.DB
}

// NewRepository creates a new audit Repository.
func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// Insert writes a single audit event to the database. Empty string values for
// ActorID, Resource, and ResourceID are stored as NULL.
func (r *Repository) Insert(ctx context.Context, event Event) error {
	var payloadJSON []byte
	if event.Payload != nil {
		var err error
		payloadJSON, err = json.Marshal(event.Payload)
		if err != nil {
			return fmt.Errorf("marshaling audit payload: %w", err)
		}
	}

	_, err := r.db.Pool().Exec(ctx,
		`INSERT INTO audit_log (action, actor_id, resource, resource_id, payload)
		 VALUES ($1, $2, $3, $4, $5)`,
		event.Action,
		nullIfEmpty(event.ActorID),
		nullIfEmpty(event.Resource),
		nullIfEmpty(event.ResourceID),
		nullableJSON(payloadJSON),
	)
	if err != nil {
		return fmt.Errorf("inserting audit event: %w", err)
	}
	return nil
}

// List retrieves a paginated, filtered list of audit entries ordered by
// created_at DESC. It returns the entries, total count, and any error.
// The caller (service/handler layer) is responsible for validating and
// clamping pagination parameters.
func (r *Repository) List(ctx context.Context, filters AuditFilters, page, perPage int) ([]*AuditEntry, int, error) {
	// Build WHERE clause from filters.
	// SECURITY: All column names below are hardcoded constants and must never
	// be replaced with user input. Filter values are parameterized ($1, $2...).
	var conditions []string
	var args []any
	paramIdx := 1

	if filters.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", paramIdx))
		args = append(args, filters.Action)
		paramIdx++
	}
	if filters.Resource != "" {
		conditions = append(conditions, fmt.Sprintf("resource = $%d", paramIdx))
		args = append(args, filters.Resource)
		paramIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total matching rows.
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_log %s", whereClause)
	var total int
	if err := r.db.Pool().QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting audit entries: %w", err)
	}

	// Fetch the page.
	offset := (page - 1) * perPage
	selectQuery := fmt.Sprintf(
		`SELECT id, action, actor_id, resource, resource_id, payload, created_at
		 FROM audit_log %s
		 ORDER BY created_at DESC
		 LIMIT $%d OFFSET $%d`,
		whereClause, paramIdx, paramIdx+1,
	)
	args = append(args, perPage, offset)

	rows, err := r.db.Pool().Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying audit entries: %w", err)
	}
	defer rows.Close()

	entries, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*AuditEntry, error) {
		var e AuditEntry
		var payloadJSON []byte
		if err := row.Scan(&e.ID, &e.Action, &e.ActorID, &e.Resource, &e.ResourceID, &payloadJSON, &e.CreatedAt); err != nil {
			return nil, err
		}
		if payloadJSON != nil {
			if err := json.Unmarshal(payloadJSON, &e.Payload); err != nil {
				return nil, fmt.Errorf("unmarshaling audit payload: %w", err)
			}
		}
		return &e, nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("scanning audit entries: %w", err)
	}

	return entries, total, nil
}

// nullIfEmpty returns nil if s is empty, otherwise returns a pointer to s.
// This maps empty string values to SQL NULL.
func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// nullableJSON returns nil if b is nil or empty, otherwise returns b.
// This avoids inserting empty JSON into the payload column.
func nullableJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}
