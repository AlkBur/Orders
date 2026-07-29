package app

import (
	"Orders/internal/app/pages"
	"net/http"
)

func (a *App) SetPasswordPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	user := CurrentUser(r)
	if user.ID == 0 || !user.NeedsPasswordSetup() {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	p := pages.SetPasswordPage{
		Title: "Orders",
		Login: user.Login,
	}

	session := CurrentSession(r)
	if session != nil && session.Flash != nil {
		p.Error = session.Flash.Message
		session.ClearFlash()
		a.sessions.Save(session)
	}

	a.Render(w, "set-password", p)
}

func (a *App) SetPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	identity := CurrentUser(r)
	if identity.ID == 0 || !identity.NeedsPasswordSetup() {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	password := r.FormValue("password")
	confirm := r.FormValue("confirm")

	if password == "" || password != confirm {
		session := CurrentSession(r)
		if session != nil {
			session.SetFlash("error", "Passwords do not match")
			a.sessions.Save(session)
		}
		http.Redirect(w, r, "/set-password", http.StatusSeeOther)
		return
	}

	user, err := a.users.GetByID(r.Context(), identity.ID)
	if err != nil {
		a.InternalError(w, err)
		return
	}

	if err := user.SetPassword(password); err != nil {
		a.InternalError(w, err)
		return
	}

	if err := a.users.Update(user); err != nil {
		a.InternalError(w, err)
		return
	}

	a.identity.Update(user)

	if session := CurrentSession(r); session != nil {
		a.sessions.Delete(session.ID)
	}
	DeleteSessionCookie(w)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
