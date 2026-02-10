package content

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/GyroZepelix/mithril-cms/internal/schema"
)

// newTestRouter creates a minimal chi router with content handler routes for testing.
// The handler uses nil service (we're testing schema lookup and public_read logic).
func newTestHandler() *Handler {
	schemas := map[string]schema.ContentType{
		"posts": {
			Name:       "posts",
			PublicRead: true,
			Fields: []schema.Field{
				{Name: "title", Type: schema.FieldTypeString},
			},
		},
		"secrets": {
			Name:       "secrets",
			PublicRead: false,
			Fields: []schema.Field{
				{Name: "data", Type: schema.FieldTypeText},
			},
		},
	}
	// We pass nil service since these tests check handler-level logic before service calls.
	return NewHandler(nil, schemas)
}

func TestHandler_LookupSchema_NotFound(t *testing.T) {
	h := newTestHandler()

	r := chi.NewRouter()
	r.Get("/api/{contentType}", h.PublicList)

	req := httptest.NewRequest(http.MethodGet, "/api/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error object in response")
	}
	if errObj["code"] != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND code, got %v", errObj["code"])
	}
}

func TestHandler_PublicList_NonPublicType(t *testing.T) {
	h := newTestHandler()

	r := chi.NewRouter()
	r.Get("/api/{contentType}", h.PublicList)

	req := httptest.NewRequest(http.MethodGet, "/api/secrets", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-public type, got %d", w.Code)
	}
}

func TestHandler_PublicGet_NonPublicType(t *testing.T) {
	h := newTestHandler()

	r := chi.NewRouter()
	r.Get("/api/{contentType}/{id}", h.PublicGet)

	// Use a valid UUID so the non-public check is reached before UUID validation.
	req := httptest.NewRequest(http.MethodGet, "/api/secrets/550e8400-e29b-41d4-a716-446655440000", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-public type, got %d", w.Code)
	}
}

func TestHandler_DecodeBody_InvalidJSON(t *testing.T) {
	h := newTestHandler()
	// We need a real service for AdminCreate, but the body decode happens before
	// the service call. We'll just test that invalid JSON returns 400.
	r := chi.NewRouter()
	r.Post("/admin/api/content/{contentType}", h.AdminCreate)

	req := httptest.NewRequest(http.MethodPost, "/admin/api/content/posts",
		strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	errObj := resp["error"].(map[string]any)
	if errObj["code"] != "INVALID_JSON" {
		t.Errorf("expected INVALID_JSON code, got %v", errObj["code"])
	}
}

func TestHandler_AdminList_InvalidQueryParams(t *testing.T) {
	// AdminList with nil service will panic when it tries to call service.List,
	// but invalid query params are caught before the service call.
	schemas := map[string]schema.ContentType{
		"posts": {
			Name:   "posts",
			Fields: []schema.Field{{Name: "title", Type: schema.FieldTypeString}},
		},
	}
	h := NewHandler(nil, schemas)

	r := chi.NewRouter()
	r.Get("/admin/api/content/{contentType}", h.AdminList)

	req := httptest.NewRequest(http.MethodGet, "/admin/api/content/posts?page=-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid page, got %d", w.Code)
	}
}

func TestConvertNumbers(t *testing.T) {
	data := map[string]any{
		"int_val":    json.Number("42"),
		"float_val":  json.Number("3.14"),
		"string_val": "hello",
		"nested": map[string]any{
			"num": json.Number("7"),
		},
	}

	convertNumbers(data)

	// Integers stay as int64.
	if v, ok := data["int_val"].(int64); !ok || v != 42 {
		t.Errorf("int_val: expected int64(42), got %T(%v)", data["int_val"], data["int_val"])
	}
	if v, ok := data["float_val"].(float64); !ok || v != 3.14 {
		t.Errorf("float_val: expected float64(3.14), got %T(%v)", data["float_val"], data["float_val"])
	}
	nested := data["nested"].(map[string]any)
	if v, ok := nested["num"].(int64); !ok || v != 7 {
		t.Errorf("nested num: expected int64(7), got %T(%v)", nested["num"], nested["num"])
	}
}

func TestHandler_AdminGet_InvalidUUID(t *testing.T) {
	h := newTestHandler()

	r := chi.NewRouter()
	r.Get("/admin/api/content/{contentType}/{id}", h.AdminGet)

	req := httptest.NewRequest(http.MethodGet, "/admin/api/content/posts/not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid UUID, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	errObj := resp["error"].(map[string]any)
	if errObj["code"] != "INVALID_ID" {
		t.Errorf("expected INVALID_ID code, got %v", errObj["code"])
	}
}

func TestHandler_PublicGet_InvalidUUID(t *testing.T) {
	h := newTestHandler()

	r := chi.NewRouter()
	r.Get("/api/{contentType}/{id}", h.PublicGet)

	req := httptest.NewRequest(http.MethodGet, "/api/posts/bad-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid UUID, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	errObj := resp["error"].(map[string]any)
	if errObj["code"] != "INVALID_ID" {
		t.Errorf("expected INVALID_ID code, got %v", errObj["code"])
	}
}
