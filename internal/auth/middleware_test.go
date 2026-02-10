package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware_MissingHeader(t *testing.T) {
	mw := Middleware(testSecret)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestMiddleware_InvalidFormat(t *testing.T) {
	mw := Middleware(testSecret)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestMiddleware_InvalidToken(t *testing.T) {
	mw := Middleware(testSecret)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-jwt")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestMiddleware_ValidToken(t *testing.T) {
	adminID := "550e8400-e29b-41d4-a716-446655440000"
	email := "admin@example.com"

	token, err := CreateAccessToken(adminID, email, testSecret)
	if err != nil {
		t.Fatalf("CreateAccessToken: %v", err)
	}

	var gotAdminID, gotEmail string
	mw := Middleware(testSecret)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAdminID = AdminIDFromContext(r.Context())
		gotEmail = EmailFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if gotAdminID != adminID {
		t.Errorf("AdminIDFromContext = %q, want %q", gotAdminID, adminID)
	}
	if gotEmail != email {
		t.Errorf("EmailFromContext = %q, want %q", gotEmail, email)
	}
}

func TestAdminIDFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	if got := AdminIDFromContext(ctx); got != "" {
		t.Errorf("AdminIDFromContext on empty context = %q, want empty", got)
	}
}

func TestEmailFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	if got := EmailFromContext(ctx); got != "" {
		t.Errorf("EmailFromContext on empty context = %q, want empty", got)
	}
}
