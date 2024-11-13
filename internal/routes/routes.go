package routes

import (
	"net/http"

	"github.com/GyroZepelix/mithril-cms/internal/storage/persistence"
	"github.com/go-chi/chi/v5"
)

type Env struct {
	DB *persistence.Queries
}

func NewRouter(e *Env) http.Handler {
	r := chi.NewRouter()

	r.Route("/api", func(r chi.Router) {
		r.Get("/content", e.handleGetContent)
		r.Post("/content", e.handlePostContent)
		r.Get("/contents", e.handleListContents)

		r.Get("/users", e.handleListUsers)
		r.Get("/user", e.handleGetUser)
	})

	return r
}
