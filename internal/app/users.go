package app

import (
	"database/sql"
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
		a.InternalError(w, r, err)
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
				a.InternalError(w, r, err)
				return
			}
			cells = append(cells, value)
		}
		rows = append(rows, ui.ListRow{
			Cells: cells,
			URL:   a.URL(u.URL()),
		})
	}

	pageFS, err := fs.Sub(users.Templates(), "list")
	if err != nil {
		a.InternalError(w, r, err)
		return
	}

	page := pages.ListViewPage{
		Title:  "Пользователи",
		Header: a.pageHeader(r, "Пользователи"),
		List: ui.ListView{
			Toolbar: &ui.ToolbarData{
				Buttons: []ui.Button{
					{Style: ui.ButtonPrimary, Text: "Добавить", URL: a.URL("/users/new"), Icon: "plus"},
				},
			},
			Search: &ui.SearchData{URL: a.URL("/users"), Placeholder: "Поиск пользователей...", Query: query, Mode: ui.SearchLive},
			List: ui.ListData{
				Columns:    columns,
				Rows:       rows,
				RenderMode: ui.RenderComfortable,
				Preset:     ui.ListDefault,
			},
		},
		NewURL: a.URL("/users/new"),
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
			a.InternalError(w, r, err)
			return
		}
	}

	title := user.Login
	if title == "" {
		title = "Новый пользователь"
	}

	pageFS, err := fs.Sub(users.Templates(), "card")
	if err != nil {
		a.InternalError(w, r, err)
		return
	}

	formAction := a.URL("/users")
	if user.ID > 0 {
		formAction = a.URL("/users/" + strconv.FormatInt(user.ID, 10))
	}

	// Удаление доступно администратору (группа /users под RequireAdmin),
	// но не для последнего администратора: действие заведомо недопустимо.
	// Серверная защита дублируется в UserDelete.
	canDelete := user.ID > 0 && !a.identity.IsLastAdministrator(user.ID)
	deleteAction := ""
	deleteConfirm := ""
	if user.ID > 0 {
		deleteAction = a.URL("/users/" + strconv.FormatInt(user.ID, 10) + "/delete")
		deleteConfirm = "Удалить пользователя «" + user.Login + "»?"
	}

	data := struct {
		Title         string
		Header        ui.HeaderData
		Card          ui.CardData
		FormAction    string
		Fields        []ui.Field
		HasPassword   bool
		CanDelete     bool
		DeleteAction  string
		DeleteConfirm string
	}{
		Title:      title,
		Header:     a.pageHeader(r, "Пользователи"),
		Card:       ui.CardData{Title: "Основная информация", CloseURL: a.URL("/users")},
		FormAction: formAction,
		Fields: []ui.Field{
			{Name: "uuid", Label: "UUID", Type: ui.FieldText, Value: user.UUID},
			{Name: "login", Label: "Логин", Type: ui.FieldText, Value: user.Login, Required: true},
			{Name: "email", Label: "Email", Type: ui.FieldText, Value: user.Email},
			{Name: "is_admin", Label: "Администратор", Type: ui.FieldCheckbox, Value: checkValue(user.IsAdmin)},
		},
		HasPassword:   user.HasPassword,
		CanDelete:     canDelete,
		DeleteAction:  deleteAction,
		DeleteConfirm: deleteConfirm,
	}

	if err := ui.RenderPage(w, TemplateFS(), pageFS, a.basePath(), data); err != nil {
		a.InternalError(w, r, err)
	}
}

func (a *App) UserSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.BadRequest(w, "Invalid request")
		return
	}

	// Снятие прав администратора уменьшает их число — операция должна быть
	// сериализована с UserDelete, чтобы инвариант «последний администратор»
	// проверялся атомарно с записью.
	a.usersMu.Lock()
	defer a.usersMu.Unlock()

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
			a.InternalError(w, r, err)
			return
		}
		user.UUID = uuid
	}

	if err := a.users.Save(r.Context(), user); err != nil {
		a.InternalError(w, r, err)
		return
	}

	if id == 0 {
		a.identity.Add(user)
	} else {
		a.identity.Update(user)
	}

	http.Redirect(w, r, a.URL("/users/"+strconv.FormatInt(user.ID, 10)), http.StatusSeeOther)
}

// UserDelete удаляет пользователя. Доступно только администратору (маршрут
// находится в группе /users под RequireAdmin). Последнего администратора
// удалить нельзя: проверка инварианта и удаление сериализованы usersMu, чтобы
// параллельные запросы не могли удалить двух администраторов одновременно.
// После удаления пользователь удаляется из IdentityService (runtime-кэш),
// как требует архитектура (удаление → Remove()).
func (a *App) UserDelete(w http.ResponseWriter, r *http.Request) {
	id := userIDFromURL(r)

	a.usersMu.Lock()
	defer a.usersMu.Unlock()

	// Проверка существования: GetByID (FindByID) отдаёт sql.ErrNoRows,
	// а DeleteByID — users.ErrNotFound, поэтому обрабатываем оба.
	if _, err := a.users.GetByID(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, users.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		a.InternalError(w, r, err)
		return
	}

	if a.identity.IsLastAdministrator(id) {
		a.BadRequest(w, users.ErrLastAdministrator.Error())
		return
	}

	if err := a.users.DeleteByID(r.Context(), id); err != nil {
		if errors.Is(err, users.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		a.InternalError(w, r, err)
		return
	}

	a.identity.Remove(id)

	http.Redirect(w, r, a.URL("/users"), http.StatusSeeOther)
}
