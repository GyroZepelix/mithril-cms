package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/GyroZepelix/mithril-cms/internal/errs"
	"github.com/GyroZepelix/mithril-cms/internal/logging"
	"github.com/GyroZepelix/mithril-cms/internal/service/auth"
	"github.com/go-chi/chi/v5"
)

func (s ServiceContext) handleGetUser(w http.ResponseWriter, r *http.Request) {
	userIdParam := chi.URLParam(r, "id")
	userId, err := strconv.ParseInt(userIdParam, 10, 32)
	if err != nil {
		logging.Errorf("Couldnt convert id %s to integer: %s", userIdParam, err)
		handleBadRequest(w, "Invalid user ID format")
		return
	}

	userData, err := s.UserManager.GetUser(int32(userId), r.Context())
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrNotFound):
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

func (s ServiceContext) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.UserManager.ListUsers(r.Context())
	if err != nil {
		logging.Error("User couldn't be fetched: ", err)
		handleInternalServerError(w, msgInternalServerError)
		return
	}

	handleJsonResponse(w, users)
}

func (s ServiceContext) handleRegisterUser(w http.ResponseWriter, r *http.Request) {
	var registerParams struct {
		Username string `json:"username" validate:"required"`
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&registerParams); err != nil {
		handleBadRequest(w, "User could not be deserialised")
		return
	}
	if err := s.Validator.Struct(registerParams); err != nil {
		handleBadRequest(w, errs.MapValidationError(err))
		return
	}

	hashedPassword, err := auth.HashPassword(registerParams.Password)
	if err != nil {
		logging.Error("Error hashing a password while creating a User: ", err)
		handleInternalServerError(w, msgInternalServerError)
		return
	}
	registeredUser, err := s.UserManager.CreateUser(
		registerParams.Username,
		registerParams.Email,
		hashedPassword,
		r.Context(),
	)
	if err != nil {
		if error, ok := err.(*errs.ErrUniqueValueViolation); ok {
			handleBadRequest(w, error)
			return
		}
		logging.Errorf("Error encountered creating a User: %v\nErr: %s", registerParams, err)
		handleInternalServerError(w, msgInternalServerError)
		return
	}

	handleJsonResponse(w, registeredUser)
}

var loginErrorMessage string = "Username or password not found!"

func (s ServiceContext) handleLoginUser(w http.ResponseWriter, r *http.Request) {
	var loginParams struct {
		Username string `json:"username" validate:"required"`
		Password string `json:"password" validate:"required"`
	}
	loginParams.Username = r.URL.Query().Get("username")
	loginParams.Password = r.URL.Query().Get("password")
	if err := s.Validator.Struct(loginParams); err != nil {
		handleBadRequest(w, errs.MapValidationError(err))
		return
	}

	userData, err := s.UserManager.GetUserByUsername(loginParams.Username, r.Context())
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrNotFound):
			handleUnauthorized(w, loginErrorMessage)
			return
		default:
			logging.Error("User couldnt be fetched: ", err)
			handleInternalServerError(w, msgInternalServerError)
			return
		}
	}

	logging.Info(userData)
	if !auth.CheckPasswordHash(loginParams.Password, userData.Password) {
		handleUnauthorized(w, loginErrorMessage)
		return
	}

	token, err := auth.CreateJWT(userData.ID, userData.Role)
	if err != nil {
		handleInternalServerError(w, msgInternalServerError)
	}

	var loginResponse struct {
		Token string `json:"token"`
	}

	loginResponse.Token = token

	handleJsonResponse(w, loginResponse)
}
