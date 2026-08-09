package app

import (
	"io/fs"
	"net/http"

	"Orders/internal/ui"
)

// RenderAuth рендерит страницу авторизации: полный Auth Layout для обычного
// запроса или только фрагмент для Fragment-режима. Имя фрагмент-шаблона
// задаёт сам обработчик: login использует "login_card", set_password —
// "page_content" (там карточка и есть весь контент страницы).
func (a *App) RenderAuth(w http.ResponseWriter, r *http.Request, mode ResponseMode, pageDir, fragment string, data any) {
	pageFS, err := fs.Sub(TemplateFS(), "pages/"+pageDir)
	if err != nil {
		a.InternalError(w, r, err)
		return
	}
	name := "auth"
	if mode == Fragment {
		name = fragment
	}
	if err := ui.Render(w, TemplateFS(), pageFS, a.basePath(), name, data); err != nil {
		a.InternalError(w, r, err)
	}
}

// Redirect перенаправляет пользователя: HX-Redirect для Fragment-запроса,
// обычный 303 See Other — для полностраничного.
func (a *App) Redirect(w http.ResponseWriter, r *http.Request, mode ResponseMode, url string) {
	if mode == Fragment {
		w.Header().Set("HX-Redirect", url)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}
