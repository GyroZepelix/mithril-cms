package routes

import (
	"fmt"
	"net/http"
)

func getContentHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "get")
}

func getContentsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "getall")
}

func postContentHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "post")
}
