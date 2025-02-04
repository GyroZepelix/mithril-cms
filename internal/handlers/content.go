package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/GyroZepelix/mithril-cms/internal/errs"
	"github.com/GyroZepelix/mithril-cms/internal/logging"
	"github.com/GyroZepelix/mithril-cms/internal/response"
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

	// contentData, err := s.UserManager.GetUser(contentId, r.Context())
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
	fmt.Fprintln(w, "post")
}

func (s ServiceContext) handlePutContent(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "put")
}
