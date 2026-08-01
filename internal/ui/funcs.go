package ui

import (
	"html/template"

	"Orders/internal/ui/display"
)

// Funcs returns the shared template functions of the platform.
// Every page template is parsed with these functions.
func Funcs() template.FuncMap {
	return template.FuncMap{
		"icon":         icon,
		"getFAB":       getFAB,
		"formatNumber": func(v int64) string { return display.FormatNumber(v) },
	}
}
