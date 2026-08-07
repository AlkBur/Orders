package app

import (
	"io/fs"
	"net/http"

	"Orders/internal/app/pages"
	"Orders/internal/ui"
)

// renderListView рендерит страницу-список: полная страница или фрагмент.
// Фрагмент возвращает только список (#list) — его подменяет HTMX-поиск.
func (a *App) renderListView(w http.ResponseWriter, r *http.Request, baseFS, pageFS fs.FS, page pages.ListViewPage) {
	if ResponseModeFromRequest(r) == Fragment {
		if err := ui.Render(w, baseFS, pageFS, "list", page.List.List); err != nil {
			a.InternalError(w, r, err)
		}
		return
	}
	if err := ui.RenderPage(w, baseFS, pageFS, page); err != nil {
		a.InternalError(w, r, err)
	}
}
