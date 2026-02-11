package audit

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/GyroZepelix/mithril-cms/internal/server"
)

// Handler provides HTTP handlers for the audit log API.
type Handler struct {
	service *Service
}

// NewHandler creates a new audit Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List handles GET /admin/api/audit-log. It returns a paginated list of audit
// entries, optionally filtered by action and/or resource.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filters := AuditFilters{
		Action:   q.Get("action"),
		Resource: q.Get("resource"),
	}

	page, perPage := parsePagination(r)

	entries, total, err := h.service.List(r.Context(), filters, page, perPage)
	if err != nil {
		slog.Error("audit log list failed", "error", err)
		server.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR",
			"an internal error occurred", nil)
		return
	}

	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}

	server.Paginated(w, entries, server.PaginationMeta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	})
}

// parsePagination extracts page and per_page query parameters with defaults.
func parsePagination(r *http.Request) (page, perPage int) {
	page = 1
	perPage = 20

	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			perPage = n
			if perPage > 100 {
				perPage = 100
			}
		}
	}
	return page, perPage
}
