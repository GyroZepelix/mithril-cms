//go:build tools

// Package tools tracks dependencies that are required by the project but not
// yet directly imported. These will be imported by their respective packages
// as implementation progresses. This file ensures go mod tidy does not remove
// them from go.mod.
package tools

import (
	_ "github.com/golang-migrate/migrate/v4"
)
