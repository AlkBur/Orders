//go:build debug

package app

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// assetDir returns the absolute path of internal/app/<sub>,
// independent of the current working directory (tests run from the
// package directory, the server from the repository root).
func assetDir(sub string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), sub)
}

func StaticFS() fs.FS {
	return os.DirFS(assetDir("static"))
}

func TemplateFS() fs.FS {
	return os.DirFS(assetDir("templates"))
}
