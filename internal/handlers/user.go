package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/GyroZepelix/mithril-cms/internal/logging"
	"github.com/GyroZepelix/mithril-cms/internal/logic/userLogic"
	"github.com/GyroZepelix/mithril-cms/internal/storage/persistence"
	"github.com/go-chi/chi/v5"
)

func (e Env) handleGetUser(w http.ResponseWriter, r *http.Request) {
	userIdParam := chi.URLParam(r, "id")
	userId, err := strconv.ParseInt(userIdParam, 10, 32)
	if err != nil {
		logging.Errorf("Couldnt convert id %s to integer: %s", userIdParam, err)
		handleBadRequest(w, "Invalid user ID format")
		return
	}

	user, err := e.UserManager.GetUser(int32(userId), r.Context())
	if err != nil {
		switch {
		case errors.Is(err, userLogic.ErrNotFound):
			handleNotFound(w, "User not found")
			return
		default:
			logging.Error("User couldnt be fetched: ", err)
			handleInternalServerError(w, msgInternalServerError)
			return
		}
	}

	handleJsonResponse(w, user)
}

func (e Env) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := e.UserManager.ListUsers(r.Context())
	if err != nil {
		logging.Error("User couldn't be fetched: ", err)
		handleInternalServerError(w, msgInternalServerError)
		return
	}

	handleJsonResponse(w, users)
}

func (e Env) handlePostUser(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "post")
}
