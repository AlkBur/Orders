package pages

import "Orders/internal/ui"

type LoginPage struct {
	Title  string
	Fields []ui.Field
	Alert  *ui.AlertData
}
