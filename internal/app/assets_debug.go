//go:build debug

package app

import (
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
)

func LoadTemplates(page string) (*template.Template, error) {
	icons := filepath.Join("internal", "app", "templates", "icons.html")

	if page == "login" {
		return template.ParseFiles(
			filepath.Join("internal", "app", "templates", "login.html"),
			icons,
		)
	}

	if page == "products" || page == "customers" || page == "users" || page == "receipts" {
		return template.ParseFiles(
			filepath.Join("internal", "app", "templates", "layout.html"),
			filepath.Join("internal", "app", "templates", "list.html"),
			icons,
		)
	}

	content := page + ".html"

	return template.ParseFiles(
		filepath.Join("internal", "app", "templates", "layout.html"),
		filepath.Join("internal", "app", "templates", content),
		icons,
	)
}

func StaticFS() fs.FS {
	return os.DirFS(filepath.Join(
		"internal",
		"app",
		"static",
	))
}

func TemplateFS() fs.FS {
	return os.DirFS(filepath.Join(
		"internal",
		"app",
		"templates",
	))
}
