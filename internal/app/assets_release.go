//go:build !debug

package app

import (
	"embed"
	"html/template"
	"io/fs"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed templates/layout
//go:embed templates/components
//go:embed templates/pages
var showcaseTemplatesFS embed.FS

//go:embed static/**
var staticFS embed.FS

func LoadTemplates(page string) (*template.Template, error) {
	if page == "login" {
		return template.ParseFS(
			templatesFS,
			"templates/login.html",
			"templates/icons.html",
		)
	}

	if page == "products" || page == "organizations" || page == "customers" || page == "users" || page == "receipts" {
		return template.ParseFS(
			templatesFS,
			"templates/layout.html",
			"templates/list.html",
			"templates/icons.html",
		)
	}

	content := "templates/" + page + ".html"

	return template.ParseFS(
		templatesFS,
		"templates/layout.html",
		content,
		"templates/icons.html",
	)
}

func StaticFS() fs.FS {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return sub
}

func TemplateFS() fs.FS {
	sub, err := fs.Sub(showcaseTemplatesFS, "templates")
	if err != nil {
		panic(err)
	}
	return sub
}
