// Package schemaapi provides HTTP handlers for schema management operations.
// It is separated from the schema package to avoid import cycles, since the
// handler depends on the server, auth, and audit packages.
package schemaapi

import (
	"log/slog"
	"net/http"
	"sync"

	"github.com/GyroZepelix/mithril-cms/internal/audit"
	"github.com/GyroZepelix/mithril-cms/internal/auth"
	"github.com/GyroZepelix/mithril-cms/internal/schema"
	"github.com/GyroZepelix/mithril-cms/internal/server"
)

// Handler provides HTTP handlers for schema management operations.
type Handler struct {
	engine    *schema.Engine
	schemaDir string
	audit     *audit.Service

	// mu protects schemaMap during concurrent refreshes. Schema refreshes are
	// rare admin operations, but we still guard against concurrent access.
	mu        sync.RWMutex
	schemaMap map[string]schema.ContentType

	// onRefresh is called after a successful refresh with the new schemas.
	// This allows the server to update other components that hold schema maps
	// (e.g., content handler, content service).
	onRefresh func(schemas []schema.ContentType)
}

// NewHandler creates a new schema Handler.
// The audit service is optional; if nil, audit events are silently skipped.
// The onRefresh callback is optional; if nil, only the handler's own schema
// map is updated on refresh.
func NewHandler(engine *schema.Engine, schemaDir string, schemaMap map[string]schema.ContentType, auditSvc *audit.Service, onRefresh func([]schema.ContentType)) *Handler {
	return &Handler{
		engine:    engine,
		schemaDir: schemaDir,
		audit:     auditSvc,
		schemaMap: schemaMap,
		onRefresh: onRefresh,
	}
}

// SchemaMap returns the current schema map. This is safe for concurrent reads
// because the map reference is only replaced (not mutated) during refresh.
func (h *Handler) SchemaMap() map[string]schema.ContentType {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.schemaMap
}

// Refresh handles POST /admin/api/schema/refresh. It reloads schemas from
// disk, diffs against the database, and applies changes. Breaking changes
// block the entire refresh and result in a 409 Conflict response.
//
// Response on success (200):
//
//	{"data": {"applied": [...], "new_types": [...], "updated_types": [...]}}
//
// Response on breaking changes (409):
//
//	{"error": {"code": "BREAKING_CHANGES", "message": "...", "details": [...]}}
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	result, schemas, err := h.engine.Refresh(r.Context(), h.schemaDir, false)
	if err != nil {
		slog.Error("schema refresh failed", "error", err)
		server.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR",
			"schema refresh failed: "+err.Error(), nil)
		return
	}

	// If there are breaking changes, the refresh was blocked. Return 409 with
	// details. Do NOT update the in-memory schema map because the database
	// state was not modified.
	if len(result.Breaking) > 0 {
		details := make([]server.FieldError, 0, len(result.Breaking))
		for _, c := range result.Breaking {
			details = append(details, server.FieldError{
				Field:   c.Table + "." + c.Column,
				Message: c.Detail,
			})
		}

		server.Error(w, http.StatusConflict, "BREAKING_CHANGES",
			"schema refresh blocked due to breaking changes", details)
		return
	}

	// Success: schemas were applied (or there were no changes).
	// Update the handler's schema map and notify other components.
	if schemas != nil {
		newMap := make(map[string]schema.ContentType, len(schemas))
		for _, ct := range schemas {
			newMap[ct.Name] = ct
		}

		h.mu.Lock()
		h.schemaMap = newMap
		h.mu.Unlock()

		// Notify other components (e.g., content handler/service).
		if h.onRefresh != nil {
			h.onRefresh(schemas)
		}
	}

	// Log audit event.
	if h.audit != nil {
		adminID := auth.AdminIDFromContext(r.Context())
		h.audit.Log(r.Context(), audit.Event{
			Action:  "schema.refresh",
			ActorID: adminID,
			Payload: map[string]any{
				"applied_count": len(result.Applied),
				"new_types":     result.NewTypes,
				"updated_types": result.UpdatedTypes,
			},
		})
	}

	// Build response.
	type changeResponse struct {
		Type   string `json:"type"`
		Table  string `json:"table"`
		Column string `json:"column,omitempty"`
		Detail string `json:"detail"`
		Safe   bool   `json:"safe"`
	}

	appliedResp := make([]changeResponse, 0, len(result.Applied))
	for _, c := range result.Applied {
		appliedResp = append(appliedResp, changeResponse{
			Type:   string(c.Type),
			Table:  c.Table,
			Column: c.Column,
			Detail: c.Detail,
			Safe:   c.Safe,
		})
	}

	server.JSON(w, http.StatusOK, map[string]any{
		"applied":       appliedResp,
		"new_types":     result.NewTypes,
		"updated_types": result.UpdatedTypes,
	})
}
