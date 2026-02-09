// Package server provides the HTTP server, router, middleware, and JSON
// response helpers for the Mithril CMS.
package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// FieldError represents a single field-level validation error in an API response.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// PaginationMeta holds pagination metadata for list responses.
type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// successResponse wraps a single data item.
type successResponse struct {
	Data any `json:"data"`
}

// paginatedResponse wraps a list of data items with pagination metadata.
type paginatedResponse struct {
	Data any            `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

// errorBody is the inner structure of an error response.
type errorBody struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Details []FieldError `json:"details,omitempty"`
}

// errorResponse is the top-level error response envelope.
type errorResponse struct {
	Error errorBody `json:"error"`
}

// JSON writes a JSON response with the given status code. The data is wrapped
// in a {"data": ...} envelope per the API spec.
func JSON(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, successResponse{Data: data})
}

// Error writes a JSON error response with the given status code, error code,
// message, and optional field-level details.
func Error(w http.ResponseWriter, status int, code string, message string, details []FieldError) {
	writeJSON(w, status, errorResponse{
		Error: errorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// Paginated writes a JSON list response with pagination metadata.
func Paginated(w http.ResponseWriter, data any, meta PaginationMeta) {
	writeJSON(w, http.StatusOK, paginatedResponse{Data: data, Meta: meta})
}

// writeJSON marshals v to JSON and writes it to the response writer.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		// At this point headers are already sent, so we can only log.
		slog.Error("failed to encode JSON response", "error", err)
	}
}
