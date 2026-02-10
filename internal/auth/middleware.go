package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/GyroZepelix/mithril-cms/internal/server"
)

type contextKey string

const (
	// ContextKeyAdminID is the context key for the authenticated admin's UUID.
	ContextKeyAdminID contextKey = "admin_id"
	// ContextKeyEmail is the context key for the authenticated admin's email.
	ContextKeyEmail contextKey = "email"
)

// Middleware returns an HTTP middleware that validates JWT Bearer tokens from
// the Authorization header. On success it sets the admin ID and email in the
// request context. On failure it returns a 401 JSON error response.
func Middleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				server.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header", nil)
				return
			}

			// Expect "Bearer <token>" format.
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				server.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization header format", nil)
				return
			}

			tokenString := parts[1]
			claims, err := ValidateAccessToken(tokenString, jwtSecret)
			if err != nil {
				server.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token", nil)
				return
			}

			// Set admin info in context for downstream handlers.
			ctx := context.WithValue(r.Context(), ContextKeyAdminID, claims.AdminID())
			ctx = context.WithValue(ctx, ContextKeyEmail, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminIDFromContext extracts the authenticated admin's UUID from the request
// context. Returns an empty string if no admin is authenticated.
func AdminIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyAdminID).(string)
	return v
}

// EmailFromContext extracts the authenticated admin's email from the request
// context. Returns an empty string if no admin is authenticated.
func EmailFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyEmail).(string)
	return v
}
