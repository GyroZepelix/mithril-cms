package server

import (
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// requireJSON returns a middleware that enforces Content-Type: application/json
// on POST, PUT, and PATCH requests that carry a body. Requests with a
// multipart Content-Type (e.g. file uploads) are exempt.
func requireJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			if r.ContentLength != 0 {
				ct := r.Header.Get("Content-Type")
				mediaType, _, _ := mime.ParseMediaType(ct)
				if strings.HasPrefix(mediaType, "multipart/") {
					next.ServeHTTP(w, r)
					return
				}
				if mediaType != "application/json" {
					Error(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE",
						"Content-Type must be application/json", nil)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// requestLogger returns a middleware that logs each HTTP request using slog.
// It captures the method, path, remote address, status code, response size,
// and duration for every request.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration", time.Since(start).String(),
			"remote", r.RemoteAddr,
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}
