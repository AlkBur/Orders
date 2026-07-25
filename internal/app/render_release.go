//go:build !debug

package app

import (
	"net/http"
)

func (a *App) Render(
	w http.ResponseWriter,
	page string,
	data any,
) error {

	tmpl, ok := a.templates[page]
	if !ok {
		return ErrInvalidTamplate
	}

	return tmpl.Execute(w, data)
}
