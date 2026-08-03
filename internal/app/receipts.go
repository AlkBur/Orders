package app

import (
	"errors"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"Orders/internal/app/pages"
	"Orders/internal/common"
	"Orders/internal/customers"
	"Orders/internal/entity"
	"Orders/internal/organizations"
	"Orders/internal/products"
	"Orders/internal/receipts"
	"Orders/internal/ui"
	"Orders/internal/ui/display"

	"github.com/go-chi/chi/v5"
)

// lookupCustomer / lookupProduct — значения select_field в callback-ссылках
// пикера. Принадлежат приложению (потребителю ReadLookup), а не библиотеке ui.
const (
	lookupCustomer = "customer"
	lookupProduct  = "product"
)

type receiptEditorItem struct {
	ProductID   int64   `json:"product_id"`
	ProductName string  `json:"product_name"`
	Unit        string  `json:"unit"`
	Quantity    float64 `json:"quantity"`
	Price       float64 `json:"price"`
	Amount      float64 `json:"amount"`
}

type receiptCustomerOption struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	OrganizationID int64  `json:"organization_id"`
}

type receiptProductOption struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Unit           string `json:"unit"`
	OrganizationID int64  `json:"organization_id"`
}

func receiptEditorJSON(items []receipts.ReceiptItem, customers []*customers.Customer, products []*products.Product) (string, string, string, error) {
	itemViews := make([]receiptEditorItem, 0, len(items))
	for _, item := range items {
		itemViews = append(itemViews, receiptEditorItem{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Unit:        item.Unit,
			Quantity:    item.Quantity,
			Price:       item.Price,
			Amount:      item.Amount,
		})
	}

	customerViews := make([]receiptCustomerOption, 0, len(customers))
	for _, customer := range customers {
		customerViews = append(customerViews, receiptCustomerOption{
			ID:             customer.ID,
			Name:           customer.Name,
			OrganizationID: customer.OrganizationID,
		})
	}

	productViews := make([]receiptProductOption, 0, len(products))
	for _, product := range products {
		productViews = append(productViews, receiptProductOption{
			ID:             product.ID,
			Name:           product.Name,
			Unit:           product.Unit,
			OrganizationID: product.OrganizationID,
		})
	}

	itemsJSON, err := common.ToJSON(itemViews)
	if err != nil {
		return "", "", "", err
	}
	customersJSON, err := common.ToJSON(customerViews)
	if err != nil {
		return "", "", "", err
	}
	productsJSON, err := common.ToJSON(productViews)
	if err != nil {
		return "", "", "", err
	}
	return itemsJSON, customersJSON, productsJSON, nil
}

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

	query := strings.TrimSpace(r.URL.Query().Get("q"))

	fields := receipts.Descriptor.ListFields()
	visibleFields := entity.Names(fields)

	list, err := a.receipts.List(r.Context(), receipts.ListOptions{Query: query}, visibleFields)
	if err != nil {
		a.InternalError(w, err)
		return
	}

	var columns []ui.ListColumn
	for _, f := range fields {
		columns = append(columns, ui.ListColumn{Label: f.Label})
	}

	var rows []ui.ListRow
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
		rows = append(rows, ui.ListRow{
			Cells: cells,
			URL:   rec.URL(),
		})
	}

	pageFS, err := fs.Sub(receipts.Templates(), "list")
	if err != nil {
		a.InternalError(w, err)
		return
	}

	page := pages.ListViewPage{
		Title:  "Товарные чеки",
		Header: pageHeader(r, "Товарные чеки"),
		List: ui.ListView{
			Toolbar: &ui.ToolbarData{
				Buttons: []ui.Button{
					{Style: ui.ButtonPrimary, Text: "Добавить", URL: "/receipts/new", Icon: "plus"},
				},
			},
			Search: &ui.SearchData{URL: "/receipts", Placeholder: "Поиск чеков...", Query: query, Mode: ui.SearchLive},
			List: ui.ListData{
				Columns:    columns,
				Rows:       rows,
				RenderMode: ui.RenderComfortable,
				Preset:     ui.ListWide,
			},
		},
		NewURL: "/receipts/new",
	}

	a.renderListView(w, r, TemplateFS(), pageFS, page)
}

