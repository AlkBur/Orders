package ui

import (
	"html/template"
	"io/fs"
	"net/http"
)

// RenderPage renders a full page.
//
// baseFS must contain the shared platform templates:
//
//	layout/*.html
//	components/*.html
//
// pageFS must contain a single page.html that defines "page_content".
// The page template is located next to its caller (a domain package or
// an app section), never in the shared template set.
func RenderPage(w http.ResponseWriter, baseFS, pageFS fs.FS, data any) error {
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
	return tmpl.ExecuteTemplate(w, "base", data)
}
