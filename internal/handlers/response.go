package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/GyroZepelix/mithril-cms/internal/logging"
)

const (
	msgInternalServerError = "Internal server error"
)

func handleJsonResponse(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(v); err != nil {
		logging.Error("Failed to marshal JSON response: ", err)
		handleInternalServerError(w, msgInternalServerError)
	}
}

type ResponseError struct {
	StatusCode int    `json:"status_code"`
	Error      string `json:"error"`
}

func handleGenericError(w http.ResponseWriter, clientErrorMessage string, statusCode int) {
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

func handleInternalServerError(w http.ResponseWriter, clientErrorMessage string) {
	handleGenericError(w, clientErrorMessage, http.StatusInternalServerError)
}

func handleBadRequest(w http.ResponseWriter, clientErrorMessage string) {
	handleGenericError(w, clientErrorMessage, http.StatusBadRequest)
}

func handleNotFound(w http.ResponseWriter, clientErrorMessage string) {
	handleGenericError(w, clientErrorMessage, http.StatusNotFound)
}
