package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func (e Env) handleGetUser(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "get")
}

func (e Env) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := e.DB.ListUsers(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Fatal(err)
	}
	log.Println(users)

	usersResponse, err := json.Marshal(users)
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(usersResponse)
	log.Println("called handleListUsers")
}

func (e Env) handlePostUser(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "post")
}
