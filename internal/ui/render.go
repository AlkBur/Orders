package ui

import (
	"html/template"
	"io/fs"
	"net/http"
)

// Render renders any named template from the platform template set.
//
// baseFS must contain the shared platform templates:
//
//	layout/*.html
//	components/*.html
//
// pageFS must contain a single page.html. name selects the template to
// execute (e.g. "base", "auth", "page_content", "dialog", ...).
//
// Render is a general mechanism for full pages and fragments alike.
// HTMX is just one consumer of fragment rendering; it does not define
// the capability.
func Render(w http.ResponseWriter, baseFS, pageFS fs.FS, name string, data any) error {
	tmpl, err := template.New("").Funcs(Funcs()).ParseFS(baseFS,
		"layout/*.html",
		"components/*.html",
	)
	if err != nil {
		return err
	}
	if tmpl, err = tmpl.ParseFS(pageFS, "page.html"); err != nil {
		return err
	}
	return tmpl.ExecuteTemplate(w, name, data)
}

// RenderPage renders a full application page using the base layout.
func RenderPage(w http.ResponseWriter, baseFS, pageFS fs.FS, data any) error {
	return Render(w, baseFS, pageFS, "base", data)
}

// RenderAuthPage renders a full authentication page using the auth layout.
func RenderAuthPage(w http.ResponseWriter, baseFS, pageFS fs.FS, data any) error {
	return Render(w, baseFS, pageFS, "auth", data)
}
