//go:build debug

package app

import (
	"net/http"
)

func (a *App) Render(w http.ResponseWriter, page string, data any) error {
	tmpl, err := LoadTemplates(page)
	if err != nil {
		return err
	}

	if tmpl.Lookup("layout") != nil {
		return tmpl.ExecuteTemplate(w, "layout", data)
	}

	return tmpl.Execute(w, data)
}
