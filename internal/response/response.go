package response

import (
	"encoding/json"
	"net/http"

	"github.com/GyroZepelix/mithril-cms/internal/logging"
)

const (
	MsgInternalServerError = "Internal server error"
)

func JsonResponse(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(v); err != nil {
		logging.Error("Failed to marshal JSON response: ", err)
		InternalServerError(w, MsgInternalServerError)
	}
}

type ResponseError struct {
	StatusCode int `json:"status_code"`
	Error      any `json:"error"`
}

func GenericError(w http.ResponseWriter, clientErrorMessage any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ResponseError{
		StatusCode: statusCode,
		Error:      clientErrorMessage,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logging.Error("Failed to marshal error response: ", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func InternalServerError(w http.ResponseWriter, clientErrorMessage any) {
	GenericError(w, clientErrorMessage, http.StatusInternalServerError)
}

func BadRequest(w http.ResponseWriter, clientErrorMessage any) {
	GenericError(w, clientErrorMessage, http.StatusBadRequest)
}

func NotFound(w http.ResponseWriter, clientErrorMessage any) {
	GenericError(w, clientErrorMessage, http.StatusNotFound)
}

func Unauthorized(w http.ResponseWriter, clientErrorMessage any) {
	GenericError(w, clientErrorMessage, http.StatusUnauthorized)
}

func Forbidden(w http.ResponseWriter, clientErrorMessage any) {
	GenericError(w, clientErrorMessage, http.StatusForbidden)
}
