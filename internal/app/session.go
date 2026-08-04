package app

import "net/http"

const SessionCookie = "orders_session"

func SetSessionCookie(w http.ResponseWriter, id string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func DeleteSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   SessionCookie,
		Path:   "/",
		MaxAge: -1,
	})
}

// RenderPageStatus рендерит форму входа с указанным HTTP-статусом.
// Используется для инфраструктурных ошибок (400/413/429), которые должны быть
// видны и в FullPage-режиме. WriteHeader вызывается до рендеринга, потому что
// ui.Render свой статус не записывает.
func (a *App) RenderPageStatus(w http.ResponseWriter, r *http.Request, mode ResponseMode, status int, data any) {
	NoCache(w)
	w.WriteHeader(status)
	a.RenderAuth(w, r, mode, "login", data)
}
