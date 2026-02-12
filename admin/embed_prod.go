//go:build embed_admin

// Package admin provides access to the embedded React admin SPA built assets.
// When built with the "embed_admin" build tag, the admin/dist/ directory is
// embedded into the binary. Use DistFS() to obtain the filesystem.
package admin

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// DistFS returns the embedded admin SPA filesystem rooted at the dist/
// directory contents, or nil if the SPA was not embedded (missing build tag).
func DistFS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		// This should never happen since "dist" is a compile-time embed path.
		panic("admin: failed to create sub-filesystem: " + err.Error())
	}
	return sub
}
