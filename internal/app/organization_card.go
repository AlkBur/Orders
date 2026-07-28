package app

import (
	"errors"
	"net/http"

	"Orders/internal/app/pages"
	"Orders/internal/common"
	"Orders/internal/organizations"

	"github.com/go-chi/chi/v5"
)

func (a *App) OrganizationCard(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	id := chi.URLParam(r, "id")
	isNew := common.IsNilUUID(id)

	var org *organizations.Organization
	if isNew {
		org = a.organizations.New()
	} else {
		var err error
		org, err = a.organizations.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, organizations.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
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
		IsNew: isNew,
	})
}

func (a *App) OrganizationSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.BadRequest(w, "Invalid request")
		return
	}

	id := chi.URLParam(r, "id")

	org := &organizations.Organization{
		UUID:   id,
		Name:   r.FormValue("name"),
		Active: r.FormValue("active") == "on",
	}

	if err := a.organizations.Save(r.Context(), org); err != nil {
		if errors.Is(err, organizations.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		a.InternalError(w, err)
		return
	}

	http.Redirect(w, r, "/organizations/"+org.UUID, http.StatusSeeOther)
}
