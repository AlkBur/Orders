//go:build debug

package organizations

import (
	"io/fs"
	"os"
	"path/filepath"
)

// Templates returns the domain templates from disk so they reload
// on every request in debug builds.
func Templates() fs.FS {
	return os.DirFS(filepath.Join("internal", "organizations", "templates"))
}
