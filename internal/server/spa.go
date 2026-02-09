package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// newSPAHandler returns an http.HandlerFunc that serves the admin single-page
// application. In dev mode it reverse-proxies to the Vite dev server at
// localhost:5173. In production it returns a placeholder page until the full
// go:embed integration is wired up in Task 17.
//
// Requests to API paths (/api/ or /admin/api/) are never served by the SPA;
// they receive a JSON 404 so that API misses don't return HTML.
func newSPAHandler(devMode bool) http.HandlerFunc {
	if devMode {
		return newDevProxyHandler()
	}
	return prodSPAHandler
}

// isAPIPath reports whether the request path belongs to an API route that
// should never be handled by the SPA catch-all.
func isAPIPath(path string) bool {
	return strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/admin/api/")
}

// newDevProxyHandler returns a handler that reverse-proxies all requests to
// the Vite dev server for hot-reload during development.
func newDevProxyHandler() http.HandlerFunc {
	target := &url.URL{Scheme: "http", Host: "localhost:5173"}
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Override the error handler to log proxy failures gracefully.
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Warn("SPA dev proxy error (is Vite running?)",
			"error", err,
			"path", r.URL.Path,
		)
		Error(w, http.StatusBadGateway, "DEV_PROXY_ERROR",
			"Admin UI dev server not reachable. Is Vite running on :5173?", nil)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if isAPIPath(r.URL.Path) {
			Error(w, http.StatusNotFound, "NOT_FOUND", "The requested API endpoint does not exist", nil)
			return
		}
		proxy.ServeHTTP(w, r)
	}
}

// prodSPAHandler serves a placeholder page in production until the React SPA
// is embedded into the binary (Task 17).
func prodSPAHandler(w http.ResponseWriter, r *http.Request) {
	if isAPIPath(r.URL.Path) {
		Error(w, http.StatusNotFound, "NOT_FOUND", "The requested API endpoint does not exist", nil)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>Mithril CMS</title></head>
<body>
<h1>Mithril CMS</h1>
<p>Admin UI not built yet. Run <code>make build-admin</code> to build the SPA.</p>
</body>
</html>`)
}
