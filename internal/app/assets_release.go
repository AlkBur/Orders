//go:build !debug

package app

import (
	"embed"
	"html/template"
	"io/fs"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/**
var staticFS embed.FS

func LoadTemplates(page string) (*template.Template, error) {

	if page == "login" {
		return template.ParseFS(
			templatesFS,
			"templates/login.html",
		)
	}

	if page == "products" {
		return template.ParseFS(
			templatesFS,
			"templates/layout.html",
			"templates/list.html",
		)
	}

	content := "templates/" + page + ".html"

	return template.ParseFS(
		templatesFS,
		"templates/layout.html",
		content,
	)
}

func StaticFS() fs.FS {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return sub
}
