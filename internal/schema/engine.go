package schema

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/GyroZepelix/mithril-cms/internal/database"
)

// Engine orchestrates schema diffing and application against the database.
// It loads existing content types from the content_types table, compares them
// to the YAML-defined schemas, and applies safe changes (or all changes in dev
// mode).
type Engine struct {
	db      *database.DB
	devMode bool
}

// NewEngine creates a new schema engine.
func NewEngine(db *database.DB, devMode bool) *Engine {
	return &Engine{
		db:      db,
		devMode: devMode,
	}
}

// existingContentType holds a content type record as stored in the content_types table.
type existingContentType struct {
	Name        string
	DisplayName string
	SchemaHash  string
	Fields      []Field
	PublicRead  bool
}

// Apply compares the given schemas against the database state and applies
// changes. The process is:
//  1. Query all existing content types from the content_types table.
//  2. For each loaded schema, diff against existing (by name + schema_hash).
//  3. If schema_hash matches, skip (no changes).
//  4. Collect all changes and separate safe vs breaking.
//  5. If any breaking changes and NOT dev mode, return an error listing them.
//  6. Execute all DDL changes AND upsert content_types rows in a single transaction.
func (e *Engine) Apply(ctx context.Context, schemas []ContentType) error {
	existing, err := e.loadExisting(ctx)
	if err != nil {
		return fmt.Errorf("loading existing content types: %w", err)
	}

	existingMap := make(map[string]existingContentType, len(existing))
	for _, ct := range existing {
		existingMap[ct.Name] = ct
	}

	var allChanges []Change
	var changedSchemas []ContentType // schemas that need content_types upsert

	for _, loaded := range schemas {
		ex, found := existingMap[loaded.Name]

		// If the schema hash matches, the schema has not changed.
		if found && ex.SchemaHash == loaded.SchemaHash {
			slog.Debug("schema unchanged, skipping", "content_type", loaded.Name)
			continue
		}

		var existingCT *ContentType
		if found {
			// Convert DB record to ContentType for diffing.
			ct := ContentType{
				Name:        ex.Name,
				DisplayName: ex.DisplayName,
				Fields:      ex.Fields,
				PublicRead:  ex.PublicRead,
				SchemaHash:  ex.SchemaHash,
			}
			existingCT = &ct
		}

		changes := DiffSchema(loaded, existingCT)
		if len(changes) > 0 {
			allChanges = append(allChanges, changes...)
			changedSchemas = append(changedSchemas, loaded)
		} else {
			// Hash changed but no structural diff (e.g., whitespace or comment change).
			// Still update the hash in content_types.
			changedSchemas = append(changedSchemas, loaded)
		}
	}

	if len(allChanges) == 0 && len(changedSchemas) == 0 {
		slog.Info("all schemas up to date, no changes to apply")
		return nil
	}

	// Separate safe vs breaking.
	var safeChanges, breakingChanges []Change
	for _, c := range allChanges {
		if c.Safe {
			safeChanges = append(safeChanges, c)
		} else {
			breakingChanges = append(breakingChanges, c)
		}
	}

	// Block breaking changes in non-dev mode.
	if len(breakingChanges) > 0 && !e.devMode {
		return &BreakingChangesError{Changes: breakingChanges}
	}

	// Apply all DDL changes and upsert content_types in a single transaction.
	if err := e.applyInTransaction(ctx, allChanges, changedSchemas); err != nil {
		return fmt.Errorf("applying schema changes: %w", err)
	}

	slog.Info("schema changes applied",
		"safe", len(safeChanges),
		"breaking", len(breakingChanges),
		"content_types_updated", len(changedSchemas),
	)

	return nil
}

