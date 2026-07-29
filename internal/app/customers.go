package app

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"

	"Orders/internal/app/pages"
	"Orders/internal/common"
	"Orders/internal/customers"
	"Orders/internal/entity"
	"Orders/internal/ui/display"

	"github.com/go-chi/chi/v5"
)

type customerSyncItem struct {
	UUID   string `json:"id"`
	Name   string `json:"name"`
	Active *bool  `json:"active,omitempty"`
}

func (a *App) CustomersPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	oid := orgIDFromURL(r)

	list, err := a.customers.List(r.Context(), oid)
	if err != nil {
		a.InternalError(w, err)
		return
	}

	showOrg := oid == 0

	fields := customers.Descriptor.ListFields()
	if !showOrg {
		var filtered []entity.Field
		for _, f := range fields {
			if f.GoName != "OrganizationName" {
				filtered = append(filtered, f)
			}
		}
		fields = filtered
	}

	var pageColumns []pages.Column
	for _, f := range fields {
		pageColumns = append(pageColumns, pages.Column{
			Name:  f.GoName,
			Label: f.Label,
		})
	}

	var rows []pages.Row
	for _, c := range list {
		var item display.Values = c

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
			ID:    strconv.FormatInt(c.ID, 10),
			URL:   c.URL(),
		})
	}

	newURL := "/organizations/" + chi.URLParam(r, "oid") + "/customers/new"
	if oid == 0 {
		newURL = "/customers/new"
	}

	page := pages.ListPage{
		Title:   "Контрагенты",
		Columns: pageColumns,
		Rows:    rows,

		NewURL: newURL,
		RowAction: pages.RowAction{
			Label:   "Открыть",
			BaseURL: "/organizations/" + chi.URLParam(r, "oid") + "/customers",
		},

		EmptyText: "Нет контрагентов",
	}

	a.Render(w, "customers", page)
}

func customerIDFromURL(r *http.Request) int64 {
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

func (a *App) CustomerCard(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	oid := orgIDFromURL(r)
	id := customerIDFromURL(r)

	var customer *customers.Customer

	if id == 0 {
		customer = a.customers.New()
		customer.OrganizationID = oid
	} else {
		var err error
		customer, err = a.customers.GetByID(r.Context(), id)
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
	}

	if customer.ID == 0 && oid == 0 {
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

	oid := orgIDFromURL(r)
	id := customerIDFromURL(r)

	customer := &customers.Customer{
		ID:             id,
		UUID:           r.FormValue("uuid"),
		Name:           r.FormValue("name"),
		Active:         r.FormValue("active") == "on",
		OrganizationID: oid,
	}

	if id == 0 && customer.UUID == "" {
		uuid, err := common.GenerateUUID()
		if err != nil {
			a.InternalError(w, err)
			return
		}
		customer.UUID = uuid
	}

	if oid == 0 {
		orgUUID := r.FormValue("organization_id")
		org, err := a.organizations.GetByUUID(r.Context(), orgUUID)
		if err != nil {
			a.InternalError(w, err)
			return
		}
		customer.OrganizationID = org.ID
	}

	if err := a.customers.Save(r.Context(), customer); err != nil {
		if errors.Is(err, customers.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		a.InternalError(w, err)
		return
	}

	http.Redirect(w, r,
		"/organizations/"+strconv.FormatInt(customer.OrganizationID, 10)+"/customers/"+strconv.FormatInt(customer.ID, 10),
		http.StatusSeeOther,
	)
}

func (a *App) CustomerDelete(w http.ResponseWriter, r *http.Request) {
	id := customerIDFromURL(r)

	if err := a.customers.DeleteByID(r.Context(), id); err != nil {
		a.InternalError(w, err)
		return
	}

	oid := orgIDFromURL(r)
	http.Redirect(w, r,
		"/organizations/"+strconv.FormatInt(oid, 10)+"/customers",
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

	orgUUID := chi.URLParam(r, "oid")

	org, err := a.organizations.GetByUUID(r.Context(), orgUUID)
	if err != nil {
		a.Unauthorized(w)
		return
	}

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
		if item.UUID == "" || item.Name == "" {
			a.BadRequest(w, "uuid and name are required")
			return
		}
		active := true
		if item.Active != nil {
			active = *item.Active
		}
		models[i] = customers.Customer{
			UUID:   item.UUID,
			Name:   item.Name,
			Active: active,
		}
	}

	result, err := a.customers.Synchronize(r.Context(), org.ID, models)
	if err != nil {
		a.InternalError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
