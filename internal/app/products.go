package app

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"Orders/internal/app/pages"
	"Orders/internal/common"
	"Orders/internal/entity"
	"Orders/internal/organizations"
	"Orders/internal/products"
	"Orders/internal/ui"
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
	query := strings.TrimSpace(r.URL.Query().Get("q"))

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

	visibleFields := entity.Names(fields)

	list, err := a.products.List(r.Context(), oid, products.ListOptions{Query: query}, visibleFields)
	if err != nil {
		a.InternalError(w, err)
		return
	}

	var columns []ui.ListColumn
	for _, f := range fields {
		columns = append(columns, ui.ListColumn{Label: f.Label})
	}

	mode := r.URL.Query().Get("mode")
	pickerField := r.URL.Query().Get("field")
	pickerMode := mode == "picker" && pickerField != ""

	var rows []ui.ListRow
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

		row := ui.ListRow{Cells: cells}
		if pickerMode {
			row.Actions = []ui.RowAction{{
				ID:    "select",
				Icon:  "check",
				Label: "Выбрать",
				URL:   pickerSelectURL(r, p.ID),
			}}
		} else {
			row.URL = p.URL()
		}
		rows = append(rows, row)
	}

	pageFS, err := fs.Sub(products.Templates(), "list")
	if err != nil {
		a.InternalError(w, err)
		return
	}

	oidPath := chi.URLParam(r, "oid")
	listPath := "/products"
	newURL := "/products/new"
	if oidPath != "" {
		listPath = "/organizations/" + oidPath + "/products"
		newURL = listPath + "/new"
	}

	page := pages.ListViewPage{
		Title:  "Товары",
		Header: pageHeader(r, "Товары"),
		List: ui.ListView{
			List: ui.ListData{
				Columns:    columns,
				Rows:       rows,
				RenderMode: ui.RenderComfortable,
				Preset:     ui.ListDefault,
			},
		},
	}

	if !pickerMode {
		page.NewURL = newURL
		page.List.Toolbar = &ui.ToolbarData{
			Buttons: []ui.Button{
				{Style: ui.ButtonPrimary, Text: "Добавить", URL: newURL, Icon: "plus"},
			},
		}
	}
	searchURL := listPath
	if pickerMode {
		searchURL = pickerListURL(r, listPath)
	}
	page.List.Search = &ui.SearchData{URL: searchURL, Placeholder: "Поиск товаров...", Query: query, Mode: ui.SearchLive}

	a.renderListView(w, r, TemplateFS(), pageFS, page)
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

	pageFS, err := fs.Sub(products.Templates(), "card")
	if err != nil {
		a.InternalError(w, err)
		return
	}

	formAction := "/products"
	closeURL := "/products"
	if oid > 0 {
		formAction = "/organizations/" + strconv.FormatInt(oid, 10) + "/products"
		closeURL = formAction
		if product.ID > 0 {
			formAction += "/" + strconv.FormatInt(product.ID, 10)
		}
	} else if product.ID > 0 {
		formAction = "/organizations/" + strconv.FormatInt(product.OrganizationID, 10) + "/products/" + strconv.FormatInt(product.ID, 10)
	}

	fields := []ui.Field{
		{Name: "uuid", Label: "UUID", Type: ui.FieldText, Value: product.UUID},
	}

	if product.ID == 0 && oid == 0 {
		orgs, err := a.organizations.List(r.Context(), organizations.ListOptions{}, nil)
		if err != nil {
			a.InternalError(w, err)
			return
		}
		options := make([]ui.SelectOption, 0, len(orgs))
		for _, org := range orgs {
			options = append(options, ui.SelectOption{Value: org.UUID, Label: org.Name})
		}
		fields = append(fields, ui.Field{
			Name:        "organization_id",
			Label:       "Организация",
			Type:        ui.FieldSelect,
			Value:       "",
			Required:    true,
			Placeholder: "Выберите организацию",
			Options:     options,
		})
	}

	fields = append(fields,
		ui.Field{Name: "name", Label: "Наименование", Type: ui.FieldText, Value: product.Name, Required: true},
		ui.Field{Name: "unit", Label: "Ед. изм", Type: ui.FieldText, Value: product.Unit},
		ui.Field{Name: "active", Label: "Активен", Type: ui.FieldCheckbox, Value: checkValue(product.Active)},
	)

	data := struct {
		Title      string
		Header     ui.HeaderData
		Card       ui.CardData
		FormAction string
		Fields     []ui.Field
	}{
		Title:      title,
		Header:     pageHeader(r, "Товары"),
		Card:       ui.CardData{Title: "Основная информация", CloseURL: closeURL},
		FormAction: formAction,
		Fields:     fields,
	}

	if err := ui.RenderPage(w, TemplateFS(), pageFS, data); err != nil {
		a.InternalError(w, err)
	}
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
