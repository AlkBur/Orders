package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"strconv"

	"Orders/internal/common"
	"Orders/internal/customers"
	"Orders/internal/entity"
	"Orders/internal/organizations"
	"Orders/internal/sessions"
	"Orders/internal/ui"
	"Orders/internal/ui/display"

	"github.com/go-chi/chi/v5"
)

type customerSyncItem struct {
	UUID   string `json:"id"`
	Name   string `json:"name"`
	Active *bool  `json:"active,omitempty"`
}

type customersListData struct {
	Title   string
	Header  ui.HeaderData
	Toolbar ui.ToolbarData
	List    ui.ListData
	NewURL  string
	Alert   *ui.AlertData
}

func (d customersListData) FAB() *ui.FAB {
	if d.NewURL == "" {
		return nil
	}
	return &ui.FAB{Icon: "plus", URL: d.NewURL, Text: "Добавить"}
}

type customerCardData struct {
	Title      string
	Header     ui.HeaderData
	FormAction string
	Fields     []ui.Field
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

	var columns []ui.ListColumn
	for _, f := range fields {
		columns = append(columns, ui.ListColumn{Label: f.Label})
	}

	mode := r.URL.Query().Get("mode")
	pickerField := r.URL.Query().Get("field")
	pickerMode := mode == "picker" && pickerField != ""

	var rows []ui.ListRow
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

		row := ui.ListRow{Cells: cells}
		if pickerMode {
			row.Actions = []ui.RowAction{{
				ID:    "select",
				Icon:  "check",
				Label: "Выбрать",
				URL:   pickerSelectURL(r, c.ID),
			}}
		} else {
			row.URL = c.URL()
		}
		rows = append(rows, row)
	}

	pageFS, err := fs.Sub(customers.Templates(), "list")
	if err != nil {
		a.InternalError(w, err)
		return
	}

	data := customersListData{
		Title:  "Контрагенты",
		Header: pageHeader(r, "Контрагенты"),
		List: ui.ListData{
			Columns:    columns,
			Rows:       rows,
			RenderMode: ui.RenderComfortable,
			Preset:     ui.ListDefault,
		},
	}

	if !pickerMode {
		newURL := "/customers/new"
		if oid > 0 {
			newURL = "/organizations/" + strconv.FormatInt(oid, 10) + "/customers/new"
		}
		data.NewURL = newURL
		data.Toolbar = ui.ToolbarData{
			Buttons: []ui.Button{
				{Style: ui.ButtonPrimary, Text: "Добавить", URL: newURL, Icon: "plus"},
			},
		}
	}

	flash, err := a.consumeFlash(r)
	if err != nil {
		a.InternalError(w, err)
		return
	}
	if flash != nil {
		data.Alert = FlashToAlert(*flash)
	}

	if err := ui.RenderPage(w, TemplateFS(), pageFS, data); err != nil {
		a.InternalError(w, err)
	}
}

// pickerSelectURL builds the picker callback link consumed by the
// receipts page: {return_to}?select_id=...&select_field=...&return_to=...
func pickerSelectURL(r *http.Request, id int64) string {
	returnURL := r.URL.Query().Get("return_to")
	return fmt.Sprintf("%s?select_id=%d&select_field=%s&return_to=%s",
		returnURL, id, r.URL.Query().Get("field"), returnURL)
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
		if err == customers.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			a.InternalError(w, err)
			return
		}
	}

	title := "Контрагент"

	fields := []ui.Field{
		{Name: "uuid", Label: "UUID", Type: ui.FieldText, Value: customer.UUID, Readonly: true},
	}

	if customer.ID == 0 && oid == 0 {
		orgs, err := a.organizations.List(r.Context(), organizations.ListOptions{})
		if err != nil {
			a.InternalError(w, err)
			return
		}
		options := make([]ui.SelectOption, 0, len(orgs))
		for _, o := range orgs {
			options = append(options, ui.SelectOption{Value: o.UUID, Label: o.Name})
		}
		fields = append(fields, ui.Field{
			Name:        "organization_id",
			Label:       "Организация",
			Type:        ui.FieldSelect,
			Required:    true,
			Placeholder: "Выберите организацию",
			Options:     options,
		})
	}

	fields = append(fields,
		ui.Field{Name: "name", Label: "Наименование", Type: ui.FieldText, Value: customer.Name, Required: true},
		ui.Field{Name: "active", Label: "Активен", Type: ui.FieldCheckbox, Value: checkValue(customer.Active)},
	)

	pageFS, err := fs.Sub(customers.Templates(), "card")
	if err != nil {
		a.InternalError(w, err)
		return
	}

	action := "/customers"
	if oid > 0 {
		action = "/organizations/" + strconv.FormatInt(oid, 10) + "/customers"
		if id > 0 {
			action += "/" + strconv.FormatInt(id, 10)
		}
	}

	data := customerCardData{
		Title:      title,
		Header:     pageHeader(r, "Контрагенты"),
		FormAction: action,
		Fields:     fields,
	}

	if err := ui.RenderPage(w, TemplateFS(), pageFS, data); err != nil {
		a.InternalError(w, err)
	}
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

	if err := a.SetFlash(r, sessions.FlashSuccess, "Контрагент сохранён."); err != nil {
		a.InternalError(w, err)
		return
	}

	target := "/customers"
	if oid > 0 {
		target = "/organizations/" + strconv.FormatInt(oid, 10) + "/customers"
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
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
