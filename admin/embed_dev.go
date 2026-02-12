//go:build !embed_admin

// Package admin provides access to the embedded React admin SPA built assets.
// Without the "embed_admin" build tag, no assets are embedded and DistFS
// returns nil. The server falls back to a placeholder page or the Vite dev
// proxy.
package admin

import "io/fs"

// DistFS returns nil when the admin SPA is not embedded (no embed_admin tag).
func DistFS() fs.FS {
	return nil
}
