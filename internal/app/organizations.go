package app

import (
	"errors"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"Orders/internal/app/pages"
	"Orders/internal/common"
	"Orders/internal/organizations"
	"Orders/internal/ui"

	"github.com/go-chi/chi/v5"
)

type organizationsListData struct {
	Title   string
	Header  pages.HeaderData
	Toolbar pages.ToolbarData
	Search  pages.SearchData
	List    pages.ListData
}

func (organizationsListData) FAB() *ui.FAB {
	return &ui.FAB{Icon: "plus", URL: "/organizations/new", Text: "Добавить"}
}

type organizationCardData struct {
	Title      string
	Header     pages.HeaderData
	ID         int64
	Name       string
	Active     bool
	FormAction string
	Fields     []pages.Field
}

func (a *App) OrganizationsPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	query := strings.TrimSpace(r.URL.Query().Get("q"))

	orgs, err := a.organizations.List(r.Context(), organizations.ListOptions{Query: query})
	if err != nil {
		a.InternalError(w, err)
		return
	}

	var rows []pages.ListRow
	for _, o := range orgs {
		rows = append(rows, pages.ListRow{
			URL: "/organizations/" + strconv.FormatInt(o.ID, 10),
			Cells: []string{
				o.Name,
				orgStatus(o.Active),
				o.CreatedAt.Format("02.01.2006"),
			},
		})
	}

	pageFS, err := fs.Sub(organizations.Templates(), "list")
	if err != nil {
		a.InternalError(w, err)
		return
	}

	data := organizationsListData{
		Title:  "Организации",
		Header: pageHeader(r, "Организации"),
		Toolbar: pages.ToolbarData{
			Buttons: []pages.Button{
				{Style: pages.ButtonPrimary, Text: "Добавить", URL: "/organizations/new", Icon: "plus"},
			},
		},
		Search: pages.SearchData{Placeholder: "Поиск организаций...", Value: query},
		List: pages.ListData{
			Columns: []pages.ListColumn{
				{Label: "Название"},
				{Label: "Статус"},
				{Label: "Создана"},
			},
			Rows:       rows,
			RenderMode: pages.RenderComfortable,
			Preset:     pages.ListOrganizations,
		},
	}

	if err := ui.RenderPage(w, TemplateFS(), pageFS, data); err != nil {
		a.InternalError(w, err)
	}
}

func organizationID(r *http.Request) int64 {
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

func (a *App) OrganizationCard(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	id := organizationID(r)

	var org *organizations.Organization
	if id == 0 {
		org = a.organizations.New()
	} else {
		var err error
		org, err = a.organizations.GetByID(r.Context(), id)
		if err == organizations.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			a.InternalError(w, err)
			return
		}
	}

	pageFS, err := fs.Sub(organizations.Templates(), "card")
	if err != nil {
		a.InternalError(w, err)
		return
	}

	name := org.Name
	if name == "" {
		name = "Новая организация"
	}

	action := "/organizations"
	if id > 0 {
		action = "/organizations/" + strconv.FormatInt(id, 10)
	}

	data := organizationCardData{
		Title:      name,
		Header:     pageHeader(r, "Организации"),
		ID:         id,
		Name:       name,
		Active:     org.Active,
		FormAction: action,
		Fields: []pages.Field{
			{Name: "uuid", Label: "UUID", Type: pages.FieldText, Value: org.UUID, Readonly: true},
			{Name: "name", Label: "Наименование", Type: pages.FieldText, Value: org.Name, Required: true},
			{Name: "active", Label: "Активна", Type: pages.FieldCheckbox, Value: checkValue(org.Active)},
			{Name: "apikey", Label: "API Key", Type: pages.FieldText, Value: org.APIKey, Readonly: true},
		},
	}

	if err := ui.RenderPage(w, TemplateFS(), pageFS, data); err != nil {
		a.InternalError(w, err)
	}
}

func (a *App) OrganizationDelete(w http.ResponseWriter, r *http.Request) {
	id := organizationID(r)
	if id == 0 {
		http.NotFound(w, r)
		return
	}

	if err := a.organizations.DeleteByID(r.Context(), id); err != nil {
		if errors.Is(err, organizations.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		a.InternalError(w, err)
		return
	}

	http.Redirect(w, r, "/organizations", http.StatusSeeOther)
}

func (a *App) OrganizationSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.BadRequest(w, "Invalid request")
		return
	}

	id := organizationID(r)

	org := &organizations.Organization{
		ID:     id,
		UUID:   r.FormValue("uuid"),
		Name:   strings.TrimSpace(r.FormValue("name")),
		Active: r.FormValue("active") == "on",
	}

	if id == 0 && org.UUID == "" {
		uuid, err := common.GenerateUUID()
		if err != nil {
			a.InternalError(w, err)
			return
		}
		org.UUID = uuid
	}

	if err := a.organizations.Save(r.Context(), org); err != nil {
		if errors.Is(err, organizations.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		a.InternalError(w, err)
		return
	}

	a.orgKeysMu.Lock()
	if a.orgKeys != nil {
		a.orgKeys[org.UUID] = org.APIKey
	}
	a.orgKeysMu.Unlock()

	http.Redirect(w, r, "/organizations/"+strconv.FormatInt(org.ID, 10), http.StatusSeeOther)
}

func pageHeader(r *http.Request, section string) pages.HeaderData {
	user := CurrentUser(r)
	return pages.HeaderData{
		Section:  section,
		Username: user.Login,
		Menu: []pages.MenuItem{
			{ID: "logout", Label: "Выход", Icon: "logout", URL: "/logout"},
		},
	}
}

func orgStatus(active bool) string {
	if active {
		return "Активна"
	}
	return "Неактивна"
}

func checkValue(v bool) string {
	if v {
		return "true"
	}
	return ""
}
