package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/GyroZepelix/mithril-cms/internal/constant"
	"github.com/GyroZepelix/mithril-cms/internal/response"
	"github.com/GyroZepelix/mithril-cms/internal/service/auth"
	"github.com/golang-jwt/jwt"
)

const (
	bearerPrefix string = "Bearer "
)

// JWTAuth is a middleware function that performs JWT authentication.
// It validates the JWT token from the request, extracts claims, and adds them to the request context.
//
// The middleware:
// 1. Extracts the JWT token from the request.
// 2. Validates the token.
// 3. If invalid, responds with a "Forbidden" error.
// 4. If valid, extracts user ID and role from claims.
// 5. Adds user ID and role to the request context.
// 6. Calls the next handler in the chain.
//
// Example usage:
//
//	http.Handle("/protected", JWTAuth(protectedHandler))
func JWTAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := getTokenFromRequest(r)

		token, err := auth.ValidateJWT(tokenString)
		if err != nil {
			response.Forbidden(w, "Permission denied")
			return
		}

		if !token.Valid {
			response.Forbidden(w, "Premission denied")
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		userID := claims[auth.UserIdKey]
		role := claims[auth.RoleKey].(string)
		userRole := constant.UserRoleMap[role]

		ctx := r.Context()
		ctx = context.WithValue(ctx, auth.UserIdKey, userID)
		ctx = context.WithValue(ctx, auth.RoleKey, userRole)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// getTokenFromRequest extracts the JWT token from the HTTP request's Authorization header.
// It expects the token to be in the format "Bearer <token>".
//
// If the Authorization header is not present or doesn't have the correct prefix,
// an empty string is returned.
//
// This function is intended for internal use by the JWTAuth middleware.
func getTokenFromRequest(r *http.Request) string {
	tokenAuth := r.Header.Get("Authorization")

	if strings.HasPrefix(tokenAuth, bearerPrefix) {
		return strings.TrimPrefix(tokenAuth, bearerPrefix)
	}

	return ""
}
