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

func RoleRequired(r constant.UserRole) func(http.Handler) http.Handler {
	expectedRole := constant.UserRoleName[r]

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			actualRole := ctx.Value(auth.RoleKey)

			if actualRole != expectedRole {
				response.Forbidden(w, "Permission denied")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

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
		role := claims[auth.RoleKey]

		ctx := r.Context()
		ctx = context.WithValue(ctx, auth.UserIdKey, userID)
		ctx = context.WithValue(ctx, auth.RoleKey, role)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func getTokenFromRequest(r *http.Request) string {
	tokenAuth := r.Header.Get("Authorization")

	if strings.HasPrefix(tokenAuth, bearerPrefix) {
		return strings.TrimPrefix(tokenAuth, bearerPrefix)
	}

	return ""
}
