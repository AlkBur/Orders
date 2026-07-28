package app

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"Orders/internal/app/pages"
	"Orders/internal/common"
	"Orders/internal/entity"
	"Orders/internal/products"
	"Orders/internal/ui/display"

	"github.com/go-chi/chi/v5"
)

type productSyncItem struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Unit   string `json:"unit"`
	Active *bool  `json:"active,omitempty"`
}

func (a *App) ProductsPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	oid := chi.URLParam(r, "oid")
	if common.IsNilUUID(oid) {
		oid = common.NilUUID
	}
	if oid == "" {
		oid = common.NilUUID
	}

	list, err := a.products.List(r.Context(), oid)
	if err != nil {
		a.InternalError(w, err)
		return
	}

	showOrg := common.IsNilUUID(oid)

	fields := products.Descriptor.ListFields()
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
	for _, p := range list {
		var item display.Values = p

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
			ID:    p.ID,
			URL:   p.URL(),
		})
	}

	newURL := "/organizations/" + oid + "/products/" + common.NilUUID

	page := pages.ListPage{
		Title:   "Товары",
		Columns: pageColumns,
		Rows:    rows,

		NewURL: newURL,
		RowAction: pages.RowAction{
			Label:   "Открыть",
			BaseURL: "/organizations/" + oid + "/products",
		},

		EmptyText: "Нет товаров",
	}

	a.Render(w, "products", page)
}

func (a *App) ProductCard(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	oid := chi.URLParam(r, "oid")
	id := chi.URLParam(r, "id")

	isNew := common.IsNilUUID(id)

	var product *products.Product

	if isNew {
		product = a.products.New()

		if !common.IsNilUUID(oid) {
			product.OrganizationID = oid
		}
	} else {
		var err error
		product, err = a.products.Get(r.Context(), oid, id)
		if err != nil {
			a.InternalError(w, err)
			return
		}
	}

	title := product.Name
	if title == "" {
		title = "Новый товар"
	}

	page := pages.ProductCardPage{
		Title:          title,
		Product:        product,
		OrganizationID: oid,
		IsNew:          isNew,
	}

	if isNew && common.IsNilUUID(product.OrganizationID) {
		orgs, err := a.organizations.List(r.Context())
		if err != nil {
			a.InternalError(w, err)
			return
		}
		page.Orgs = orgs
	}

	a.Render(w, "product_card", page)
}

func (a *App) ProductSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.BadRequest(w, "Invalid request")
		return
	}

	oid := chi.URLParam(r, "oid")

	product := &products.Product{
		ID:     r.FormValue("id"),
		Name:   r.FormValue("name"),
		Unit:   r.FormValue("unit"),
		Active: r.FormValue("active") == "on",
	}

	if common.IsNilUUID(oid) {
		product.OrganizationID = r.FormValue("organization_id")
	} else {
		product.OrganizationID = oid
	}

	if err := a.products.Save(r.Context(), product); err != nil {
		if errors.Is(err, products.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		a.InternalError(w, err)
		return
	}

	http.Redirect(w, r,
		"/organizations/"+product.OrganizationID+"/products/"+product.ID,
		http.StatusSeeOther,
	)
}

func (a *App) ProductDelete(w http.ResponseWriter, r *http.Request) {
	oid := chi.URLParam(r, "oid")
	id := chi.URLParam(r, "id")

	if err := a.products.Delete(r.Context(), oid, id); err != nil {
		a.InternalError(w, err)
		return
	}

	http.Redirect(w, r,
		"/organizations/"+oid+"/products",
		http.StatusSeeOther,
	)
}

func (a *App) HandlePutProducts(w http.ResponseWriter, r *http.Request) {
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

	var items []productSyncItem
	if err := dec.Decode(&items); err != nil {
		a.BadRequest(w, "Invalid JSON")
		return
	}

	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		a.BadRequest(w, "Unexpected data after JSON body")
		return
	}

	models := make([]products.Product, len(items))
	for i, item := range items {
		if item.ID == "" || item.Name == "" {
			a.BadRequest(w, "id and name are required")
			return
		}
		active := true
		if item.Active != nil {
			active = *item.Active
		}
		models[i] = products.Product{
			ID:     item.ID,
			Name:   item.Name,
			Unit:   item.Unit,
			Active: active,
		}
	}

	result, err := a.products.Synchronize(r.Context(), oid, models)
	if err != nil {
		a.InternalError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}


