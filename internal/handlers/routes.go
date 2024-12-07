package handlers

import (
	"net/http"

	"github.com/GyroZepelix/mithril-cms/internal/service/user"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type ServiceContext struct {
	UserManager *user.Manager
	Validator   *validator.Validate
}

func NewRouter(e *ServiceContext) http.Handler {
	r := chi.NewRouter()

	r.Route("/api", func(r chi.Router) {

		r.Get("/login", e.handleLoginUser)
		r.Get("/register", e.handleRegisterUser)

		r.Route("/contents", func(r chi.Router) {
			r.Get("/", e.handleListContents)
			r.Get("/{id}", e.handleGetContent)
			r.Post("/", e.handlePostContent)
		})

		r.Route("/users", func(r chi.Router) {
			r.Get("/", e.handleListUsers)
			r.Get("/{id}", e.handleGetUser)
			// r.Post("/", e.handlePostUser)
		})

	})

	return r
}