// loadExisting queries all existing content types from the content_types table.
func (e *Engine) loadExisting(ctx context.Context) ([]existingContentType, error) {
	rows, err := e.db.Pool().Query(ctx,
		`SELECT name, display_name, schema_hash, fields, public_read FROM content_types`)
	if err != nil {
		return nil, fmt.Errorf("querying content_types: %w", err)
	}
	defer rows.Close()

	var result []existingContentType
	for rows.Next() {
		var ct existingContentType
		var fieldsJSON []byte

		if err := rows.Scan(&ct.Name, &ct.DisplayName, &ct.SchemaHash, &fieldsJSON, &ct.PublicRead); err != nil {
			return nil, fmt.Errorf("scanning content_type row: %w", err)
		}

		if err := json.Unmarshal(fieldsJSON, &ct.Fields); err != nil {
			return nil, fmt.Errorf("unmarshaling fields for %q: %w", ct.Name, err)
		}

		result = append(result, ct)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating content_type rows: %w", err)
	}

	return result, nil
}

// applyInTransaction executes all DDL change SQL statements and upserts
// content_types rows in a single transaction. This ensures atomicity: either
// all DDL changes and metadata updates succeed together, or none do.
func (e *Engine) applyInTransaction(ctx context.Context, changes []Change, schemas []ContentType) error {
	tx, err := e.db.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() {
		// Rollback is a no-op if the tx has been committed.
		_ = tx.Rollback(ctx)
	}()

	// Execute DDL changes.
	for _, c := range changes {
		if c.SQL == "" {
			slog.Warn("skipping change with empty SQL", "type", c.Type, "table", c.Table, "column", c.Column)
			continue
		}

		slog.Info("applying schema change", "type", c.Type, "detail", c.Detail)

		if _, err := tx.Exec(ctx, c.SQL); err != nil {
			return fmt.Errorf("executing %s on %s.%s: %w", c.Type, c.Table, c.Column, err)
		}
	}

	// Upsert content_types rows in the same transaction.
	for _, ct := range schemas {
		fieldsJSON, err := json.Marshal(ct.Fields)
		if err != nil {
			return fmt.Errorf("marshaling fields for %q: %w", ct.Name, err)
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO content_types (name, display_name, schema_hash, fields, public_read)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (name) DO UPDATE SET
			   display_name = EXCLUDED.display_name,
			   schema_hash = EXCLUDED.schema_hash,
			   fields = EXCLUDED.fields,
			   public_read = EXCLUDED.public_read,
			   updated_at = now()`,
			ct.Name, ct.DisplayName, ct.SchemaHash, fieldsJSON, ct.PublicRead,
		)
		if err != nil {
			return fmt.Errorf("upserting content type %q: %w", ct.Name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// GetExistingContentType returns a single existing content type by name, or
// nil if it does not exist. This is useful for targeted diffing.
func (e *Engine) GetExistingContentType(ctx context.Context, name string) (*ContentType, error) {
	var displayName, schemaHash string
	var fieldsJSON []byte
	var publicRead bool

	err := e.db.Pool().QueryRow(ctx,
		`SELECT display_name, schema_hash, fields, public_read FROM content_types WHERE name = $1`,
		name,
	).Scan(&displayName, &schemaHash, &fieldsJSON, &publicRead)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying content type %q: %w", name, err)
	}

	var fields []Field
	if err := json.Unmarshal(fieldsJSON, &fields); err != nil {
		return nil, fmt.Errorf("unmarshaling fields for %q: %w", name, err)
	}

	return &ContentType{
		Name:        name,
		DisplayName: displayName,
		Fields:      fields,
		PublicRead:  publicRead,
		SchemaHash:  schemaHash,
	}, nil
}

// RefreshResult holds the outcome of a schema Refresh operation, including
// which changes were applied, which were blocked as breaking, and which
// content types were created or updated.
type RefreshResult struct {
	// Applied is the list of changes that were successfully applied.
	Applied []Change

	// Breaking is the list of breaking changes that were NOT applied
	// (unless force was true, in which case they appear in Applied instead).
	Breaking []Change

	// NewTypes lists the names of content types that were newly created.
	NewTypes []string

	// UpdatedTypes lists the names of existing content types that were modified.
	UpdatedTypes []string
}

// Refresh reloads schemas from the given directory, validates them, diffs
// against the current database state, and applies changes based on force mode.
// Breaking changes block the entire refresh unless force is true.
//
// Returns the refresh result, the newly loaded schemas (nil if breaking changes
// blocked the refresh), and any error.
func (e *Engine) Refresh(ctx context.Context, schemaDir string, force bool) (*RefreshResult, []ContentType, error) {
	// Step 1: Load schemas from disk.
	schemas, err := LoadSchemas(schemaDir)
	if err != nil {
		return nil, nil, fmt.Errorf("loading schemas: %w", err)
	}

	// Step 2: Validate schemas.
	if err := ValidateSchemas(schemas); err != nil {
		return nil, nil, fmt.Errorf("validating schemas: %w", err)
	}

	// Step 3: Load existing content types from DB.
	existing, err := e.loadExisting(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("loading existing content types: %w", err)
	}

	existingMap := make(map[string]existingContentType, len(existing))
	for _, ct := range existing {
		existingMap[ct.Name] = ct
	}

	var allChanges []Change
	var changedSchemas []ContentType

	// Track which content types are new vs updated.
	newTypeSet := make(map[string]bool)
	updatedTypeSet := make(map[string]bool)

	for _, loaded := range schemas {
		ex, found := existingMap[loaded.Name]

		if found && ex.SchemaHash == loaded.SchemaHash {
			slog.Debug("schema unchanged, skipping", "content_type", loaded.Name)
			continue
		}

		var existingCT *ContentType
		if found {
			ct := ContentType{
				Name:        ex.Name,
				DisplayName: ex.DisplayName,
				Fields:      ex.Fields,
				PublicRead:  ex.PublicRead,
				SchemaHash:  ex.SchemaHash,
			}
			existingCT = &ct
		}

		changes := DiffSchema(loaded, existingCT)
		if len(changes) > 0 {
			allChanges = append(allChanges, changes...)
			changedSchemas = append(changedSchemas, loaded)
		} else {
			// Hash changed but no structural diff; still update in content_types.
			changedSchemas = append(changedSchemas, loaded)
		}

		if found {
			updatedTypeSet[loaded.Name] = true
		} else {
			newTypeSet[loaded.Name] = true
		}
	}

	// Step 4: Separate safe vs breaking.
	var safeChanges, breakingChanges []Change
	for _, c := range allChanges {
		if c.Safe {
			safeChanges = append(safeChanges, c)
		} else {
			breakingChanges = append(breakingChanges, c)
		}
	}

	result := &RefreshResult{}

	// Step 5: Determine which changes to apply.
	// Simplified logic: if any breaking changes exist and force=false,
	// don't apply ANYTHING to avoid partial state inconsistency.
	if len(breakingChanges) > 0 && !force {
		// Block the entire refresh. Return the breaking changes but don't
		// update the database or return updated schemas.
		result.Breaking = breakingChanges
		result.Applied = nil
		slog.Warn("schema refresh blocked due to breaking changes",
			"breaking", len(breakingChanges))
		return result, nil, nil
	}

	// No breaking changes, or force=true: apply all changes.
	if len(allChanges) > 0 || len(changedSchemas) > 0 {
		if err := e.applyInTransaction(ctx, allChanges, changedSchemas); err != nil {
			return nil, nil, fmt.Errorf("applying schema changes: %w", err)
		}
		result.Applied = allChanges
	}

	// Build new/updated type lists.
	for name := range newTypeSet {
		result.NewTypes = append(result.NewTypes, name)
	}
	for name := range updatedTypeSet {
		result.UpdatedTypes = append(result.UpdatedTypes, name)
	}

	slog.Info("schema refresh completed",
		"applied", len(result.Applied),
		"breaking", len(result.Breaking),
		"new_types", len(result.NewTypes),
		"updated_types", len(result.UpdatedTypes),
	)

	return result, schemas, nil
}


// BreakingChangesError is returned when Apply detects breaking schema changes
// and the engine is not in dev mode.
type BreakingChangesError struct {
	Changes []Change
}

// Error returns a human-readable summary of all breaking changes.
func (e *BreakingChangesError) Error() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("schema migration blocked: %d breaking change(s) detected (use dev mode to force):\n", len(e.Changes)))
	for _, c := range e.Changes {
		b.WriteString(fmt.Sprintf("  - %s\n", c.Detail))
	}
	return b.String()
}
