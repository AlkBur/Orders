package app

import "net/http"

func (a *App) Logout(w http.ResponseWriter, r *http.Request) {

	DeleteSession(w)

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
