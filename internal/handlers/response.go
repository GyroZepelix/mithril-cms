package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/GyroZepelix/mithril-cms/internal/logging"
)

func handleJsonResponse(w http.ResponseWriter, v any) {
	jsonResponse, err := json.Marshal(v)
	if err != nil {
		logging.Error("Failed marshaling response: ", err)
		handleInternalServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

type ResponseError struct {
	StatusCode int    `json:"status_code"`
	Error      string `json:"error"`
}

func handleGenericError(w http.ResponseWriter, err error, statusCode int) {
	w.WriteHeader(statusCode)
	response, err := json.Marshal(ResponseError{
		StatusCode: statusCode,
		Error:      err.Error(),
	})
	if err != nil {
		logging.Error("Failed marshaling error: ", err)
		fmt.Fprintln(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func handleInternalServerError(w http.ResponseWriter, err error) {
	handleGenericError(w, err, http.StatusInternalServerError)
}

func handleBadRequest(w http.ResponseWriter, err error) {
	handleGenericError(w, err, http.StatusBadRequest)
}

func handleNotFound(w http.ResponseWriter, err error) {
	handleGenericError(w, err, http.StatusNotFound)
}
