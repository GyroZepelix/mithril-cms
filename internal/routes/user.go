package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/GyroZepelix/mithril-cms/internal/logging"
	"github.com/go-chi/chi/v5"
)

func (e Env) handleGetUser(w http.ResponseWriter, r *http.Request) {
	userIdParam := chi.URLParam(r, "id")
	userId, err := strconv.ParseInt(userIdParam, 10, 32)
	if err != nil {
		logging.Errorf("Couldnt convert id %s to integer: %s", userIdParam, err)
		handleBadRequest(w, err)
		return
	}

	user, err := e.DB.GetUser(r.Context(), int32(userId))
	if err != nil {
		handleInternalServerError(w, err)
		return
	}

	userResponse, err := json.Marshal(user)
	if err != nil {
		log.Fatal(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(userResponse)
	logging.Info("called handleGetUser")
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
