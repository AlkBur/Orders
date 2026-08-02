package app

import (
	"errors"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"Orders/internal/app/pages"
	"Orders/internal/common"
	"Orders/internal/entity"
	"Orders/internal/ui"
	"Orders/internal/ui/display"
	"Orders/internal/users"

	"github.com/go-chi/chi/v5"
)

func (a *App) UsersPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	query := strings.TrimSpace(r.URL.Query().Get("q"))

	fields := users.Descriptor.ListFields()
	visibleFields := entity.Names(fields)

	list, err := a.users.List(r.Context(), users.ListOptions{Query: query}, visibleFields)
	if err != nil {
		a.InternalError(w, err)
		return
	}

	var columns []ui.ListColumn
	for _, f := range fields {
		columns = append(columns, ui.ListColumn{Label: f.Label})
	}

	var rows []ui.ListRow
	for _, u := range list {
		var item display.Values = u

		var cells []string
		for _, f := range fields {
			value, err := item.DisplayValue(f.GoName)
			if err != nil {
				a.InternalError(w, err)
				return
			}
			cells = append(cells, value)
		}
		rows = append(rows, ui.ListRow{
			Cells: cells,
			URL:   u.URL(),
		})
	}

	pageFS, err := fs.Sub(users.Templates(), "list")
	if err != nil {
		a.InternalError(w, err)
		return
	}

	page := pages.ListViewPage{
		Title:  "Пользователи",
		Header: pageHeader(r, "Пользователи"),
		List: ui.ListView{
			Toolbar: &ui.ToolbarData{
				Buttons: []ui.Button{
					{Style: ui.ButtonPrimary, Text: "Добавить", URL: "/users/new", Icon: "plus"},
				},
			},
			Search: &ui.SearchData{URL: "/users", Placeholder: "Поиск пользователей...", Query: query},
			List: ui.ListData{
				Columns:    columns,
				Rows:       rows,
				RenderMode: ui.RenderComfortable,
				Preset:     ui.ListDefault,
			},
		},
		NewURL: "/users/new",
	}

	a.renderListView(w, r, TemplateFS(), pageFS, page)
}

func userIDFromURL(r *http.Request) int64 {
	idStr := chi.URLParam(r, "id")
	if idStr == "" || idStr == "new" {
		return 0
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0
	}
	return id
}

func (a *App) UserCard(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	id := userIDFromURL(r)

	var user *users.User
	if id == 0 {
		user = a.users.New()
	} else {
		var err error
		user, err = a.users.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, users.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			a.InternalError(w, err)
			return
		}
	}

	title := user.Login
	if title == "" {
		title = "Новый пользователь"
	}

	page := pages.UserCardPage{
		Title: title,
		User:  user,
	}

	a.Render(w, "user_card", page)
}

func (a *App) UserSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.BadRequest(w, "Invalid request")
		return
	}

	id := userIDFromURL(r)

	login := r.FormValue("login")
	normalized := users.NormalizeLogin(login)
	if a.identity.IsLoginTaken(normalized, id) {
		a.BadRequest(w, "Login already taken")
		return
	}

	if id != 0 && r.FormValue("is_admin") != "on" && a.identity.IsLastAdministrator(id) {
		a.BadRequest(w, users.ErrLastAdministrator.Error())
		return
	}

	user := &users.User{
		ID:      id,
		UUID:    r.FormValue("uuid"),
		Login:   login,
		Email:   r.FormValue("email"),
		IsAdmin: r.FormValue("is_admin") == "on",
	}

	if id == 0 && user.UUID == "" {
		uuid, err := common.GenerateUUID()
		if err != nil {
			a.InternalError(w, err)
			return
		}
		user.UUID = uuid
	}

	if err := a.users.Save(r.Context(), user); err != nil {
		a.InternalError(w, err)
		return
	}

	if id == 0 {
		a.identity.Add(user)
	} else {
		a.identity.Update(user)
	}

	http.Redirect(w, r, "/users/"+strconv.FormatInt(user.ID, 10), http.StatusSeeOther)
}

func (a *App) UserDelete(w http.ResponseWriter, r *http.Request) {
	id := userIDFromURL(r)

	if a.identity.IsLastAdministrator(id) {
		a.BadRequest(w, users.ErrLastAdministrator.Error())
		return
	}

	if err := a.users.DeleteByID(r.Context(), id); err != nil {
		a.InternalError(w, err)
		return
	}

	a.identity.Remove(id)

	http.Redirect(w, r, "/users", http.StatusSeeOther)
}
