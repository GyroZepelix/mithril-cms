package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()

	r.Route("/api", func(r chi.Router) {
		r.Get("/content", handleGetContent)
		r.Post("/content", handlePostContent)
		r.Get("/contents", handleListContents)

		r.Get("/users", handleListUsers)
		r.Get("/user", handleGetUser)
	})

	return r
}
