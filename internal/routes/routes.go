package routes

import (
	"net/http"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/content", contentHandler)

	return mux
}

func contentHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetContent(w, r)
	case http.MethodPost:
		handlePostContent(w, r)
	}
}
