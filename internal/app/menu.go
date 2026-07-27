package app

import (
	"net/http"

	"Orders/internal/app/pages"
	"Orders/internal/ui"
)

var mainMenuItems = []ui.MenuItem{
	{Text: "Товарные чеки", Href: "/receipts"},
	{Text: "Организации", Href: "/organizations", Access: ui.Access{Admin: true}},
	{Text: "Контрагенты", Href: "/counterparties", Access: ui.Access{Admin: true}},
	{Text: "Товары", Href: "/products", Access: ui.Access{Admin: true}},
	{Text: "Пользователи", Href: "/users", Access: ui.Access{Admin: true}},
}

func (a *App) MenuPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)
	user := CurrentUser(r)
	var visible []ui.MenuItem
	for _, item := range mainMenuItems {
		// Visibility filtering is performed before rendering.
		if item.Access.Admin && !user.IsAdmin {
			continue
		}
		visible = append(visible, item)
	}
	a.Render(w, "menu", pages.MenuPage{
		Title: "Главное меню",
		Items: visible,
	})
}
