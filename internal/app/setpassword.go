package app

import (
	"net/http"

	"Orders/internal/app/pages"
	"Orders/internal/ui"
)

func (a *App) SetPasswordPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	user := CurrentUser(r)
	if user.ID == 0 || !user.NeedsPasswordSetup() {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	a.RenderAuth(w, r, ResponseModeFromRequest(r), "set_password", a.setPasswordPageData(user.Login, nil))
}

func (a *App) SetPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	mode := ResponseModeFromRequest(r)

	identity := CurrentUser(r)
	if identity.ID == 0 || !identity.NeedsPasswordSetup() {
		a.Redirect(w, r, mode, "/")
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	password := r.FormValue("password")
	confirm := r.FormValue("confirm")

	var msgs []string
	if password == "" {
		msgs = append(msgs, "Новый пароль обязателен.")
	}
	if confirm == "" {
		msgs = append(msgs, "Подтверждение пароля обязательно.")
	}
	if password != "" && confirm != "" && password != confirm {
		msgs = append(msgs, "Пароли не совпадают.")
	}

	if len(msgs) > 0 {
		NoCache(w)
		a.RenderAuth(w, r, mode, "set_password", a.setPasswordPageData(identity.Login, &ui.AlertData{
			Type:     ui.AlertError,
			Messages: msgs,
		}))
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

	a.Redirect(w, r, mode, "/")
}

func (a *App) setPasswordPageData(login string, alert *ui.AlertData) pages.SetPasswordPage {
	return pages.SetPasswordPage{
		Title: "Установка пароля",
		Login: login,
		Fields: []ui.Field{
			{Name: "password", Label: "Новый пароль", Type: ui.FieldPassword},
			{Name: "confirm", Label: "Повторите пароль", Type: ui.FieldPassword},
		},
		Alert: alert,
	}
}
