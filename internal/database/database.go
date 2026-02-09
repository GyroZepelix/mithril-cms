// Package database provides PostgreSQL connection pool management for Mithril CMS.
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgx connection pool and provides convenience methods
// for health checks and pool access.
type DB struct {
	pool *pgxpool.Pool
}

// New creates a new DB with a pgxpool connection pool configured from the
// given database URL. The context controls the timeout for the initial
// connection attempt.
func New(ctx context.Context, databaseURL string) (*DB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Close releases all connections in the pool. It should be called when
// the application shuts down.
func (db *DB) Close() {
	db.pool.Close()
}

// Health pings the database to verify the connection is alive.
func (db *DB) Health(ctx context.Context) error {
	if err := db.pool.Ping(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}
	return nil
}

// Pool returns the underlying pgxpool.Pool for direct query access.
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}
