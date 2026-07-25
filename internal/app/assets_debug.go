//go:build debug

package app

import (
	"html/template"
)

func LoadTemplates() (*template.Template, error) {
	return template.ParseFiles(
		"internal/app/templates/layout.html",
		"internal/app/templates/index.html",
	)
}
