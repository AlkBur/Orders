package app

import (
	"Orders/internal/app/pages"
	"Orders/internal/users"
	"net/http"
)

func (a *App) LoginPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	session := CurrentSession(r)
	if session != nil && session.UserID != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

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

	identity, ok := app.identity.GetByLogin(page.Login)
	if !ok {
		NoCache(w)
		page.Error = "Invalid login or password"
		app.Render(w, "login", page)
		return
	}

	var authenticated bool

	if identity.PasswordHash != "" {
		ok, err := users.VerifyPassword(password, identity.PasswordHash)
		authenticated = (err == nil && ok)
	} else {
		authenticated = users.VerifyBootstrapPassword(password, app.config.Auth.InitialPassword)
	}

	if !authenticated {
		NoCache(w)
		page.Error = "Invalid login or password"
		app.Render(w, "login", page)
		return
	}

	session, err := app.sessions.Create(identity.ID, r.UserAgent())
	if err != nil {
		app.InternalError(w, err)
		return
	}
	SetSessionCookie(w, session.ID)

	if identity.NeedsPasswordSetup() {
		http.Redirect(w, r, "/set-password", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
