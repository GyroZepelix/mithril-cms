package content

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/GyroZepelix/mithril-cms/internal/database"
	"github.com/GyroZepelix/mithril-cms/internal/schema"
	"github.com/GyroZepelix/mithril-cms/internal/search"
)

// ErrNotFound is returned when a content entry does not exist.
var ErrNotFound = errors.New("content entry not found")

// Repository handles dynamic SQL generation and execution for content entries.
type Repository struct {
	db *database.DB
}

// NewRepository creates a new content Repository.
func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// allColumns returns the list of all columns to SELECT for a content type:
// id, status, user-defined fields (excluding many-relations), then system columns.
func allColumns(fields []schema.Field) []string {
	cols := []string{"id", "status"}
	for _, f := range fields {
		if f.Type == schema.FieldTypeRelation && f.RelationType == schema.RelationMany {
			continue
		}
		cols = append(cols, f.Name)
	}
	cols = append(cols, "created_by", "updated_by", "created_at", "updated_at", "published_at")
	return cols
}

// quotedColumns returns a comma-separated string of quoted column names.
func quotedColumns(cols []string) string {
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = schema.QuoteIdent(c)
	}
	return strings.Join(quoted, ", ")
}

// searchableFields returns the subset of fields marked as searchable.
func searchableFields(fields []schema.Field) []schema.Field {
	var result []schema.Field
	for _, f := range fields {
		if f.Searchable {
			result = append(result, f)
		}
	}
	return result
}

// List retrieves a paginated list of content entries with optional filtering and sorting.
func (r *Repository) List(ctx context.Context, tableName string, fields []schema.Field, q QueryParams, publishedOnly bool) ([]map[string]any, int, error) {
	cols := allColumns(fields)
	qTable := schema.QuoteIdent(tableName)

	// Build WHERE clause.
	var whereParts []string
	var args []any
	argIdx := 1

	if publishedOnly {
		whereParts = append(whereParts, fmt.Sprintf("%s = $%d", schema.QuoteIdent("status"), argIdx))
		args = append(args, "published")
		argIdx++
	}

	// Sort filter keys for deterministic parameter ordering.
	filterKeys := make([]string, 0, len(q.Filters))
	for field := range q.Filters {
		filterKeys = append(filterKeys, field)
	}
	sort.Strings(filterKeys)

	for _, field := range filterKeys {
		whereParts = append(whereParts, fmt.Sprintf("%s = $%d", schema.QuoteIdent(field), argIdx))
		args = append(args, q.Filters[field])
		argIdx++
	}

	// Full-text search integration.
	var searchWhere, searchOrder, searchHeadline string
	var searchArgs []any
	sFields := searchableFields(fields)
	if q.Search != "" && len(sFields) > 0 {
		searchWhere, searchOrder, searchHeadline, searchArgs = search.BuildSearchClause(
			q.Search, sFields, argIdx,
		)
		if searchWhere != "" {
			whereParts = append(whereParts, searchWhere)
			args = append(args, searchArgs...)
			argIdx += len(searchArgs)
		}
	}

	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = "WHERE " + strings.Join(whereParts, " AND ")
	}

	// Count query.
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", qTable, whereClause)
	var total int
	if err := r.db.Pool().QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting entries: %w", err)
	}

	// Build SELECT columns, including search headline when active.
	selectCols := quotedColumns(cols)
	if searchHeadline != "" {
		selectCols += ", " + searchHeadline
	}

	// Data query with ORDER BY, LIMIT, OFFSET.
	orderDir := "DESC"
	if strings.EqualFold(q.Order, "asc") {
		orderDir = "ASC"
	}

	// When search is active, rank first, then user's sort.
	var orderParts []string
	if searchOrder != "" {
		orderParts = append(orderParts, searchOrder)
	}
	orderParts = append(orderParts, fmt.Sprintf("%s %s", schema.QuoteIdent(q.Sort), orderDir))

	offset := (q.Page - 1) * q.PerPage

	dataSQL := fmt.Sprintf("SELECT %s FROM %s %s ORDER BY %s LIMIT $%d OFFSET $%d",
		selectCols,
		qTable,
		whereClause,
		strings.Join(orderParts, ", "),
		argIdx,
		argIdx+1,
	)
	args = append(args, q.PerPage, offset)

	rows, err := r.db.Pool().Query(ctx, dataSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying entries: %w", err)
	}
	defer rows.Close()

	entries, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, 0, fmt.Errorf("scanning entries: %w", err)
	}

	return entries, total, nil
}

