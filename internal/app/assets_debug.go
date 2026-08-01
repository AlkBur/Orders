//go:build debug

package app

import (
	"html/template"
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

func LoadTemplates(page string) (*template.Template, error) {
	tmplDir := assetDir("templates")
	icons := filepath.Join(tmplDir, "icons.html")

	if page == "products" || page == "users" || page == "receipts" {
		return template.ParseFiles(
			filepath.Join(tmplDir, "layout.html"),
			filepath.Join(tmplDir, "list.html"),
			icons,
		)
	}

	content := page + ".html"

	return template.ParseFiles(
		filepath.Join(tmplDir, "layout.html"),
		filepath.Join(tmplDir, content),
		icons,
	)
}

func StaticFS() fs.FS {
	return os.DirFS(assetDir("static"))
}

func TemplateFS() fs.FS {
	return os.DirFS(assetDir("templates"))
}
