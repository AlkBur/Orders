package pages

import "Orders/internal/ui"

type SetPasswordPage struct {
	Title  string
	Login  string
	Fields []ui.Field
	Alert  *ui.AlertData
}