// GetByID retrieves a single content entry by UUID.
func (r *Repository) GetByID(ctx context.Context, tableName string, fields []schema.Field, id string, publishedOnly bool) (map[string]any, error) {
	cols := allColumns(fields)
	qTable := schema.QuoteIdent(tableName)

	whereClause := fmt.Sprintf("WHERE %s = $1", schema.QuoteIdent("id"))
	args := []any{id}

	if publishedOnly {
		whereClause += fmt.Sprintf(" AND %s = $2", schema.QuoteIdent("status"))
		args = append(args, "published")
	}

	sql := fmt.Sprintf("SELECT %s FROM %s %s", quotedColumns(cols), qTable, whereClause)

	rows, err := r.db.Pool().Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("querying entry: %w", err)
	}
	defer rows.Close()

	entry, err := pgx.CollectOneRow(rows, pgx.RowToMap)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scanning entry: %w", err)
	}

	return entry, nil
}

// Insert creates a new content entry and returns the full row.
func (r *Repository) Insert(ctx context.Context, tableName string, fields []schema.Field, data map[string]any, adminID string) (map[string]any, error) {
	qTable := schema.QuoteIdent(tableName)

	// Build column and value lists from provided data (only schema fields).
	var colNames []string
	var placeholders []string
	var args []any
	argIdx := 1

	for _, f := range fields {
		if f.Type == schema.FieldTypeRelation && f.RelationType == schema.RelationMany {
			continue
		}
		val, ok := data[f.Name]
		if !ok {
			continue
		}
		colNames = append(colNames, schema.QuoteIdent(f.Name))
		placeholders = append(placeholders, fmt.Sprintf("$%d", argIdx))
		args = append(args, val)
		argIdx++
	}

	// Add created_by and updated_by.
	colNames = append(colNames, schema.QuoteIdent("created_by"), schema.QuoteIdent("updated_by"))
	placeholders = append(placeholders, fmt.Sprintf("$%d", argIdx), fmt.Sprintf("$%d", argIdx+1))
	args = append(args, adminID, adminID)

	returnCols := allColumns(fields)

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING %s",
		qTable,
		strings.Join(colNames, ", "),
		strings.Join(placeholders, ", "),
		quotedColumns(returnCols),
	)

	rows, err := r.db.Pool().Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("inserting entry: %w", err)
	}
	defer rows.Close()

	entry, err := pgx.CollectOneRow(rows, pgx.RowToMap)
	if err != nil {
		return nil, fmt.Errorf("scanning inserted entry: %w", err)
	}

	return entry, nil
}

// Update modifies an existing content entry and returns the full updated row.
func (r *Repository) Update(ctx context.Context, tableName string, fields []schema.Field, id string, data map[string]any, adminID string) (map[string]any, error) {
	qTable := schema.QuoteIdent(tableName)

	var setParts []string
	var args []any
	argIdx := 1

	for _, f := range fields {
		if f.Type == schema.FieldTypeRelation && f.RelationType == schema.RelationMany {
			continue
		}
		val, ok := data[f.Name]
		if !ok {
			continue
		}
		setParts = append(setParts, fmt.Sprintf("%s = $%d", schema.QuoteIdent(f.Name), argIdx))
		args = append(args, val)
		argIdx++
	}

	// Always update updated_by and updated_at (defense-in-depth alongside trigger).
	setParts = append(setParts, fmt.Sprintf("%s = $%d", schema.QuoteIdent("updated_by"), argIdx))
	args = append(args, adminID)
	argIdx++
	setParts = append(setParts, fmt.Sprintf("%s = now()", schema.QuoteIdent("updated_at")))

	// ID for WHERE clause.
	args = append(args, id)

	returnCols := allColumns(fields)

	sql := fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d RETURNING %s",
		qTable,
		strings.Join(setParts, ", "),
		schema.QuoteIdent("id"),
		argIdx,
		quotedColumns(returnCols),
	)

	rows, err := r.db.Pool().Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("updating entry: %w", err)
	}
	defer rows.Close()

	entry, err := pgx.CollectOneRow(rows, pgx.RowToMap)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scanning updated entry: %w", err)
	}

	return entry, nil
}

// Publish sets an entry's status to 'published' and published_at to now().
func (r *Repository) Publish(ctx context.Context, tableName string, fields []schema.Field, id, adminID string) (map[string]any, error) {
	qTable := schema.QuoteIdent(tableName)
	returnCols := allColumns(fields)

	sql := fmt.Sprintf("UPDATE %s SET %s = 'published', %s = now(), %s = $2, %s = now() WHERE %s = $1 RETURNING %s",
		qTable,
		schema.QuoteIdent("status"),
		schema.QuoteIdent("published_at"),
		schema.QuoteIdent("updated_by"),
		schema.QuoteIdent("updated_at"),
		schema.QuoteIdent("id"),
		quotedColumns(returnCols),
	)

	rows, err := r.db.Pool().Query(ctx, sql, id, adminID)
	if err != nil {
		return nil, fmt.Errorf("publishing entry: %w", err)
	}
	defer rows.Close()

	entry, err := pgx.CollectOneRow(rows, pgx.RowToMap)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scanning published entry: %w", err)
	}

	return entry, nil
}
