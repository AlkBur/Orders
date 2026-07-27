package app

import (
	"net/http"

	"Orders/internal/app/pages"
	"Orders/internal/organizations"

	"github.com/go-chi/chi/v5"
)

func (a *App) OrganizationCard(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	uuid := chi.URLParam(r, "uuid")

	var org *organizations.Organization
	if uuid == "" {
		org = a.organizations.New()
	} else {
		var err error
		org, err = a.organizations.GetByUUID(r.Context(), uuid)
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

	org := &organizations.Organization{
		UUID:   r.FormValue("uuid"),
		Name:   r.FormValue("name"),
		Active: r.FormValue("active") == "on",
	}

	if err := a.organizations.Save(r.Context(), org); err != nil {
		a.InternalError(w, err)
		return
	}

	http.Redirect(w, r, "/organizations/"+org.UUID, http.StatusSeeOther)
}
