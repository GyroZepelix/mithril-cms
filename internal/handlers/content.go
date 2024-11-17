package handlers

import (
	"fmt"
	"net/http"
)

func (e Env) handleGetContent(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "get")
}

func (e Env) handleListContents(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "getall")
}

func (e Env) handlePostContent(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "post")
}
