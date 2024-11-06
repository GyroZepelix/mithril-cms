package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()

	r.Route("/api", func(r chi.Router) {
		r.Get("/content", getContentHandler)
		r.Post("/content", postContentHandler)

		r.Get("/contents", getContentsHandler)
	})

	return r
}
