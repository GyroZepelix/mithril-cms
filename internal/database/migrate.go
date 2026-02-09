package database

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	// Register the postgres database driver for golang-migrate.
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/GyroZepelix/mithril-cms/migrations"
)

// RunMigrations applies all pending UP migrations embedded in the binary.
// The databaseURL must be a valid PostgreSQL connection string
// (e.g. postgres://user:pass@host:5432/dbname?sslmode=disable).
func RunMigrations(databaseURL string) (retErr error) {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("creating migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, databaseURL)
	if err != nil {
		return fmt.Errorf("creating migrate instance: %w", err)
	}
	defer func() {
		sourceErr, dbErr := m.Close()
		if retErr == nil {
			if sourceErr != nil {
				retErr = fmt.Errorf("closing migration source: %w", sourceErr)
			} else if dbErr != nil {
				retErr = fmt.Errorf("closing migration database: %w", dbErr)
			}
		}
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}
