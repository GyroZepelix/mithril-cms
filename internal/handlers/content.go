package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/GyroZepelix/mithril-cms/internal/errs"
	"github.com/GyroZepelix/mithril-cms/internal/logging"
	"github.com/GyroZepelix/mithril-cms/internal/response"
	"github.com/GyroZepelix/mithril-cms/internal/service/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (s ServiceContext) handleGetContent(w http.ResponseWriter, r *http.Request) {
	contentIdParam := chi.URLParam(r, "id")
	contentId, err := uuid.Parse(contentIdParam)
	if err != nil {
		logging.Errorf("Couldnt convert id %s to UUID: %s", contentIdParam, err)
		response.BadRequest(w, "Invalid content ID format")
		return
	}

	contentData, err := s.ContentManager.GetContent(contentId, r.Context())
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrNotFound):
			response.NotFound(w, "Content not found")
			return
		default:
			logging.Error("Content couldnt be fetched: ", err)
			response.InternalServerError(w, response.MsgInternalServerError)
			return
		}
	}

	response.JsonResponse(w, contentData)
}

func (s ServiceContext) handleListContents(w http.ResponseWriter, r *http.Request) {
	contents, err := s.ContentManager.ListContents(r.Context())
	if err != nil {
		logging.Error("Content couldn't be fetched: ", err)
		response.InternalServerError(w, response.MsgInternalServerError)
		return
	}

	response.JsonResponse(w, contents)
}

func (s ServiceContext) handlePostContent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var createPostParams struct {
		Title   string `json:"title" validate:"required"`
		Content string `json:"content" validate:"required"`
	}
	userId, ok := ctx.Value(auth.UserIdKey).(uuid.UUID)

	if err := json.NewDecoder(r.Body).Decode(&createPostParams); err != nil {
		response.BadRequest(w, "Post could not be deserialised")
		return
	}
	if err := s.Validator.Struct(createPostParams); err != nil {
		response.UnprocessableContent(w, errs.MapValidationError(err))
		return
	}
	if !ok {
		logging.Errorf("userId was %s while checking current user. Are you authenticating before calling this?", &userId)
		response.InternalServerError(w, response.MsgInternalServerError)
		return
	}

	contentData, err := s.ContentManager.CreateContent(
		createPostParams.Title,
		createPostParams.Content,
		userId,
		ctx,
	)
	if err != nil {
		logging.Errorf("Error encountered creating a Post: %v\nErr: %s", createPostParams, err)
		response.InternalServerError(w, response.MsgInternalServerError)
		return
	}

	response.JsonResponse(w, contentData)
}

func (s ServiceContext) handlePutContent(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "put")
}
