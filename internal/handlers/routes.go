package handlers

import (
	"net/http"

	"github.com/GyroZepelix/mithril-cms/internal/middleware"
	"github.com/GyroZepelix/mithril-cms/internal/service/user"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type ServiceContext struct {
	UserManager *user.Manager
	Validator   *validator.Validate
}

func NewRouter(s *ServiceContext) http.Handler {
	r := chi.NewRouter()

	r.Route("/api", func(r chi.Router) {

		r.Get("/login", s.handleLoginUser)
		r.Post("/register", s.handleRegisterUser)

		r.Route("/contents", func(r chi.Router) {
			r.Use(middleware.JWTAuth)

			r.Get("/", s.handleListContents)
			r.Get("/{id}", s.handleGetContent)
			r.Post("/", s.handlePostContent)
			r.Put("{id}", s.handlePutContent)
		})

		r.Route("/users", func(r chi.Router) {
			r.Get("/", s.handleListUsers)
			r.Get("/{id}", s.handleGetUser)
			// r.Post("/", e.handlePostUser)
		})

	})

	return r
}
