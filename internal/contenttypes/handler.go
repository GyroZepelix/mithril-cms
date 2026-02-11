// Package contenttypes provides HTTP handlers for content type introspection,
// allowing the admin UI to discover available content types with their field
// definitions and entry counts.
package contenttypes

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/GyroZepelix/mithril-cms/internal/schema"
	"github.com/GyroZepelix/mithril-cms/internal/server"
)

// FieldResponse represents a single field in the content type introspection response.
type FieldResponse struct {
	Name         string              `json:"name"`
	Type         schema.FieldType    `json:"type"`
	Required     bool                `json:"required"`
	Unique       bool                `json:"unique"`
	Searchable   bool                `json:"searchable"`
	MinLength    *int                `json:"min_length,omitempty"`
	MaxLength    *int                `json:"max_length,omitempty"`
	Min          *float64            `json:"min,omitempty"`
	Max          *float64            `json:"max,omitempty"`
	Regex        string              `json:"regex,omitempty"`
	Values       []string            `json:"values,omitempty"`
	RelatesTo    string              `json:"relates_to,omitempty"`
	RelationType schema.RelationType `json:"relation_type,omitempty"`
}

// ContentTypeResponse represents a content type in the introspection API response.
type ContentTypeResponse struct {
	Name        string          `json:"name"`
	DisplayName string          `json:"display_name"`
	PublicRead  bool            `json:"public_read"`
	Fields      []FieldResponse `json:"fields"`
	EntryCount  int             `json:"entry_count"`
}

// Handler provides HTTP handlers for content type introspection.
type Handler struct {
	pool *pgxpool.Pool
	mu   sync.RWMutex
	schemas map[string]schema.ContentType
}

// NewHandler creates a new content types Handler.
// The schemas map is copied defensively to avoid external mutation.
func NewHandler(pool *pgxpool.Pool, schemas map[string]schema.ContentType) *Handler {
	// Copy the schemas map for defensive programming
	schemasCopy := make(map[string]schema.ContentType, len(schemas))
	for k, v := range schemas {
		schemasCopy[k] = v
	}
	return &Handler{
		pool:    pool,
		schemas: schemasCopy,
	}
}

// UpdateSchemas replaces the in-memory schema map. This is called after a
// schema refresh to ensure the handler uses the latest content type definitions.
func (h *Handler) UpdateSchemas(schemas map[string]schema.ContentType) {
	h.mu.Lock()
	h.schemas = schemas
	h.mu.Unlock()
}

// getSchemas returns a snapshot of all schemas, sorted by name for deterministic output.
func (h *Handler) getSchemas() []schema.ContentType {
	h.mu.RLock()
	defer h.mu.RUnlock()

	types := make([]schema.ContentType, 0, len(h.schemas))
	for _, ct := range h.schemas {
		types = append(types, ct)
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i].Name < types[j].Name
	})
	return types
}

// getSchema safely retrieves a schema by name with read locking.
func (h *Handler) getSchema(name string) (schema.ContentType, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ct, ok := h.schemas[name]
	return ct, ok
}

// List handles GET /admin/api/content-types.
// Returns all content types with their field definitions and entry counts.
// This endpoint is not paginated because the number of content types is typically
// small (< 100) and the payload is lightweight.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	types := h.getSchemas()

	responses := make([]ContentTypeResponse, 0, len(types))
	for _, ct := range types {
		count, err := h.countEntries(r.Context(), ct.Name)
		if err != nil {
			// Log the error but continue with count=0 for graceful degradation.
			// A missing count shouldn't block the entire content type list.
			slog.Error("failed to count entries", "content_type", ct.Name, "error", err)
			count = 0
		}
		responses = append(responses, buildResponse(ct, count))
	}

	server.JSON(w, http.StatusOK, responses)
}

// Get handles GET /admin/api/content-types/{name}.
// Returns a single content type with its full field definitions and entry count.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	ct, ok := h.getSchema(name)
	if !ok {
		server.Error(w, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("content type '%s' not found", name), nil)
		return
	}

	count, err := h.countEntries(r.Context(), ct.Name)
	if err != nil {
		slog.Error("failed to count entries", "content_type", ct.Name, "error", err)
		server.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR",
			"failed to retrieve entry count", nil)
		return
	}

	server.JSON(w, http.StatusOK, buildResponse(ct, count))
}

// countEntries queries the database for the total number of entries in a content
// type's table. The table name is constructed from the content type name using
// the ct_{name} convention.
func (h *Handler) countEntries(ctx context.Context, name string) (int, error) {
	tableName := "ct_" + name
	query := fmt.Sprintf("SELECT count(*) FROM %s", schema.QuoteIdent(tableName))

	var count int
	if err := h.pool.QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("counting entries for %s: %w", name, err)
	}
	return count, nil
}

// buildResponse converts a schema.ContentType and entry count into the API response type.
func buildResponse(ct schema.ContentType, entryCount int) ContentTypeResponse {
	fields := make([]FieldResponse, len(ct.Fields))
	for i, f := range ct.Fields {
		fields[i] = FieldResponse{
			Name:         f.Name,
			Type:         f.Type,
			Required:     f.Required,
			Unique:       f.Unique,
			Searchable:   f.Searchable,
			MinLength:    f.MinLength,
			MaxLength:    f.MaxLength,
			Min:          f.Min,
			Max:          f.Max,
			Regex:        f.Regex,
			Values:       f.Values,
			RelatesTo:    f.RelatesTo,
			RelationType: f.RelationType,
		}
	}

	return ContentTypeResponse{
		Name:        ct.Name,
		DisplayName: ct.DisplayName,
		PublicRead:  ct.PublicRead,
		Fields:      fields,
		EntryCount:  entryCount,
	}
}
