package app

import (
	"errors"
	"net/http"
	"strconv"

	"Orders/internal/app/pages"
	"Orders/internal/common"
	"Orders/internal/organizations"
	"Orders/internal/ui/display"

	"github.com/go-chi/chi/v5"
)

func (a *App) OrganizationsPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	orgs, err := a.organizations.List(r.Context())
	if err != nil {
		a.InternalError(w, err)
		return
	}

	fields := organizations.Descriptor.ListFields()

	var pageColumns []pages.Column
	for _, f := range fields {
		pageColumns = append(pageColumns, pages.Column{
			Name:  f.GoName,
			Label: f.Label,
		})
	}

	var rows []pages.Row
	for _, o := range orgs {
		var item display.Values = o

		var cells []string
		for _, f := range fields {
			value, err := item.DisplayValue(f.GoName)
			if err != nil {
				a.InternalError(w, err)
				return
			}
			cells = append(cells, value)
		}
		rows = append(rows, pages.Row{
			Cells: cells,
			ID:    strconv.FormatInt(o.ID, 10),
			URL:   o.URL(),
		})
	}

	page := pages.ListPage{
		Title:   "Организации",
		Columns: pageColumns,
		Rows:    rows,

		NewURL: "/organizations/new",
		RowAction: pages.RowAction{
			Label:   "Открыть",
			BaseURL: "/organizations",
		},

		EmptyText: "Нет организаций",
	}

	a.Render(w, "organizations", page)
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

	title := org.Name
	if title == "" {
		title = "Новая организация"
	}

	a.Render(w, "organization_card", pages.OrganizationCardPage{
		Title: title,
		Org:   org,
	})
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
		Name:   r.FormValue("name"),
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
