package content

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/GyroZepelix/mithril-cms/internal/auth"
	"github.com/GyroZepelix/mithril-cms/internal/schema"
	"github.com/GyroZepelix/mithril-cms/internal/server"
)

// maxBodySize is the maximum allowed request body size (1 MiB).
const maxBodySize = 1 << 20

// Handler provides HTTP handlers for content CRUD operations.
type Handler struct {
	service *Service
	schemas map[string]schema.ContentType
}

// NewHandler creates a new content Handler.
func NewHandler(service *Service, schemas map[string]schema.ContentType) *Handler {
	return &Handler{
		service: service,
		schemas: schemas,
	}
}

// lookupSchema validates that the content type exists and returns it.
// Returns false if the content type was not found (404 already written).
func (h *Handler) lookupSchema(w http.ResponseWriter, r *http.Request) (schema.ContentType, bool) {
	name := chi.URLParam(r, "contentType")
	ct, ok := h.schemas[name]
	if !ok {
		server.Error(w, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("content type '%s' not found", name), nil)
		return schema.ContentType{}, false
	}
	return ct, true
}

// decodeBody reads and decodes a JSON request body into a map.
func decodeBody(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	var data map[string]any
	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	if err := dec.Decode(&data); err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_JSON",
			"invalid or too-large JSON body", nil)
		return nil, false
	}

	// Convert json.Number values to native Go types for downstream processing.
	convertNumbers(data)

	return data, true
}

// convertNumbers walks a map and converts json.Number values to int64 or float64.
func convertNumbers(data map[string]any) {
	for key, val := range data {
		switch v := val.(type) {
		case json.Number:
			// Try int first (keep as int64), then float.
			if i, err := v.Int64(); err == nil {
				data[key] = i
			} else if f, err := v.Float64(); err == nil {
				data[key] = f
			}
		case map[string]any:
			convertNumbers(v)
		}
	}
}

// handleServiceError writes the appropriate error response for service errors.
func handleServiceError(w http.ResponseWriter, err error) {
	var valErr *ValidationError
	if errors.As(err, &valErr) {
		server.Error(w, http.StatusBadRequest, "VALIDATION_ERROR",
			"Validation failed", valErr.Fields)
		return
	}
	if errors.Is(err, ErrNotFound) {
		server.Error(w, http.StatusNotFound, "NOT_FOUND", "entry not found", nil)
		return
	}
	slog.Error("content service error", "error", err)
	server.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR",
		"an internal error occurred", nil)
}

// --- Admin handlers ---

// AdminList handles GET /admin/api/content/{contentType}.
func (h *Handler) AdminList(w http.ResponseWriter, r *http.Request) {
	ct, ok := h.lookupSchema(w, r)
	if !ok {
		return
	}

	q, err := ParseQueryParams(r, ct)
	if err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_PARAMS", err.Error(), nil)
		return
	}

	entries, total, err := h.service.List(r.Context(), ct.Name, q, false)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	totalPages := 0
	if q.PerPage > 0 {
		totalPages = (total + q.PerPage - 1) / q.PerPage
	}

	server.Paginated(w, entries, server.PaginationMeta{
		Page:       q.Page,
		PerPage:    q.PerPage,
		Total:      total,
		TotalPages: totalPages,
	})
}

// AdminGet handles GET /admin/api/content/{contentType}/{id}.
func (h *Handler) AdminGet(w http.ResponseWriter, r *http.Request) {
	ct, ok := h.lookupSchema(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	if !isValidUUID(id) {
		server.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be a valid UUID", nil)
		return
	}
	entry, err := h.service.GetByID(r.Context(), ct.Name, id, false)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, entry)
}

// AdminCreate handles POST /admin/api/content/{contentType}.
func (h *Handler) AdminCreate(w http.ResponseWriter, r *http.Request) {
	ct, ok := h.lookupSchema(w, r)
	if !ok {
		return
	}

	data, ok := decodeBody(w, r)
	if !ok {
		return
	}

	adminID := auth.AdminIDFromContext(r.Context())
	entry, err := h.service.Create(r.Context(), ct.Name, data, adminID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	server.JSON(w, http.StatusCreated, entry)
}

// AdminUpdate handles PUT /admin/api/content/{contentType}/{id}.
func (h *Handler) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	ct, ok := h.lookupSchema(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	if !isValidUUID(id) {
		server.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be a valid UUID", nil)
		return
	}
	data, ok := decodeBody(w, r)
	if !ok {
		return
	}

	adminID := auth.AdminIDFromContext(r.Context())
	entry, err := h.service.Update(r.Context(), ct.Name, id, data, adminID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, entry)
}

// AdminPublish handles POST /admin/api/content/{contentType}/{id}/publish.
func (h *Handler) AdminPublish(w http.ResponseWriter, r *http.Request) {
	ct, ok := h.lookupSchema(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	if !isValidUUID(id) {
		server.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be a valid UUID", nil)
		return
	}
	adminID := auth.AdminIDFromContext(r.Context())
	entry, err := h.service.Publish(r.Context(), ct.Name, id, adminID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, entry)
}

// --- Public handlers ---

// PublicList handles GET /api/{contentType}.
func (h *Handler) PublicList(w http.ResponseWriter, r *http.Request) {
	ct, ok := h.lookupSchema(w, r)
	if !ok {
		return
	}

	if !ct.PublicRead {
		server.Error(w, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("content type '%s' not found", ct.Name), nil)
		return
	}

	q, err := ParseQueryParams(r, ct)
	if err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_PARAMS", err.Error(), nil)
		return
	}

	entries, total, err := h.service.List(r.Context(), ct.Name, q, true)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	totalPages := 0
	if q.PerPage > 0 {
		totalPages = (total + q.PerPage - 1) / q.PerPage
	}

	server.Paginated(w, entries, server.PaginationMeta{
		Page:       q.Page,
		PerPage:    q.PerPage,
		Total:      total,
		TotalPages: totalPages,
	})
}

// PublicGet handles GET /api/{contentType}/{id}.
func (h *Handler) PublicGet(w http.ResponseWriter, r *http.Request) {
	ct, ok := h.lookupSchema(w, r)
	if !ok {
		return
	}

	if !ct.PublicRead {
		server.Error(w, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("content type '%s' not found", ct.Name), nil)
		return
	}

	id := chi.URLParam(r, "id")
	if !isValidUUID(id) {
		server.Error(w, http.StatusBadRequest, "INVALID_ID", "id must be a valid UUID", nil)
		return
	}
	entry, err := h.service.GetByID(r.Context(), ct.Name, id, true)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	server.JSON(w, http.StatusOK, entry)
}
