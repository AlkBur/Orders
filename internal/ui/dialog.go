package ui

import "html/template"

type DialogData struct {
	ID      string
	Title   string
	Content template.HTML
}
