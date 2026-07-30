package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"Orders/internal/app/pages"
	"Orders/internal/common"
	"Orders/internal/receipts"
	"Orders/internal/ui"
	"Orders/internal/ui/display"

	"github.com/go-chi/chi/v5"
)

func receiptIDFromURL(r *http.Request) int64 {
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

func (a *App) ReceiptsPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	list, err := a.receipts.List(r.Context())
	if err != nil {
		a.InternalError(w, err)
		return
	}

	fields := receipts.Descriptor.ListFields()

	var pageColumns []pages.Column
	for _, f := range fields {
		pageColumns = append(pageColumns, pages.Column{
			Name:  f.GoName,
			Label: f.Label,
		})
	}

	var rows []pages.Row
	for _, rec := range list {
		var item display.Values = *rec

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
			ID:    strconv.FormatInt(rec.ID, 10),
			URL:   rec.URL(),
		})
	}

	page := pages.ListPage{
		Title:     "Товарные чеки",
		Columns:   pageColumns,
		Rows:      rows,
		NewURL:    "/receipts/new",
		RowAction: pages.RowAction{
			Label:   "Открыть",
			BaseURL: "/receipts",
		},
		EmptyText: "Нет товарных чеков",
	}

	a.Render(w, "receipts", page)
}

func (a *App) ReceiptCard(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	id := receiptIDFromURL(r)

	var doc *receipts.Document
	if id == 0 {
		rec := a.receipts.New()
		var items []receipts.ReceiptItem

		if lookup, ok := ui.ReadLookup(r); ok {
			switch lookup.FieldName {
			case ui.LookupCustomer:
				if a.customers != nil {
					cust, err := a.customers.GetByID(r.Context(), lookup.ID)
					if err == nil {
						rec.CustomerID = cust.ID
						rec.CustomerName = cust.Name
					}
				}
			case ui.LookupProduct:
				if a.products != nil {
					prod, err := a.products.GetByID(r.Context(), lookup.ID)
					if err == nil {
						items = append(items, receipts.ReceiptItem{
							LineNum:     1,
							ProductID:   prod.ID,
							ProductName: prod.Name,
							Unit:        prod.Unit,
							Quantity:    1,
							Price:       0,
							Amount:      0,
						})
					}
				}
			}
		}

		doc = &receipts.Document{Receipt: rec, Items: items}
	} else {
		var err error
		doc, err = a.receipts.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, receipts.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			a.InternalError(w, err)
			return
		}
	}

	title := "Товарный чек №" + doc.Receipt.Number
	if doc.Receipt.ID == 0 {
		title = "Новый товарный чек"
	}

	itemsJSON, _ := toJSON(doc.Items)

	page := pages.ReceiptCardPage{
		Title:          title,
		Receipt:        doc.Receipt,
		Items:          doc.Items,
		CustomerID:     doc.Receipt.CustomerID,
		CustomerName:   doc.Receipt.CustomerName,
		OrganizationID: doc.Receipt.OrganizationID,
		Errors:         make(map[string]string),
		ErrorsJSON:     "{}",
		ItemsJSON:      itemsJSON,
	}

	if doc.Receipt.ID == 0 && a.organizations != nil {
		orgs, err := a.organizations.List(r.Context())
		if err != nil {
			a.InternalError(w, err)
			return
		}
		page.Orgs = orgs
	} else if a.customers != nil {
		customers, err := a.customers.List(r.Context(), doc.Receipt.OrganizationID)
		if err != nil {
			a.InternalError(w, err)
			return
		}
		page.Customers = customers
	}

	a.Render(w, "receipt_card", page)
}

func toJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (a *App) ReceiptSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.BadRequest(w, "Invalid request")
		return
	}

	id := receiptIDFromURL(r)

	if id > 0 {
		existing, err := a.receipts.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, receipts.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			a.InternalError(w, err)
			return
		}
		if existing.Receipt.SentAt != nil {
			a.BadRequest(w, receipts.ErrReceiptReadOnly.Error())
			return
		}
	}

	orgID := parseInt64(r.FormValue("organization_id"))
	customerID := parseInt64(r.FormValue("customer_id"))
	total := parseFloat(r.FormValue("total"))

	var items []receipts.ReceiptItem
	for i := 0; ; i++ {
		productID := parseInt64(r.FormValue("items[" + strconv.Itoa(i) + "][product_id]"))
		if productID == 0 && i > 0 {
			break
		}
		if productID == 0 {
			continue
		}
		items = append(items, receipts.ReceiptItem{
			LineNum:   i + 1,
			ProductID: productID,
			Unit:      r.FormValue("items[" + strconv.Itoa(i) + "][unit]"),
			Quantity:  parseFloat(r.FormValue("items[" + strconv.Itoa(i) + "][quantity]")),
			Price:     parseFloat(r.FormValue("items[" + strconv.Itoa(i) + "][price]")),
			Amount:    parseFloat(r.FormValue("items[" + strconv.Itoa(i) + "][amount]")),
		})
	}

	dateStr := r.FormValue("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		a.BadRequest(w, "Invalid date")
		return
	}

	var customerName string
	if customerID > 0 && a.customers != nil {
		if c, err := a.customers.GetByID(r.Context(), customerID); err == nil {
			customerName = c.Name
		}
	}

	userID := CurrentUser(r).ID
	if userID == 0 {
		userID = parseInt64(r.FormValue("user_id"))
	}

	rec := &receipts.Receipt{
		ID:             id,
		Number:         r.FormValue("number"),
		Date:           date,
		OrganizationID: orgID,
		UserID:         userID,
		CustomerID:     customerID,
		CustomerName:   customerName,
		Total:          total,
	}

	if id == 0 {
		if orgID == 0 || customerID == 0 {
			errs := map[string]string{}
			if orgID == 0 {
				errs["organization_id"] = "Выберите организацию"
			}
			if customerID == 0 {
				errs["customer_id"] = "Выберите клиента"
			}
			renderReceiptFormWithErrors(w, r, a, errs, items)
			return
		}
		if len(items) == 0 {
			renderReceiptFormWithErrors(w, r, a, map[string]string{"": "Добавьте хотя бы одну позицию"}, items)
			return
		}

		uuid, err := common.GenerateUUID()
		if err != nil {
			a.InternalError(w, err)
			return
		}
		rec.ExchangeID = uuid
	}

	doc := &receipts.Document{Receipt: rec, Items: items}
	if err := a.receipts.Save(r.Context(), doc); err != nil {
		a.InternalError(w, err)
		return
	}

	if isHtmxRequest(r) {
		w.Header().Set("HX-Redirect", "/receipts/"+strconv.FormatInt(rec.ID, 10))
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/receipts/"+strconv.FormatInt(rec.ID, 10), http.StatusSeeOther)
}

func renderReceiptFormWithErrors(w http.ResponseWriter, r *http.Request, a *App, errs map[string]string, items []receipts.ReceiptItem) {
	itemsJSON, _ := toJSON(items)
	customerID := parseInt64(r.FormValue("customer_id"))
	var customerName string
	if customerID > 0 && a.customers != nil {
		if c, err := a.customers.GetByID(r.Context(), customerID); err == nil {
			customerName = c.Name
		}
	}

	page := pages.ReceiptCardPage{
		Title:          "Новый товарный чек",
		Receipt:        a.receipts.New(),
		Errors:         errs,
		OrganizationID: parseInt64(r.FormValue("organization_id")),
		CustomerID:     customerID,
		CustomerName:   customerName,
		Items:          items,
		ItemsJSON:      itemsJSON,
	}
	if a.organizations != nil {
		orgs, err := a.organizations.List(r.Context())
		if err == nil {
			page.Orgs = orgs
		}
	}
	page.ErrorsJSON, _ = toJSON(errs)
	if page.ErrorsJSON == "" {
		page.ErrorsJSON = "{}"
	}
	w.WriteHeader(http.StatusUnprocessableEntity)
	a.Render(w, "receipt_card", page)
}

func isHtmxRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func (a *App) ReceiptDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	existing, err := a.receipts.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, receipts.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		a.InternalError(w, err)
		return
	}
	if existing.Receipt.SentAt != nil {
		a.BadRequest(w, receipts.ErrReceiptReadOnly.Error())
		return
	}

	if err := a.receipts.DeleteByID(r.Context(), id); err != nil {
		a.InternalError(w, err)
		return
	}

	http.Redirect(w, r, "/receipts", http.StatusSeeOther)
}

func (a *App) ReceiptSubmit(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	doc, err := a.receipts.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, receipts.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		a.InternalError(w, err)
		return
	}

	if doc.Receipt.SentAt != nil {
		a.BadRequest(w, receipts.ErrReceiptReadOnly.Error())
		return
	}

	now := time.Now()
	doc.Receipt.SentAt = &now
	doc.Receipt.UpdatedAt = now

	if err := a.receipts.Save(r.Context(), doc); err != nil {
		a.InternalError(w, err)
		return
	}

	http.Redirect(w, r, "/receipts/"+idStr, http.StatusSeeOther)
}

func parseInt64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
