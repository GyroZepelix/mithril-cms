package middleware

import (
	"context"
	"net/http"

	"github.com/GyroZepelix/mithril-cms/internal/constant"
	"github.com/GyroZepelix/mithril-cms/internal/logging"
	"github.com/GyroZepelix/mithril-cms/internal/service/auth"
	"github.com/GyroZepelix/mithril-cms/internal/service/permission"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type PermissionMiddleware interface {
	RequirePermission(next http.HandlerFunc, accessPermissions ...permission.AccessPermission) http.HandlerFunc
	UseResourceId(newResourceId string) func(http.Handler) http.HandlerFunc
}

type permissionMiddleware struct {
	resourceIdKey       string
	unauthorizedHandler func(w http.ResponseWriter)
	ownershipChecker    permission.OwnershipChecker
	permissionValidator permission.PermissionValidator
}

func NewPermissionMiddleware(defaultResourceIdKey string, unauthorizedHandler func(http.ResponseWriter), ownershipChecker permission.OwnershipChecker, permissionValidator permission.PermissionValidator) *permissionMiddleware {
	return &permissionMiddleware{
		resourceIdKey:       defaultResourceIdKey,
		unauthorizedHandler: unauthorizedHandler,
		ownershipChecker:    ownershipChecker,
		permissionValidator: permissionValidator,
	}
}

// RequirePermission is a middleware method that checks if the user has the required permissions to access a resource.
// It takes an http.HandlerFunc and a variadic number of permission.AccessPermission as input.
//
// The middleware performs the following steps:
//  1. Extracts user ID and role from the request context.
//  2. Validates the user's permissions against the provided access permissions.
//  3. For "Owned" permission level, it checks if the user owns the resource.
//  4. If the user doesn't have the required permissions, it calls the unauthorized handler.
//  5. If the user has the required permissions, it calls the next handler in the chain.
//
// Parameters:
//   - next: The next http.HandlerFunc in the middleware chain.
//   - accessPermissions: A variadic list of permission.AccessPermission to check against.
//
// Returns:
//   - An http.HandlerFunc that wraps the permission checking logic.
//
// Notes:
//   - This method assumes that authentication has been performed and user information is available in the context.
//   - AccessPermission order matters, first accessPermissions argument will be checked first
func (m *permissionMiddleware) RequirePermission(next http.HandlerFunc, accessPermissions ...permission.AccessPermission) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userId, ok := ctx.Value(auth.UserIdKey).(uuid.UUID)
		if !ok {
			logging.Warnf("userId was %s while checking ownership. Are you authenticating before authorizing?", &userId)
			m.unauthorizedHandler(w)
			return
		}
		userRole, ok := ctx.Value(auth.RoleKey).(constant.UserRole)
		if !ok {
			logging.Warn("userRole was blank while checking ownership. Are you authenticating before authorizing?")
			m.unauthorizedHandler(w)
			return
		}

		var validPermission *permission.AccessPermission = nil
		for _, accessPermission := range accessPermissions {
			hasValidPermission := m.permissionValidator.ValidatePermission(userRole, accessPermission)
			if hasValidPermission {
				validPermission = &accessPermission
				break
			}
		}

		if validPermission == nil {
			m.unauthorizedHandler(w)
			return
		}

		switch validPermission.PermissionLevel {
		case permission.Owned:
			resourceIdKey := m.resourceIdKey
			if override, ok := ctx.Value(resourceIdKeyOverrideKey).(string); ok && override != "" {
				resourceIdKey = override
			}
			resourceIdParam := chi.URLParam(r, resourceIdKey)
			resourceId, err := uuid.Parse(resourceIdParam)
			if err != nil {
				logging.Warnf("Value of key %s is %s, should be an UUID!", resourceIdKey, resourceIdParam)
				m.unauthorizedHandler(w)
				return
			}

			isOwner, err := m.ownershipChecker.IsOwner(userId, validPermission.ResourceType, resourceId, ctx)
			if err != nil {
				logging.Error("Error while checking user ownership:", err)
				m.unauthorizedHandler(w)
				return
			}

			if !isOwner {
				m.unauthorizedHandler(w)
				return
			}
		case permission.All:
		}

		next.ServeHTTP(w, r)
	})
}

const resourceIdKeyOverrideKey string = "RESOURCE_ID_KEY_OVERRIDE"

// UseResourceId is a middleware generator that allows overriding the default resource ID key used in URL parameters.
//
// Parameters:
//   - newResourceId: The new resource ID key to be used.
//
// Returns:
//   - A function that generates an http.HandlerFunc middleware.
//
// Usage:
//
//	router.With(permissionMiddleware.UseResourceId("custom_id")).Get("/resource/{custom_id}", handler)
func (m *permissionMiddleware) UseResourceId(newResourceId string) func(http.Handler) http.HandlerFunc {
	return func(next http.Handler) http.HandlerFunc {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = context.WithValue(ctx, resourceIdKeyOverrideKey, newResourceId)
			r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}
