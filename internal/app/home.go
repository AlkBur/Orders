package app

import "net/http"

// Home is the single application entry point.
// It never decides where to send the user itself.
// All routing decisions are delegated to LandingURL().
func (a *App) Home(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	session := CurrentSession(r)
	if session == nil || session.UserID == nil {
		http.Redirect(w, r, RouteLogin, http.StatusSeeOther)
		return
	}

	user, ok := a.identity.GetByID(*session.UserID)
	if !ok {
		a.sessions.Delete(session.ID)
		DeleteSessionCookie(w)
		http.Redirect(w, r, RouteLogin, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, LandingURL(user), http.StatusSeeOther)
}
