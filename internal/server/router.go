package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/GyroZepelix/mithril-cms/internal/database"
	"github.com/GyroZepelix/mithril-cms/internal/schema"
)

// AuthHandler defines the interface for authentication HTTP handlers, allowing
// the router to be decoupled from the concrete auth implementation.
type AuthHandler interface {
	Login(w http.ResponseWriter, r *http.Request)
	Refresh(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
	Me(w http.ResponseWriter, r *http.Request)
}

// Dependencies holds all injectable dependencies used by route handlers.
type Dependencies struct {
	DB             *database.DB
	Engine         *schema.Engine
	Schemas        []schema.ContentType
	DevMode        bool
	AuthHandler    AuthHandler
	AuthMiddleware func(http.Handler) http.Handler
}

// NewRouter builds the chi router with the full route tree, middleware stack,
// and placeholder handlers. Real handler implementations will be wired in as
// they are built in subsequent tasks.
func NewRouter(deps Dependencies) chi.Router {
	r := chi.NewRouter()

	// --- Global middleware stack ---
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestLogger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware(deps.DevMode))

	// --- Health check ---
	r.Get("/health", healthHandler(deps))

	// --- Public API ---
	r.Route("/api", func(r chi.Router) {
		r.Use(requireJSON)
		r.Get("/{contentType}", notImplemented)
		r.Get("/{contentType}/{id}", notImplemented)
	})

	// --- Admin API ---
	r.Route("/admin/api", func(r chi.Router) {
		r.Use(requireJSON)

		// Public auth routes (no auth middleware required).
		if deps.AuthHandler != nil {
			r.Post("/auth/login", deps.AuthHandler.Login)
			r.Post("/auth/refresh", deps.AuthHandler.Refresh)
			r.Post("/auth/logout", deps.AuthHandler.Logout)
		} else {
			r.Post("/auth/login", notImplemented)
			r.Post("/auth/refresh", notImplemented)
			r.Post("/auth/logout", notImplemented)
		}

		// Protected routes - require valid JWT.
		r.Group(func(r chi.Router) {
			if deps.AuthMiddleware != nil {
				r.Use(deps.AuthMiddleware)
			}

			if deps.AuthHandler != nil {
				r.Get("/auth/me", deps.AuthHandler.Me)
			} else {
				r.Get("/auth/me", notImplemented)
			}

			// Content type introspection.
			r.Get("/content-types", notImplemented)
			r.Get("/content-types/{name}", notImplemented)

			// Content CRUD.
			r.Route("/content/{contentType}", func(r chi.Router) {
				r.Get("/", notImplemented)
				r.Post("/", notImplemented)
				r.Get("/{id}", notImplemented)
				r.Put("/{id}", notImplemented)
				r.Post("/{id}/publish", notImplemented)
			})

			// Media management.
			r.Route("/media", func(r chi.Router) {
				r.Post("/", notImplemented)
				r.Get("/", notImplemented)
				r.Delete("/{id}", notImplemented)
			})

			// Audit log.
			r.Get("/audit-log", notImplemented)

			// Schema refresh.
			r.Post("/schema/refresh", notImplemented)
		})
	})

	// --- Public media serving ---
	r.Get("/media/{filename}", notImplemented)

	// --- SPA catch-all (must be last) ---
	r.NotFound(newSPAHandler(deps.DevMode))

	return r
}

// corsMiddleware returns a CORS middleware configured for the application.
// In dev mode all origins are allowed; in production only same-origin is
// permitted (can be made configurable via config in a future task).
func corsMiddleware(devMode bool) func(http.Handler) http.Handler {
	var allowedOrigins []string
	if devMode {
		allowedOrigins = []string{"http://localhost:5173", "http://localhost:8080"}
	} else {
		// In production, only same-origin requests are allowed by default.
		// This can be extended with a config value later.
		allowedOrigins = []string{}
	}

	return cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	})
}

// healthHandler returns a handler that reports the health status of the
// application, including a database connectivity check.
func healthHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := deps.DB.Health(r.Context()); err != nil {
			Error(w, http.StatusServiceUnavailable, "DB_UNHEALTHY", "database health check failed", nil)
			return
		}
		JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// notImplemented is a placeholder handler that returns 501 Not Implemented
// with a proper JSON error body for routes not yet built.
func notImplemented(w http.ResponseWriter, r *http.Request) {
	Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not yet implemented", nil)
}
