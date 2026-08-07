package app

import "net/http"

func (a *App) Logout(w http.ResponseWriter, r *http.Request) {
	session := CurrentSession(r)
	if session != nil {
		a.sessions.Delete(session.ID)
	}

	DeleteSessionCookie(w)
	http.Redirect(w, r, RouteHome, http.StatusSeeOther)
}
