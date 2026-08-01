//go:build debug

package customers

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// Templates returns the domain templates from disk so they reload
// on every request in debug builds. The path is resolved from the
// source file, so it works regardless of the working directory.
func Templates() fs.FS {
	_, file, _, _ := runtime.Caller(0)
	return os.DirFS(filepath.Join(filepath.Dir(file), "templates"))
}
