package app

import (
	"Orders/internal/app/pages"
	"net/http"
)

func (a *App) OrdersPage(w http.ResponseWriter, r *http.Request) {
	page := pages.OrdersPage{
		Title: "Orders",
	}

	a.Render(w, "orders", page)
}
