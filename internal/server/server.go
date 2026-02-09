package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// Server wraps an http.Server with graceful shutdown support.
type Server struct {
	httpServer *http.Server
	router     chi.Router
}

// New creates a new Server that listens on the given address and routes
// requests through the provided chi.Router.
func New(addr string, router chi.Router) *Server {
	return &Server{
		router: router,
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           router,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
		},
	}
}

// Start begins listening and serving HTTP requests. It blocks until the
// server is shut down. On normal shutdown it returns nil; callers should
// check for http.ErrServerClosed and treat it as a clean exit.
func (s *Server) Start() error {
	err := s.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Shutdown gracefully shuts down the server without interrupting active
// connections. The provided context controls the timeout for outstanding
// requests to complete.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
