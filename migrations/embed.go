// Package migrations embeds the SQL migration files so they can be used
// by the database package without requiring the files on disk at runtime.
package migrations

import "embed"

// FS contains the embedded SQL migration files.
//
//go:embed *.sql
var FS embed.FS
