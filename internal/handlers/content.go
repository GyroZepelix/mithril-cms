package handlers

import (
	"fmt"
	"net/http"
)

func (s ServiceContext) handleGetContent(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "get")
}

func (s ServiceContext) handleListContents(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "getall")
}

func (s ServiceContext) handlePostContent(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "post")
}