func (a *App) ReceiptCard(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	id := receiptIDFromURL(r)

	var doc *receipts.Document
	if id == 0 {
		rec := a.receipts.New()
		rec.Date = time.Now()
		var items []receipts.ReceiptItem

		if lookup, ok := ui.ReadLookup(r); ok {
			switch lookup.FieldName {
			case lookupCustomer:
				if a.customers != nil {
					cust, err := a.customers.GetByID(r.Context(), lookup.ID)
					if err == nil {
						rec.CustomerID = cust.ID
						rec.CustomerName = cust.Name
					}
				}
			case lookupProduct:
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

	var pickerCustomers []*customers.Customer
	var pickerProducts []*products.Product
	var organizationOptions []pages.ReceiptOrganizationOption

	if doc.Receipt.ID == 0 {
		if a.organizations != nil {
			orgs, err := a.organizations.List(r.Context(), organizations.ListOptions{}, nil)
			if err != nil {
				a.InternalError(w, err)
				return
			}
			organizationOptions = make([]pages.ReceiptOrganizationOption, 0, len(orgs))
			for _, org := range orgs {
				organizationOptions = append(organizationOptions, pages.ReceiptOrganizationOption{ID: org.ID, Name: org.Name})
			}
		}
		if a.customers != nil {
			var err error
			pickerCustomers, err = a.customers.List(r.Context(), 0, customers.ListOptions{}, nil)
			if err != nil {
				a.InternalError(w, err)
				return
			}
		}
		if a.products != nil {
			var err error
			pickerProducts, err = a.products.List(r.Context(), 0, products.ListOptions{}, nil)
			if err != nil {
				a.InternalError(w, err)
				return
			}
		}
	}

	itemsJSON, customersJSON, productsJSON, err := receiptEditorJSON(doc.Items, pickerCustomers, pickerProducts)
	if err != nil {
		a.InternalError(w, err)
		return
	}

	page := pages.ReceiptCardPage{
		Header:         pageHeader(r, "Товарные чеки"),
		Title:          title,
		Receipt:        doc.Receipt,
		Items:          doc.Items,
		CustomerID:     doc.Receipt.CustomerID,
		CustomerName:   doc.Receipt.CustomerName,
		OrganizationID: doc.Receipt.OrganizationID,
		Errors:         make(map[string]string),
		ErrorsJSON:     "{}",
		ItemsJSON:      itemsJSON,
		CustomersJSON:  customersJSON,
		ProductsJSON:   productsJSON,
		Orgs:           organizationOptions,
	}

	pageFS, err := fs.Sub(receipts.Templates(), "card")
	if err != nil {
		a.InternalError(w, err)
		return
	}
	if err := ui.RenderPage(w, TemplateFS(), pageFS, page); err != nil {
		a.InternalError(w, err)
	}
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

	var items []receipts.ReceiptItem
	var total float64
	for i := 0; ; i++ {
		productID := parseInt64(r.FormValue("items[" + strconv.Itoa(i) + "][product_id]"))
		if productID == 0 && i > 0 {
			break
		}
		if productID == 0 {
			continue
		}
		if a.products == nil {
			renderReceiptFormWithErrors(w, r, a, map[string]string{"": "Товар недоступен"}, items)
			return
		}
		product, err := a.products.GetByID(r.Context(), productID)
		if err != nil || product.OrganizationID != orgID {
			renderReceiptFormWithErrors(w, r, a, map[string]string{"": "Товар не принадлежит выбранной организации"}, items)
			return
		}
		quantity := parseFloat(r.FormValue("items[" + strconv.Itoa(i) + "][quantity]"))
		price := parseFloat(r.FormValue("items[" + strconv.Itoa(i) + "][price]"))
		amount := quantity * price
		items = append(items, receipts.ReceiptItem{
			LineNum:   i + 1,
			ProductID: productID,
			Unit:      product.Unit,
			Quantity:  quantity,
			Price:     price,
			Amount:    amount,
		})
		total += amount
	}

	if customerID > 0 && a.customers != nil {
		customer, err := a.customers.GetByID(r.Context(), customerID)
		if err != nil || customer.OrganizationID != orgID {
			renderReceiptFormWithErrors(w, r, a, map[string]string{"customer_id": "Контрагент не принадлежит выбранной организации"}, items)
			return
		}
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
	customerID := parseInt64(r.FormValue("customer_id"))
	organizationID := parseInt64(r.FormValue("organization_id"))
	var customerName string
	if customerID > 0 && a.customers != nil {
		if c, err := a.customers.GetByID(r.Context(), customerID); err == nil {
			customerName = c.Name
		}
	}

	var orgOptions []pages.ReceiptOrganizationOption
	if a.organizations != nil {
		orgs, err := a.organizations.List(r.Context(), organizations.ListOptions{}, nil)
		if err == nil {
			orgOptions = make([]pages.ReceiptOrganizationOption, 0, len(orgs))
			for _, org := range orgs {
				orgOptions = append(orgOptions, pages.ReceiptOrganizationOption{ID: org.ID, Name: org.Name})
			}
		}
	}

	var pickerCustomers []*customers.Customer
	if a.customers != nil {
		pickerCustomers, _ = a.customers.List(r.Context(), 0, customers.ListOptions{}, nil)
	}
	var pickerProducts []*products.Product
	if a.products != nil {
		pickerProducts, _ = a.products.List(r.Context(), 0, products.ListOptions{}, nil)
	}
	itemsJSON, customersJSON, productsJSON, err := receiptEditorJSON(items, pickerCustomers, pickerProducts)
	if err != nil {
		a.InternalError(w, err)
		return
	}

	receipt := a.receipts.New()
	receipt.Number = r.FormValue("number")
	receipt.OrganizationID = organizationID
	receipt.CustomerID = customerID
	receipt.CustomerName = customerName
	if date, err := time.Parse("2006-01-02", r.FormValue("date")); err == nil {
		receipt.Date = date
	} else {
		receipt.Date = time.Now()
	}

	page := pages.ReceiptCardPage{
		Header:         pageHeader(r, "Товарные чеки"),
		Title:          "Новый товарный чек",
		Receipt:        receipt,
		Errors:         errs,
		OrganizationID: organizationID,
		CustomerID:     customerID,
		CustomerName:   customerName,
		Items:          items,
		ItemsJSON:      itemsJSON,
		CustomersJSON:  customersJSON,
		ProductsJSON:   productsJSON,
		Orgs:           orgOptions,
	}
	page.ErrorsJSON, _ = common.ToJSON(errs)
	if page.ErrorsJSON == "" {
		page.ErrorsJSON = "{}"
	}
	pageFS, err := fs.Sub(receipts.Templates(), "card")
	if err != nil {
		a.InternalError(w, err)
		return
	}
	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := ui.RenderPage(w, TemplateFS(), pageFS, page); err != nil {
		a.InternalError(w, err)
	}
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
