// Package main is the entrypoint for the Mithril CMS server.
//
// Usage:
//
//	mithril              — start the HTTP server (default)
//	mithril serve        — start the HTTP server (explicit)
//	mithril schema diff  — load schemas, diff against DB, print changes, exit
//	mithril schema apply — apply safe schema changes, exit
//	mithril schema apply --force — apply ALL schema changes including breaking, exit
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GyroZepelix/mithril-cms/internal/audit"
	"github.com/GyroZepelix/mithril-cms/internal/auth"
	"github.com/GyroZepelix/mithril-cms/internal/config"
	"github.com/GyroZepelix/mithril-cms/internal/content"
	"github.com/GyroZepelix/mithril-cms/internal/contenttypes"
	"github.com/GyroZepelix/mithril-cms/internal/database"
	"github.com/GyroZepelix/mithril-cms/internal/media"
	"github.com/GyroZepelix/mithril-cms/internal/schema"
	"github.com/GyroZepelix/mithril-cms/internal/schemaapi"
	"github.com/GyroZepelix/mithril-cms/internal/server"
)

func main() {
	// Parse subcommand from os.Args.
	cmd := parseCommand(os.Args[1:])

	switch cmd {
	case cmdServe:
		runServe()
	case cmdSchemaDiff:
		runSchemaDiff()
	case cmdSchemaApply:
		runSchemaApply(false)
	case cmdSchemaApplyForce:
		runSchemaApply(true)
	default:
		printUsage()
		os.Exit(1)
	}
}

// command represents the parsed CLI subcommand.
type command int

const (
	cmdServe           command = iota
	cmdSchemaDiff
	cmdSchemaApply
	cmdSchemaApplyForce
	cmdUnknown
)

// parseCommand determines which subcommand was requested from CLI arguments.
func parseCommand(args []string) command {
	if len(args) == 0 {
		return cmdServe
	}

	switch args[0] {
	case "serve":
		return cmdServe
	case "schema":
		if len(args) < 2 {
			return cmdUnknown
		}
		switch args[1] {
		case "diff":
			return cmdSchemaDiff
		case "apply":
			if len(args) >= 3 && args[2] == "--force" {
				return cmdSchemaApplyForce
			}
			return cmdSchemaApply
		default:
			return cmdUnknown
		}
	default:
		return cmdUnknown
	}
}

// printUsage prints CLI usage information to stderr.
func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: mithril [command]

