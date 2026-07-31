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
	"Orders/internal/entity"
	"Orders/internal/organizations"
	"Orders/internal/products"
	"Orders/internal/ui/display"

	"github.com/go-chi/chi/v5"
)

type productSyncItem struct {
	UUID   string `json:"id"`
	Name   string `json:"name"`
	Unit   string `json:"unit"`
	Active *bool  `json:"active,omitempty"`
}

func orgIDFromURL(r *http.Request) int64 {
	oidStr := chi.URLParam(r, "oid")
	if oidStr == "" || oidStr == "new" {
		return 0
	}
	id, err := strconv.ParseInt(oidStr, 10, 64)
	if err != nil {
		return 0
	}
	return id
}

func (a *App) ProductsPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	oid := orgIDFromURL(r)

	list, err := a.products.List(r.Context(), oid)
	if err != nil {
		a.InternalError(w, err)
		return
	}

	showOrg := oid == 0

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
			ID:    strconv.FormatInt(p.ID, 10),
			URL:   p.URL(),
		})
	}

	mode := r.URL.Query().Get("mode")
	pickerField := r.URL.Query().Get("field")

	page := pages.ListPage{
		Title:   "Товары",
		Columns: pageColumns,
		Rows:    rows,

		EmptyText: "Нет товаров",
	}

	if mode == "picker" && pickerField != "" {
		page.PickerMode = true
		page.PickerField = pickerField
		page.ReturnURL = r.URL.Query().Get("return_to")
		page.RowAction = pages.RowAction{Label: "Выбрать"}
	} else {
		newURL := "/organizations/" + chi.URLParam(r, "oid") + "/products/new"
		if oid == 0 {
			newURL = "/products/new"
		}
		page.NewURL = newURL
		page.RowAction = pages.RowAction{
			Label:   "Открыть",
			BaseURL: "/organizations/" + chi.URLParam(r, "oid") + "/products",
		}
	}

	a.Render(w, "products", page)
}

func productIDFromURL(r *http.Request) int64 {
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

func (a *App) ProductCard(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	oid := orgIDFromURL(r)
	id := productIDFromURL(r)

	var product *products.Product

	if id == 0 {
		product = a.products.New()
		product.OrganizationID = oid
	} else {
		var err error
		product, err = a.products.GetByID(r.Context(), id)
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
	}

	if product.ID == 0 && oid == 0 {
		orgs, err := a.organizations.List(r.Context(), organizations.ListOptions{})
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

	oid := orgIDFromURL(r)
	id := productIDFromURL(r)

	product := &products.Product{
		ID:             id,
		UUID:           r.FormValue("uuid"),
		Name:           r.FormValue("name"),
		Unit:           r.FormValue("unit"),
		Active:         r.FormValue("active") == "on",
		OrganizationID: oid,
	}

	if id == 0 && product.UUID == "" {
		uuid, err := common.GenerateUUID()
		if err != nil {
			a.InternalError(w, err)
			return
		}
		product.UUID = uuid
	}

	if oid == 0 {
		orgUUID := r.FormValue("organization_id")
		org, err := a.organizations.GetByUUID(r.Context(), orgUUID)
		if err != nil {
			a.InternalError(w, err)
			return
		}
		product.OrganizationID = org.ID
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
		"/organizations/"+strconv.FormatInt(product.OrganizationID, 10)+"/products/"+strconv.FormatInt(product.ID, 10),
		http.StatusSeeOther,
	)
}

func (a *App) ProductDelete(w http.ResponseWriter, r *http.Request) {
	id := productIDFromURL(r)

	if err := a.products.DeleteByID(r.Context(), id); err != nil {
		a.InternalError(w, err)
		return
	}

	oid := orgIDFromURL(r)
	http.Redirect(w, r,
		"/organizations/"+strconv.FormatInt(oid, 10)+"/products",
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

	orgUUID := chi.URLParam(r, "oid")

	org, err := a.organizations.GetByUUID(r.Context(), orgUUID)
	if err != nil {
		a.Unauthorized(w)
		return
	}

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
		if item.UUID == "" || item.Name == "" {
			a.BadRequest(w, "uuid and name are required")
			return
		}
		active := true
		if item.Active != nil {
			active = *item.Active
		}
		models[i] = products.Product{
			UUID:   item.UUID,
			Name:   item.Name,
			Unit:   item.Unit,
			Active: active,
		}
	}

	result, err := a.products.Synchronize(r.Context(), org.ID, models)
	if err != nil {
		a.InternalError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
