package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/GyroZepelix/mithril-cms/internal/logging"
	"github.com/GyroZepelix/mithril-cms/internal/service/auth"
	"github.com/GyroZepelix/mithril-cms/internal/service/user"
	"github.com/GyroZepelix/mithril-cms/internal/service/validation"
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

	userData, err := e.UserManager.GetUser(int32(userId), r.Context())
	if err != nil {
		switch {
		case errors.Is(err, user.ErrNotFound):
			handleNotFound(w, "User not found")
			return
		default:
			logging.Error("User couldnt be fetched: ", err)
			handleInternalServerError(w, msgInternalServerError)
			return
		}
	}

	handleJsonResponse(w, userData)
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
	var userParams struct {
		Username string `json:"username" validate:"required"`
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&userParams); err != nil {
		handleBadRequest(w, "User could not be deserialised")
		return
	}
	if err := e.Validator.Struct(userParams); err != nil {
		handleBadRequest(w, validation.ParseHttpErrorMessage(err))
		return
	}

	hashedPassword, err := auth.HashPassword(userParams.Password)
	if err != nil {
		logging.Error("Error hashing a password while creating a User: ", err)
		handleInternalServerError(w, msgInternalServerError)
		return
	}
	createdUser, err := e.UserManager.CreateUser(
		userParams.Username,
		userParams.Email,
		hashedPassword,
		r.Context(),
	)
	if err != nil {
		if error, ok := err.(*user.ErrUniqueValueViolation); ok {
			handleBadRequest(w, error)
			return
		}
		logging.Errorf("Error encountered creating a User: %v\nErr: %s", userParams, err)
		handleInternalServerError(w, msgInternalServerError)
		return
	}

	handleJsonResponse(w, createdUser)
}