Commands:
  serve                  Start the HTTP server (default)
  schema diff            Show pending schema changes
  schema apply           Apply safe schema changes
  schema apply --force   Apply ALL schema changes (including breaking)`)
}

// initBase performs common initialization steps shared by all commands:
// config loading, logging setup, DB connection, and migrations.
func initBase() (*config.Config, *database.DB) {
	cfg := config.Load()

	logLevel := slog.LevelInfo
	if cfg.DevMode {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

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

	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		db.Close()
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database connected and migrations applied")

	return cfg, db
}

// runSchemaDiff loads schemas, diffs against DB state, and prints the results
// to stdout. Exits with code 0 if no changes, 1 on error, 2 if breaking
// changes are detected.
func runSchemaDiff() {
	cfg, db := initBase()
	defer db.Close()

	engine := schema.NewEngine(db, cfg.DevMode)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use Refresh in a read-only way by examining the result without force.
	// However, Refresh actually applies safe changes, so we need to do the
	// diff manually to avoid side effects.
	schemas, err := schema.LoadSchemas(cfg.SchemaDir)
	if err != nil {
		slog.Error("failed to load schemas", "error", err)
		os.Exit(1)
	}

	if err := schema.ValidateSchemas(schemas); err != nil {
		slog.Error("schema validation failed", "error", err)
		os.Exit(1)
	}

	// Use the engine to get the diff without applying.
	changes := diffAllSchemas(ctx, engine, schemas)

	if len(changes) == 0 {
		fmt.Println("All schemas are up to date. No changes detected.")
		return
	}

	// Group changes by table for display.
	breakingCount := printChanges(changes)

	fmt.Println()
	if breakingCount > 0 {
		fmt.Printf("%d breaking change(s) detected. Use --force to apply.\n", breakingCount)
		os.Exit(2)
	}
	fmt.Println("No breaking changes detected.")
}

// diffAllSchemas computes the diff between loaded schemas and DB state without
// applying anything. This is used by the CLI diff command.
func diffAllSchemas(ctx context.Context, engine *schema.Engine, schemas []schema.ContentType) []schema.Change {
	var allChanges []schema.Change
	for _, loaded := range schemas {
		existing, err := engine.GetExistingContentType(ctx, loaded.Name)
		if err != nil {
			slog.Error("failed to query existing content type", "name", loaded.Name, "error", err)
			os.Exit(1)
		}

		if existing != nil && existing.SchemaHash == loaded.SchemaHash {
			continue
		}

		changes := schema.DiffSchema(loaded, existing)
		allChanges = append(allChanges, changes...)
	}
	return allChanges
}

// printChanges groups changes by table and prints them to stdout.
// Returns the count of breaking changes.
func printChanges(changes []schema.Change) int {
	// Group by table.
	type tableGroup struct {
		table   string
		changes []schema.Change
	}

	seen := make(map[string]int) // table -> index in groups
	var groups []tableGroup

	for _, c := range changes {
		idx, ok := seen[c.Table]
		if !ok {
			idx = len(groups)
			seen[c.Table] = idx
			groups = append(groups, tableGroup{table: c.Table})
		}
		groups[idx].changes = append(groups[idx].changes, c)
	}

	breakingCount := 0
	for _, g := range groups {
		fmt.Printf("Content type: %s\n", g.table)
		for _, c := range g.changes {
			label := "SAFE"
			if !c.Safe {
				label = "BREAKING"
				breakingCount++
			}
			fmt.Printf("  [%s] %s\n", label, c.Detail)
		}
		fmt.Println()
	}

	return breakingCount
}

// runSchemaApply loads schemas, applies changes (safe only, or all if force),
// and prints the results to stdout.
func runSchemaApply(force bool) {
	cfg, db := initBase()
	defer db.Close()

	engine := schema.NewEngine(db, cfg.DevMode)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, _, err := engine.Refresh(ctx, cfg.SchemaDir, force)
	if err != nil {
		slog.Error("schema apply failed", "error", err)
		os.Exit(1)
	}

	if len(result.Applied) == 0 && len(result.Breaking) == 0 {
		fmt.Println("All schemas are up to date. No changes to apply.")
		return
	}

	if len(result.Applied) > 0 {
		fmt.Printf("Applied %d change(s):\n", len(result.Applied))
		for _, c := range result.Applied {
			label := "SAFE"
			if !c.Safe {
				label = "BREAKING"
			}
			fmt.Printf("  [%s] %s\n", label, c.Detail)
		}
	}

	if len(result.NewTypes) > 0 {
		fmt.Printf("\nNew content types: %v\n", result.NewTypes)
	}
	if len(result.UpdatedTypes) > 0 {
		fmt.Printf("Updated content types: %v\n", result.UpdatedTypes)
	}

	if len(result.Breaking) > 0 {
		fmt.Printf("\n%d breaking change(s) NOT applied:\n", len(result.Breaking))
		for _, c := range result.Breaking {
			fmt.Printf("  [BREAKING] %s\n", c.Detail)
		}
		fmt.Println("\nUse --force to apply breaking changes.")
		os.Exit(2)
	}

	fmt.Println("\nSchema apply completed successfully.")
}

// runServe starts the full HTTP server with all handlers wired up.
func runServe() {
	cfg, db := initBase()
	defer db.Close()

	slog.Info("starting Mithril CMS",
		"port", cfg.Port,
		"schema_dir", cfg.SchemaDir,
		"media_dir", cfg.MediaDir,
		"dev_mode", cfg.DevMode,
	)

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

	// --- Set up audit logging ---
	auditRepo := audit.NewRepository(db)
	auditService := audit.NewService(auditRepo)
	auditService.Start()
	auditHandler := audit.NewHandler(auditService)
	slog.Info("audit logging started")

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

	authHandler := auth.NewHandler(authService, auditService, cfg.DevMode)
	authMiddleware := auth.Middleware(cfg.JWTSecret)

	// --- Set up content CRUD ---
	schemaMap := make(map[string]schema.ContentType, len(schemas))
	for _, ct := range schemas {
		schemaMap[ct.Name] = ct
	}

	contentRepo := content.NewRepository(db)
	contentService := content.NewService(contentRepo, schemaMap, auditService)
	contentHandler := content.NewHandler(contentService, schemaMap)

	// --- Set up content type introspection ---
	contentTypeHandler := contenttypes.NewHandler(db.Pool(), schemaMap)

	// --- Set up media ---
	mediaStorage, err := media.NewLocalStorage(cfg.MediaDir)
	if err != nil {
		slog.Error("failed to initialize media storage", "error", err)
		os.Exit(1)
	}
	slog.Info("media storage initialized", "dir", cfg.MediaDir)

	mediaRepo := media.NewRepository(db)
	mediaService := media.NewService(mediaRepo, mediaStorage, auditService)
	mediaHandler := media.NewHandler(mediaService, cfg.DevMode)

	// --- Set up schema handler ---
	// The onRefresh callback updates the content service and handler schema
	// maps when schemas are refreshed at runtime via the admin API.
	schemaHandler := schemaapi.NewHandler(engine, cfg.SchemaDir, schemaMap, auditService, func(newSchemas []schema.ContentType) {
		newMap := make(map[string]schema.ContentType, len(newSchemas))
		for _, ct := range newSchemas {
			newMap[ct.Name] = ct
		}
		contentService.UpdateSchemas(newMap)
		contentHandler.UpdateSchemas(newMap)
		contentTypeHandler.UpdateSchemas(newMap)
	})

	// --- Build router and start server ---
	deps := server.Dependencies{
		DB:             db,
		Engine:         engine,
		Schemas:        schemas,
		DevMode:        cfg.DevMode,
		AuthHandler:    authHandler,
		AuthMiddleware: authMiddleware,
		ContentHandler: contentHandler,
		MediaHandler:   mediaHandler,
		AuditHandler:       auditHandler,
		SchemaHandler:      schemaHandler,
		ContentTypeHandler: contentTypeHandler,
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

	// Drain remaining audit events before closing the database.
	slog.Info("draining audit events...")
	auditService.Shutdown(shutdownCtx)

	slog.Info("Mithril CMS stopped")
}
