package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/GyroZepelix/mithril-cms/internal/audit"
	"github.com/GyroZepelix/mithril-cms/internal/server"
)

const (
	refreshCookieName = "refresh_token"
	refreshCookiePath = "/admin/api/auth"
	refreshCookieAge  = 7 * 24 * 60 * 60 // 7 days in seconds

	// maxRequestBodySize is the maximum allowed size for JSON request bodies
	// (1 MB). This prevents clients from sending excessively large payloads.
	maxRequestBodySize = 1 << 20
)

// Handler provides HTTP handlers for authentication endpoints.
type Handler struct {
	service      *Service
	auditService *audit.Service
	devMode      bool
}

// NewHandler creates a new auth Handler with the given service. The devMode
// flag controls whether the refresh token cookie is set with the Secure flag
// (disabled in dev mode to allow HTTP on localhost). The audit service is
// optional; if nil, audit events are silently skipped.
func NewHandler(service *Service, auditService *audit.Service, devMode bool) *Handler {
	return &Handler{
		service:      service,
		auditService: auditService,
		devMode:      devMode,
	}
}

// loginRequest is the expected JSON body for POST /admin/api/auth/login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login handles POST /admin/api/auth/login. It validates the credentials,
// returns an access token in the JSON response body, and sets the refresh
// token as an httpOnly cookie.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		server.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", nil)
		return
	}

	if req.Email == "" || req.Password == "" {
		server.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "email and password are required", nil)
		return
	}

	adminID, accessToken, refreshToken, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			h.logAudit(r.Context(), audit.Event{
				Action:  "admin.login.failure",
				Payload: map[string]any{"email": req.Email},
			})
			server.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password", nil)
			return
		}
		slog.Error("login failed", "error", err)
		server.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred", nil)
		return
	}

	h.logAudit(r.Context(), audit.Event{
		Action:  "admin.login.success",
		ActorID: adminID,
	})

	h.setRefreshCookie(w, refreshToken)
	server.JSON(w, http.StatusOK, map[string]string{
		"access_token": accessToken,
	})
}

// Refresh handles POST /admin/api/auth/refresh. It reads the refresh token
// from the httpOnly cookie, rotates it, and returns a new access token with
// a new refresh cookie.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		server.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing refresh token cookie", nil)
		return
	}

	if cookie.Value == "" {
		server.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "empty refresh token", nil)
		return
	}

	accessToken, newRefreshToken, err := h.service.Refresh(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			h.clearRefreshCookie(w)
			server.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired refresh token", nil)
			return
		}
		slog.Error("token refresh failed", "error", err)
		server.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred", nil)
		return
	}

	h.setRefreshCookie(w, newRefreshToken)
	server.JSON(w, http.StatusOK, map[string]string{
		"access_token": accessToken,
	})
}

// Logout handles POST /admin/api/auth/logout. It reads the refresh token from
// the cookie, deletes it from the database, and clears the cookie.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err == nil && cookie.Value != "" {
		if err := h.service.Logout(r.Context(), cookie.Value); err != nil {
			slog.Error("logout failed to delete refresh token", "error", err)
			// Continue to clear cookie even if DB delete fails.
		}
	}

	h.clearRefreshCookie(w)
	server.JSON(w, http.StatusOK, map[string]string{
		"message": "logged out",
	})
}

// Me handles GET /admin/api/auth/me. It reads the authenticated admin's ID
// and email from the request context (set by the auth middleware) and returns
// them in the response.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	adminID := AdminIDFromContext(r.Context())
	email := EmailFromContext(r.Context())

	if adminID == "" {
		server.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated", nil)
		return
	}

	server.JSON(w, http.StatusOK, map[string]string{
		"id":    adminID,
		"email": email,
	})
}

// logAudit sends an audit event if the audit service is configured.
func (h *Handler) logAudit(ctx context.Context, event audit.Event) {
	if h.auditService != nil {
		h.auditService.Log(ctx, event)
	}
}

// setRefreshCookie sets the refresh token as an httpOnly cookie on the response.
func (h *Handler) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     refreshCookiePath,
		MaxAge:   refreshCookieAge,
		HttpOnly: true,
		Secure:   !h.devMode,
		SameSite: http.SameSiteStrictMode,
	})
}

// clearRefreshCookie removes the refresh token cookie by setting it to an
// empty value with an immediate expiration.
func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   !h.devMode,
		SameSite: http.SameSiteStrictMode,
	})
}
