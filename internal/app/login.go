package app

import (
	"Orders/internal/app/pages"
	"net/http"
)

func (a *App) LoginPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	page := pages.LoginPage{
		Title: "Orders",
	}

	a.Render(w, "login", page)
}

func (app *App) Login(w http.ResponseWriter, r *http.Request) {
	page := pages.LoginPage{
		Title: "Orders",
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	page.Login = r.FormValue("login")
	password := r.FormValue("password")

	user, err := app.users.FindByLogin(page.Login)
	if err != nil {
		NoCache(w)
		page.Error = "Invalid login or password"
		app.Render(w, "login", page)
		return
	}

	if user.NeedsPasswordSetup() {
		session, err := app.sessions.Create(user.ID, r.UserAgent())
		if err != nil {
			app.InternalError(w, err)
			return
		}
		SetSessionCookie(w, session.ID)
		http.Redirect(w, r, "/orders", http.StatusSeeOther)
		return
	}

	ok, err := user.VerifyPassword(password)
	if err != nil || !ok {
		NoCache(w)
		page.Error = "Invalid login or password"
		app.Render(w, "login", page)
		return
	}

	session, err := app.sessions.Create(user.ID, r.UserAgent())
	if err != nil {
		app.InternalError(w, err)
		return
	}
	SetSessionCookie(w, session.ID)
	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}
