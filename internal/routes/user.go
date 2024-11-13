package routes

import (
	"fmt"
	"net/http"
)

func handleGetUser(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "get")
}

func handleListUsers(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "getall")
}

func handlePostUser(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "post")
}
