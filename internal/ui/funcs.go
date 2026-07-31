package ui

import "html/template"

// Funcs returns the shared template functions of the platform.
// Every page template is parsed with these functions.
func Funcs() template.FuncMap {
	return template.FuncMap{
		"icon":   icon,
		"getFAB": getFAB,
	}
}
