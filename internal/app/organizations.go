package app

import (
	"net/http"

	"Orders/internal/app/pages"
)

func (a *App) OrganizationsPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	orgs, err := a.organizations.List(r.Context())
	if err != nil {
		a.InternalError(w, err)
		return
	}

	a.Render(w, "organizations", pages.OrganizationsPage{
		Title: "Организации",
		Orgs:  orgs,
	})
}
