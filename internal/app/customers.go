package app

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"

	"Orders/internal/app/pages"
	"Orders/internal/common"
	"Orders/internal/customers"

	"github.com/go-chi/chi/v5"
)

type customerSyncItem struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active *bool  `json:"active,omitempty"`
}

func (a *App) CustomersPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	oid := chi.URLParam(r, "oid")
	if common.IsNilUUID(oid) {
		oid = common.NilUUID
	}
	if oid == "" {
		oid = common.NilUUID
	}

	list, err := a.customers.List(r.Context(), oid)
	if err != nil {
		a.InternalError(w, err)
		return
	}

	title := "Контрагенты"
	if !common.IsNilUUID(oid) {
		title = "Контрагенты"
	}

	a.Render(w, "customers", pages.CustomersPage{
		Title:          title,
		Customers:      list,
		OrganizationID: oid,
	})
}

func (a *App) CustomerCard(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	oid := chi.URLParam(r, "oid")
	id := chi.URLParam(r, "id")

	isNew := common.IsNilUUID(id)

	var customer *customers.Customer

	if isNew {
		customer = a.customers.New()

		if !common.IsNilUUID(oid) {
			customer.OrganizationID = oid
		}
	} else {
		var err error
		customer, err = a.customers.Get(r.Context(), oid, id)
		if err != nil {
			a.InternalError(w, err)
			return
		}
	}

	title := customer.Name
	if title == "" {
		title = "Новый контрагент"
	}

	page := pages.CustomerCardPage{
		Title:          title,
		Customer:       customer,
		OrganizationID: oid,
		IsNew:          isNew,
	}

	if isNew && common.IsNilUUID(customer.OrganizationID) {
		orgs, err := a.organizations.List(r.Context())
		if err != nil {
			a.InternalError(w, err)
			return
		}
		page.Orgs = orgs
	}

	a.Render(w, "customer_card", page)
}

func (a *App) CustomerSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.BadRequest(w, "Invalid request")
		return
	}

	oid := chi.URLParam(r, "oid")

	customer := &customers.Customer{
		ID:     r.FormValue("id"),
		Name:   r.FormValue("name"),
		Active: r.FormValue("active") == "on",
	}

	if common.IsNilUUID(oid) {
		customer.OrganizationID = r.FormValue("organization_id")
	} else {
		customer.OrganizationID = oid
	}

	if err := a.customers.Save(r.Context(), customer); err != nil {
		a.InternalError(w, err)
		return
	}

	http.Redirect(w, r,
		"/organizations/"+customer.OrganizationID+"/customers/"+customer.ID,
		http.StatusSeeOther,
	)
}

func (a *App) CustomerDelete(w http.ResponseWriter, r *http.Request) {
	oid := chi.URLParam(r, "oid")
	id := chi.URLParam(r, "id")

	if err := a.customers.Delete(r.Context(), oid, id); err != nil {
		a.InternalError(w, err)
		return
	}

	http.Redirect(w, r,
		"/organizations/"+oid+"/customers",
		http.StatusSeeOther,
	)
}

func (a *App) HandlePutCustomers(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	defer r.Body.Close()

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		a.BadRequest(w, "Content-Type must be application/json")
		return
	}

	oid := chi.URLParam(r, "oid")

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var items []customerSyncItem
	if err := dec.Decode(&items); err != nil {
		a.BadRequest(w, "Invalid JSON")
		return
	}

	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		a.BadRequest(w, "Unexpected data after JSON body")
		return
	}

	models := make([]customers.Customer, len(items))
	for i, item := range items {
		if item.ID == "" || item.Name == "" {
			a.BadRequest(w, "id and name are required")
			return
		}
		active := true
		if item.Active != nil {
			active = *item.Active
		}
		models[i] = customers.Customer{
			ID:     item.ID,
			Name:   item.Name,
			Active: active,
		}
	}

	result, err := a.customers.Synchronize(r.Context(), oid, models)
	if err != nil {
		a.InternalError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
