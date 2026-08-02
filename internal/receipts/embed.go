//go:build !debug

package receipts

import (
	"embed"
	"io/fs"
)

//go:embed templates/*
var templatesFS embed.FS

// Templates returns the domain templates (templates/list/page.html).
// In release builds they are embedded.
func Templates() fs.FS {
	sub, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		panic(err)
	}
	return sub
}
