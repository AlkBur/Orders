//go:build !debug

package app

import (
	"embed"
	"html/template"
)

//go:embed templates/*
var templatesFS embed.FS

func LoadTemplates() (*template.Template, error) {
	return template.ParseFS(
		templatesFS,
		"templates/layout.html",
		"templates/index.html",
	)
}
