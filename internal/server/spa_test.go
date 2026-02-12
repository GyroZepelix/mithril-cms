package server

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func newTestDistFS() fs.FS {
	return fstest.MapFS{
		"index.html":              {Data: []byte("<html>SPA</html>")},
		"assets/index-abc123.js":  {Data: []byte("console.log('app')")},
		"assets/index-abc123.css": {Data: []byte("body{}")},
		"favicon.ico":             {Data: []byte("icon")},
	}
}

func TestNewSPAHandler_NilFS_ServesPlaceholder(t *testing.T) {
	handler := newSPAHandler(false, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("expected text/html, got %q", ct)
	}
	body := rr.Body.String()
	if body == "" || len(body) < 20 {
		t.Fatal("expected placeholder HTML body")
	}
}

func TestNewSPAHandler_EmbeddedFS_ServesIndexHTML(t *testing.T) {
	handler := newSPAHandler(false, newTestDistFS())
	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if got := rr.Body.String(); got != "<html>SPA</html>" {
		t.Fatalf("expected index.html content, got %q", got)
	}
	if cc := rr.Header().Get("Cache-Control"); cc != "no-cache, no-store, must-revalidate" {
		t.Fatalf("index.html should have no-cache, got %q", cc)
	}
}

func TestNewSPAHandler_EmbeddedFS_ServesHashedAsset(t *testing.T) {
	handler := newSPAHandler(false, newTestDistFS())
	req := httptest.NewRequest(http.MethodGet, "/admin/assets/index-abc123.js", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if cc := rr.Header().Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Fatalf("hashed asset should have immutable cache, got %q", cc)
	}
}

func TestNewSPAHandler_EmbeddedFS_SPAFallback(t *testing.T) {
	handler := newSPAHandler(false, newTestDistFS())

	// Request a path that doesn't exist as a file — should serve index.html.
	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/settings", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if got := rr.Body.String(); got != "<html>SPA</html>" {
		t.Fatalf("expected index.html fallback, got %q", got)
	}
}

func TestNewSPAHandler_APIPath_Returns404JSON(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"public API", "/api/posts/123"},
		{"admin API", "/admin/api/content/posts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newSPAHandler(false, newTestDistFS())
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Fatalf("expected 404 for API path, got %d", rr.Code)
			}
			ct := rr.Header().Get("Content-Type")
			if ct != "application/json; charset=utf-8" {
				t.Fatalf("expected JSON content-type, got %q", ct)
			}
		})
	}
}

func TestNewSPAHandler_EmbeddedFS_RootPath(t *testing.T) {
	handler := newSPAHandler(false, newTestDistFS())

	// Request to "/" (not under /admin/) should also serve index.html via SPA fallback.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestIsAPIPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/api", true},               // bare /api
		{"/api/posts", true},
		{"/api/posts/123", true},
		{"/admin/api", true},         // bare /admin/api
		{"/admin/api/content/posts", true},
		{"/admin/", false},
		{"/admin/dashboard", false},
		{"/", false},
		{"/health", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isAPIPath(tt.path); got != tt.want {
				t.Fatalf("isAPIPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestNewSPAHandler_PathTraversal_ServesSPAFallback(t *testing.T) {
	handler := newSPAHandler(false, newTestDistFS())

	// Attempt path traversal — should serve index.html fallback, not file contents.
	tests := []struct {
		name string
		path string
	}{
		{"double-dot escape", "/admin/../../etc/passwd"},
		{"complex traversal", "/admin/assets/../../../secret"},
		{"direct double-dot", "/admin/.."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rr.Code)
			}
			if got := rr.Body.String(); got != "<html>SPA</html>" {
				t.Fatalf("expected index.html fallback, got %q", got)
			}
			if cc := rr.Header().Get("Cache-Control"); cc != "no-cache, no-store, must-revalidate" {
				t.Fatalf("expected no-cache, got %q", cc)
			}
			if nosniff := rr.Header().Get("X-Content-Type-Options"); nosniff != "nosniff" {
				t.Fatalf("expected nosniff header, got %q", nosniff)
			}
		})
	}
}
