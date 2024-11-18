package handlers

import (
	"net/http"

	"github.com/GyroZepelix/mithril-cms/internal/logic/user"
	"github.com/GyroZepelix/mithril-cms/internal/storage/persistence"
	"github.com/go-chi/chi/v5"
)

type Env struct {
	UserManager *user.Manager
}

func NewRouter(e *Env) http.Handler {
	r := chi.NewRouter()

	r.Route("/api", func(r chi.Router) {

		r.Route("/contents", func(r chi.Router) {
			r.Get("/", e.handleListContents)
			r.Get("/{id}", e.handleGetContent)
			r.Post("/", e.handlePostContent)
		})

		r.Route("/users", func(r chi.Router) {
			r.Get("/", e.handleListUsers)
			r.Get("/{id}", e.handleGetUser)
		})

	})

	return r
}
