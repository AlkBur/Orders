//go:build !debug

package app

import (
	"embed"
	"io/fs"
)

//go:embed templates/layout
//go:embed templates/components
//go:embed templates/pages
var showcaseTemplatesFS embed.FS

//go:embed static/**
var staticFS embed.FS

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
