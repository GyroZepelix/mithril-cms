package server

import (
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
)

// newSPAHandler returns an http.HandlerFunc that serves the admin single-page
// application. In dev mode it reverse-proxies to the Vite dev server at
// localhost:5173. In production, if an embedded filesystem is provided, it
// serves the built SPA assets with proper caching. Otherwise it returns a
// placeholder page prompting the user to build the admin SPA.
//
// Requests to API paths (/api/ or /admin/api/) are never served by the SPA;
// they receive a JSON 404 so that API misses don't return HTML.
func newSPAHandler(devMode bool, adminFS fs.FS) http.HandlerFunc {
	if devMode {
		return newDevProxyHandler()
	}
	if adminFS != nil {
		return newEmbeddedSPAHandler(adminFS)
	}
	return prodPlaceholderHandler
}

// isAPIPath reports whether the request path belongs to an API route that
// should never be handled by the SPA catch-all.
func isAPIPath(p string) bool {
	return p == "/api" || strings.HasPrefix(p, "/api/") ||
		p == "/admin/api" || strings.HasPrefix(p, "/admin/api/")
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

// newEmbeddedSPAHandler returns a handler that serves the embedded admin SPA.
// Static assets with hashed filenames get long-lived cache headers. All paths
// that don't match a real file serve index.html for client-side routing.
func newEmbeddedSPAHandler(distFS fs.FS) http.HandlerFunc {
	// Read index.html once at startup. If it's missing, the embed is broken
	// and we fall through to the placeholder.
	indexHTML, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		slog.Warn("embedded admin FS has no index.html, using placeholder", "error", err)
		return prodPlaceholderHandler
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if isAPIPath(r.URL.Path) {
			Error(w, http.StatusNotFound, "NOT_FOUND", "The requested API endpoint does not exist", nil)
			return
		}

		// Strip the /admin/ prefix since the embedded FS is rooted at dist/.
		// Vite builds with base: "/admin/", so asset references in the HTML
		// are /admin/assets/..., and the browser requests those paths.
		filePath := strings.TrimPrefix(r.URL.Path, "/admin")
		filePath = strings.TrimPrefix(filePath, "/")
		if filePath == "" {
			filePath = "index.html"
		}

		// Clean the path to prevent directory traversal.
		filePath = path.Clean(filePath)

		// Defense-in-depth: reject paths that escape the root after cleaning.
		// This should never happen with embed.FS, but serve SPA fallback to be safe.
		if filePath == ".." || strings.HasPrefix(filePath, "../") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(indexHTML)
			return
		}

		// Try to open the requested file from the embedded FS.
		// Note: We open the file twice here (once to check stat, once in ServeFileFS).
		// This is acceptable because embed.FS is immutable and these operations are cheap.
		f, err := distFS.Open(filePath)
		if err == nil {
			defer f.Close()

			stat, statErr := f.Stat()
			if statErr == nil && !stat.IsDir() {
				setCacheHeaders(w, filePath)
				http.ServeFileFS(w, r, distFS, filePath)
				return
			}
		}

		// File not found or is a directory: serve index.html for SPA routing.
		// index.html must never be cached so the browser always gets the
		// latest version pointing to current hashed assets.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(indexHTML)
	}
}

// setCacheHeaders sets Cache-Control headers based on the file path.
// Hashed asset files (in the assets/ directory) get immutable long-lived
// caching. All other files get short caching.
func setCacheHeaders(w http.ResponseWriter, filePath string) {
	// Always set nosniff to prevent MIME type confusion attacks.
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if strings.HasPrefix(filePath, "assets/") {
		// Vite produces content-hashed filenames in assets/, so they are
		// safe to cache indefinitely.
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	}
}

// prodPlaceholderHandler serves a placeholder page in production when the
// React admin SPA has not been embedded into the binary.
func prodPlaceholderHandler(w http.ResponseWriter, r *http.Request) {
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
<p>Admin UI not built yet. Run <code>make build-all</code> to build the SPA and embed it.</p>
</body>
</html>`)
}
