// Package main is the entrypoint for the Mithril CMS server.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GyroZepelix/mithril-cms/internal/auth"
	"github.com/GyroZepelix/mithril-cms/internal/config"
	"github.com/GyroZepelix/mithril-cms/internal/database"
	"github.com/GyroZepelix/mithril-cms/internal/schema"
	"github.com/GyroZepelix/mithril-cms/internal/server"
)

func main() {
	cfg := config.Load()

	// --- Set up structured logging ---
	logLevel := slog.LevelInfo
	if cfg.DevMode {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	slog.Info("starting Mithril CMS",
		"port", cfg.Port,
		"schema_dir", cfg.SchemaDir,
		"media_dir", cfg.MediaDir,
		"dev_mode", cfg.DevMode,
	)

	// --- Connect to database ---
	if cfg.DatabaseURL == "" {
		slog.Error("MITHRIL_DATABASE_URL is required")
		os.Exit(1)
	}

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()

	db, err := database.New(dbCtx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("database connected")

	// --- Run system table migrations ---
	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")

	// --- Load and validate schemas ---
	schemas, err := schema.LoadSchemas(cfg.SchemaDir)
	if err != nil {
		slog.Error("failed to load schemas", "error", err)
		os.Exit(1)
	}
	slog.Info("schemas loaded", "count", len(schemas))

	if err := schema.ValidateSchemas(schemas); err != nil {
		slog.Error("schema validation failed", "error", err)
		os.Exit(1)
	}
	slog.Info("schemas validated")

	// --- Apply schemas via engine ---
	engine := schema.NewEngine(db, cfg.DevMode)

	applyCtx, applyCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer applyCancel()

	if err := engine.Apply(applyCtx, schemas); err != nil {
		slog.Error("failed to apply schemas", "error", err)
		os.Exit(1)
	}
	slog.Info("schemas applied")

	// --- Set up authentication ---
	if cfg.JWTSecret == "" {
		slog.Error("MITHRIL_JWT_SECRET is required")
		os.Exit(1)
	}

	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, cfg.JWTSecret)

	// Create initial admin if configured and no admins exist yet.
	if cfg.AdminEmail != "" && cfg.AdminPassword != "" {
		adminCtx, adminCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer adminCancel()

		if err := authService.EnsureAdmin(adminCtx, cfg.AdminEmail, cfg.AdminPassword); err != nil {
			slog.Error("failed to ensure initial admin", "error", err)
			os.Exit(1)
		}
	}

	authHandler := auth.NewHandler(authService, cfg.DevMode)
	authMiddleware := auth.Middleware(cfg.JWTSecret)

	// --- Build router and start server ---
	deps := server.Dependencies{
		DB:             db,
		Engine:         engine,
		Schemas:        schemas,
		DevMode:        cfg.DevMode,
		AuthHandler:    authHandler,
		AuthMiddleware: authMiddleware,
	}

	router := server.NewRouter(deps)
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := server.New(addr, router)

	// Start server in a goroutine.
	errCh := make(chan error, 1)
	go func() {
		slog.Info("HTTP server listening", "addr", addr)
		errCh <- srv.Start()
	}()

	// --- Graceful shutdown on SIGINT/SIGTERM ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("received shutdown signal", "signal", sig.String())
	case err := <-errCh:
		if err != nil {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	slog.Info("shutting down server (30s timeout)...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("Mithril CMS stopped")
}
