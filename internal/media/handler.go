package media

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/GyroZepelix/mithril-cms/internal/auth"
	"github.com/GyroZepelix/mithril-cms/internal/server"
)

// maxFormSize is the maximum size for ParseMultipartForm (10 MiB + 1 MiB overhead).
const maxFormSize = 11 << 20

// Handler provides HTTP handlers for media operations.
type Handler struct {
	service *Service
	devMode bool
}

// NewHandler creates a new media Handler.
func NewHandler(service *Service, devMode bool) *Handler {
	return &Handler{
		service: service,
		devMode: devMode,
	}
}

// Upload handles POST /admin/api/media.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	// Limit the overall request body.
	r.Body = http.MaxBytesReader(w, r.Body, maxFormSize)

	if err := r.ParseMultipartForm(maxFormSize); err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_UPLOAD",
			"failed to parse multipart form: file may be too large", nil)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		server.Error(w, http.StatusBadRequest, "MISSING_FILE",
			"missing 'file' field in multipart form", nil)
		return
	}
	defer file.Close()

	adminID := auth.AdminIDFromContext(r.Context())

	m, err := h.service.Upload(r.Context(), header, adminID)
	if err != nil {
		var ue *UploadError
		if errors.As(err, &ue) {
			server.Error(w, http.StatusBadRequest, "UPLOAD_ERROR", ue.Message, nil)
			return
		}
		slog.Error("media upload failed", "error", err)
		server.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR",
			"an internal error occurred", nil)
		return
	}

	server.JSON(w, http.StatusCreated, m)
}

// List handles GET /admin/api/media.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, perPage := parsePagination(r)

	items, total, err := h.service.List(r.Context(), page, perPage)
	if err != nil {
		slog.Error("media list failed", "error", err)
		server.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR",
			"an internal error occurred", nil)
		return
	}

	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}

	server.Paginated(w, items, server.PaginationMeta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	})
}

// Delete handles DELETE /admin/api/media/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isValidUUID(id) {
		server.Error(w, http.StatusBadRequest, "INVALID_ID",
			"id must be a valid UUID", nil)
		return
	}

	err := h.service.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			server.Error(w, http.StatusNotFound, "NOT_FOUND", "media not found", nil)
			return
		}
		slog.Error("media delete failed", "error", err)
		server.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR",
			"an internal error occurred", nil)
		return
	}

	server.JSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// Serve handles GET /media/{filename}.
func (h *Handler) Serve(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if filename == "" {
		server.Error(w, http.StatusBadRequest, "MISSING_FILENAME",
			"filename is required", nil)
		return
	}

	// Look up the media record to verify it exists and get the MIME type.
	m, err := h.service.GetByFilename(r.Context(), filename)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			server.Error(w, http.StatusNotFound, "NOT_FOUND", "media not found", nil)
			return
		}
		slog.Error("media lookup failed", "error", err)
		server.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR",
			"an internal error occurred", nil)
		return
	}

	// Determine which variant to serve.
	variant := r.URL.Query().Get("v")
	if variant == "" {
		variant = "original"
	}
	if !isValidVariant(variant) {
		server.Error(w, http.StatusBadRequest, "INVALID_VARIANT",
			"variant must be one of: original, sm, md, lg", nil)
		return
	}

	// For non-original variants, check that the variant exists.
	serveFilename := filename
	if variant != "original" {
		variantPath, ok := m.Variants[variant]
		if !ok {
			// Fall back to original if the requested variant was not generated.
			variant = "original"
		} else {
			// Extract variant filename from the stored path (variant/filename).
			parts := strings.SplitN(variantPath, "/", 2)
			if len(parts) == 2 {
				serveFilename = parts[1]
			}
		}
	}

	filePath := h.service.storage.Path(variant, serveFilename)
	if filePath == "" {
		server.Error(w, http.StatusBadRequest, "INVALID_FILENAME",
			"invalid filename", nil)
		return
	}

	// Verify the file exists on disk.
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			server.Error(w, http.StatusNotFound, "NOT_FOUND", "media file not found on disk", nil)
			return
		}
		slog.Error("media file stat failed", "error", err)
		server.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR",
			"an internal error occurred", nil)
		return
	}

	// Set security headers.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")

	// For non-image files, force download via Content-Disposition to prevent
	// the browser from rendering potentially dangerous content inline.
	if !imageMIMETypes[m.MimeType] {
		w.Header().Set("Content-Disposition",
			fmt.Sprintf(`attachment; filename="%s"`, sanitizeFilename(m.OriginalName)))
	}

	// Set headers for caching and content type.
	w.Header().Set("Content-Type", m.MimeType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

	http.ServeFile(w, r, filePath)
}

// sanitizeFilename removes characters that are problematic in Content-Disposition
// headers (double quotes and backslashes).
func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, `"`, "")
	name = strings.ReplaceAll(name, `\`, "")
	if name == "" {
		name = "download"
	}
	return name
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
