package routes

import (
	"fmt"
	"net/http"
)

func handleGetContent(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "get")
}

func handlePostContent(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "post")
}
