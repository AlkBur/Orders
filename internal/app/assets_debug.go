//go:build debug

package app

import (
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
)

func LoadTemplates(page string) (*template.Template, error) {

	if page == "login" {
		return template.ParseFiles(
			filepath.Join("internal", "app", "templates", "login.html"),
		)
	}

	if page == "products" {
		return template.ParseFiles(
			filepath.Join("internal", "app", "templates", "layout.html"),
			filepath.Join("internal", "app", "templates", "list.html"),
		)
	}

	content := page + ".html"

	return template.ParseFiles(
		filepath.Join("internal", "app", "templates", "layout.html"),
		filepath.Join("internal", "app", "templates", content),
	)
}

func StaticFS() fs.FS {
	return os.DirFS(filepath.Join(
		"internal",
		"app",
		"static",
	))
}
