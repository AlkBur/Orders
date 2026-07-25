package app

import (
	"Orders/internal/app/pages"
	"net/http"
)

func (a *App) LoginPage(w http.ResponseWriter, r *http.Request) {
	page := pages.LoginPage{
		Title: "Orders",
	}

	a.Render(w, "login", page)
}

func (app *App) Login(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	login := r.FormValue("login")
	password := r.FormValue("password")

	user, err := app.users.Authenticate(login, password)
	if err != nil {
		http.Error(w, "Invalid login or password", http.StatusUnauthorized)
		return
	}

	CreateSession(w, app.config.Secret, Session{
		UserID: user.ID,
	})

	// Первоначальная установка пароля администратора
	if user.IsAdmin && !user.HasPassword() {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}
