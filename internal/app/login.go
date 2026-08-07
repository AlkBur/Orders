package app

import (
	"net/http"

	"Orders/internal/app/pages"
	"Orders/internal/ui"
	"Orders/internal/users"
)

func (a *App) LoginPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	session := CurrentSession(r)
	if session != nil && session.UserID != nil {
		if identity, ok := a.identity.GetByID(*session.UserID); ok {
			http.Redirect(w, r, LandingURL(identity), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, RouteLogin, http.StatusSeeOther)
		return
	}

	a.RenderAuth(w, r, ResponseModeFromRequest(r), "login", a.loginPageData("", nil))
}

func (a *App) Login(w http.ResponseWriter, r *http.Request) {
	mode := ResponseModeFromRequest(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	login := r.FormValue("login")
	password := r.FormValue("password")

	var msgs []string
	if login == "" {
		msgs = append(msgs, "Пользователь обязателен.")
	}
	if password == "" {
		msgs = append(msgs, "Пароль обязателен.")
	}
	if len(msgs) > 0 {
		NoCache(w)
		a.RenderAuth(w, r, mode, "login", a.loginPageData(login, &ui.AlertData{
			Type:     ui.AlertError,
			Messages: msgs,
		}))
		return
	}

	identity, ok := a.identity.GetByLogin(login)
	if !ok {
		NoCache(w)
		a.RenderAuth(w, r, mode, "login", a.loginPageData(login, &ui.AlertData{
			Type:     ui.AlertError,
			Messages: []string{"Неверный логин или пароль."},
		}))
		return
	}

	var authenticated bool

	if identity.PasswordHash != "" {
		ok, err := users.VerifyPassword(password, identity.PasswordHash)
		authenticated = (err == nil && ok)
	} else {
		authenticated = users.VerifyBootstrapPassword(password, a.config.Auth.InitialPassword)
	}

	if !authenticated {
		NoCache(w)
		a.RenderAuth(w, r, mode, "login", a.loginPageData(login, &ui.AlertData{
			Type:     ui.AlertError,
			Messages: []string{"Неверный логин или пароль."},
		}))
		return
	}

	session, err := a.sessions.Create(identity.ID, r.UserAgent())
	if err != nil {
		a.InternalError(w, err)
		return
	}
	SetSessionCookie(w, session.ID)

	a.Redirect(w, r, mode, LandingURL(identity))
}

func (a *App) loginPageData(login string, alert *ui.AlertData) pages.LoginPage {
	return pages.LoginPage{
		Title: "Вход",
		Fields: []ui.Field{
			{Name: "login", Label: "Пользователь", Type: ui.FieldText, Value: login, Autocomplete: "username", Autofocus: true, Icon: "user"},
			{Name: "password", Label: "Пароль", Type: ui.FieldPassword, Autocomplete: "current-password", Icon: "lock"},
		},
		Alert: alert,
	}
}
