package app

import (
	"context"
	"errors"
	"io/fs"
	"mime"
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
	"Orders/internal/sessions"
	"Orders/internal/ui"

	"github.com/go-chi/chi/v5"
)

// lookupCustomer / lookupProduct — значения select_field в callback-ссылках
// пикера. Принадлежат приложению (потребителю ReadLookup), а не библиотеке ui.
const (
	lookupCustomer = "customer"
	lookupProduct  = "product"

	receiptFromEdit = "edit"
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

// receiptReturnURL определяет, куда вернуть пользователя после отмены
// подтверждения отправки. Источник передаётся через параметр from;
// всё, кроме «edit», означает возврат в список документов.
func receiptReturnURL(doc *receipts.Document, from string) string {
	if from == receiptFromEdit {
		return doc.Receipt.URL()
	}
	return RouteReceipts
}

// renderReceiptSendConfirmPage отображает экран подтверждения отправки
// (mode=send): читает режим из файловой системы card и рендерит страницу.
// alert может быть nil.
func (a *App) renderReceiptSendConfirmPage(w http.ResponseWriter, r *http.Request, doc *receipts.Document, from string, alert *ui.AlertData) {
	returnURL := receiptReturnURL(doc, from)
	page := buildReceiptSendConfirmPage(pageHeader(r, "Товарные чеки"), doc, returnURL, alert)

	pageFS, err := fs.Sub(receipts.Templates(), "card")
	if err != nil {
		a.InternalError(w, r, err)
		return
	}
	if err := ui.RenderPage(w, TemplateFS(), pageFS, page); err != nil {
		a.InternalError(w, r, err)
	}
}

// buildReceiptSendConfirmPage — единственный путь построения экрана
// подтверждения отправки. Используется и GET (mode=send), и POST /send
// при ошибке. Alert может быть nil — ошибки нет.
func buildReceiptSendConfirmPage(header ui.HeaderData, doc *receipts.Document, returnURL string, alert *ui.AlertData) pages.ReceiptSendConfirmPage {
	title := "Товарный чек №" + doc.Receipt.Number
	confirmable := doc.Receipt.ID > 0 && doc.Receipt.SentAt == nil
	return pages.ReceiptSendConfirmPage{
		ReceiptCardPage: pages.ReceiptCardPage{
			Header:     header,
			Alert:      alert,
			CanSend:    confirmable,
			Title:      title,
			FormAction: doc.Receipt.URL(),
			Card:       ui.CardData{Title: title, CloseURL: returnURL},
			Receipt:    doc.Receipt,
			Items:      doc.Items,
		},
		CanConfirmSend: confirmable,
		ReturnURL:      returnURL,
	}
}

func (a *App) ReceiptsPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	query := strings.TrimSpace(r.URL.Query().Get("q"))

	fields := receipts.Descriptor.ListFields()
	visibleFields := entity.Names(fields)

	list, err := a.receipts.List(r.Context(), receipts.ListOptions{Query: query}, visibleFields)
	if err != nil {
		a.InternalError(w, r, err)
		return
	}

	rows := make([]pages.ReceiptListRow, 0, len(list))
	for _, rec := range list {
		total, err := rec.DisplayValue("Total")
		if err != nil {
			a.InternalError(w, r, err)
			return
		}
		status, err := rec.DisplayValue("Status")
		if err != nil {
			a.InternalError(w, r, err)
			return
		}

		sent := rec.SentAt != nil
		base := rec.URL()
		idStr := strconv.FormatInt(rec.ID, 10)

		rows = append(rows, pages.ReceiptListRow{
			Number:       rec.Number,
			Date:         rec.Date.Format("02.01.2006"),
			Organization: rec.OrganizationName,
			Customer:     rec.CustomerName,
			Total:        total,
			Status:       status,

			CanEdit: !sent,
			CanSend: !sent,

			FilesURL: "/receipts/" + idStr + "/files",
			CopyURL:  "/receipts/" + idStr + "/copy",
			SendURL:  base + "?mode=send",
			ViewURL:  base + "?mode=view",
			EditURL:  base,
		})
	}

	page := pages.ReceiptsListPage{
		Page:    pages.Page{Title: "Товарные чеки"},
		Header:  pageHeader(r, "Товарные чеки"),
		Toolbar: &ui.ToolbarData{
			Buttons: []ui.Button{
				{Style: ui.ButtonPrimary, Text: "Добавить", URL: "/receipts/new", Icon: "plus"},
			},
		},
		Search: &ui.SearchData{URL: RouteReceipts, Placeholder: "Поиск чеков...", Query: query, Mode: ui.SearchLive},
		Rows:   rows,
		NewURL: "/receipts/new",
	}

	if flash, err := a.consumeFlash(r); err != nil {
		a.InternalError(w, r, err)
		return
	} else if flash != nil {
		page.Alert = FlashToAlert(*flash)
	}

	pageFS, err := fs.Sub(receipts.Templates(), "list")
	if err != nil {
		a.InternalError(w, r, err)
		return
	}

	if ResponseModeFromRequest(r) == Fragment {
		if err := ui.Render(w, TemplateFS(), pageFS, "receipts_list", page); err != nil {
			a.InternalError(w, r, err)
		}
		return
	}
	if err := ui.RenderPage(w, TemplateFS(), pageFS, page); err != nil {
		a.InternalError(w, r, err)
	}
}

func (a *App) ReceiptCard(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	const (
		receiptModeView = "view"
		receiptModeSend = "send"
	)

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
			a.InternalError(w, r, err)
			return
		}
	}

	title := "Товарный чек №" + doc.Receipt.Number

	mode := r.URL.Query().Get("mode")
	isView := mode == receiptModeView
	isSend := mode == receiptModeSend
	sent := doc.Receipt.SentAt != nil

	canEdit := doc.Receipt.ID == 0 || (!sent && !isView && !isSend)
	canSend := isSend && !sent && doc.Receipt.ID > 0

	formAction := RouteReceipts
	if doc.Receipt.ID > 0 {
		formAction = "/receipts/" + strconv.FormatInt(doc.Receipt.ID, 10)
	}

	if isSend {
		a.renderReceiptSendConfirmPage(w, r, doc, r.URL.Query().Get("from"), nil)
		return
	}

	var pickerCustomers []*customers.Customer
	var pickerProducts []*products.Product
	var organizationOptions []pages.ReceiptOrganizationOption

	if canEdit {
		if a.organizations != nil {
			orgs, err := a.organizations.List(r.Context(), organizations.ListOptions{}, nil)
			if err != nil {
				a.InternalError(w, r, err)
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
				a.InternalError(w, r, err)
				return
			}
		}
		if a.products != nil {
			var err error
			pickerProducts, err = a.products.List(r.Context(), 0, products.ListOptions{}, nil)
			if err != nil {
				a.InternalError(w, r, err)
				return
			}
		}
	}

	itemsJSON, customersJSON, productsJSON, err := receiptEditorJSON(doc.Items, pickerCustomers, pickerProducts)
	if err != nil {
		a.InternalError(w, r, err)
		return
	}

	page := pages.ReceiptCardPage{
		Header:         pageHeader(r, "Товарные чеки"),
		CanEdit:        canEdit,
		CanSend:        canSend,
		Title:          title,
		FormAction:     formAction,
		Card:           ui.CardData{Title: title, CloseURL: RouteReceipts},
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
		a.InternalError(w, r, err)
		return
	}
	if err := ui.RenderPage(w, TemplateFS(), pageFS, page); err != nil {
		a.InternalError(w, r, err)
	}
}

// ReceiptCopyPage открывает редактор нового чека «на основании» документа id.
// Данные исходника переносятся в новый документ (ID=0, без отправки);
// сохранение создаёт новый чек. Строки копируются без идентификаторов.
func (a *App) ReceiptCopyPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	src, err := a.receipts.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, receipts.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		a.InternalError(w, r, err)
		return
	}

	rec := a.receipts.New()
	rec.Date = time.Now()
	rec.OrganizationID = src.Receipt.OrganizationID
	rec.UserID = src.Receipt.UserID
	rec.CustomerID = src.Receipt.CustomerID
	rec.CustomerName = src.Receipt.CustomerName

	items := make([]receipts.ReceiptItem, 0, len(src.Items))
	for _, it := range src.Items {
		items = append(items, receipts.ReceiptItem{
			LineNum:     it.LineNum,
			ProductID:   it.ProductID,
			ProductName: it.ProductName,
			Unit:        it.Unit,
			Quantity:    it.Quantity,
			Price:       it.Price,
			Amount:      it.Amount,
		})
	}

	a.renderReceiptEditorPage(w, r, &receipts.Document{Receipt: rec, Items: items}, src.Receipt.Number)
}

// renderReceiptEditorPage отображает редактор чека (карточку) с готовой
// к правке моделью. sourceNumber при непустом значении показывает баннер
// «Создан на основании».
func (a *App) renderReceiptEditorPage(w http.ResponseWriter, r *http.Request, doc *receipts.Document, sourceNumber string) {
	ctx := r.Context()

	var organizationOptions []pages.ReceiptOrganizationOption
	if a.organizations != nil {
		orgs, err := a.organizations.List(ctx, organizations.ListOptions{}, nil)
		if err != nil {
			a.InternalError(w, r, err)
			return
		}
		organizationOptions = make([]pages.ReceiptOrganizationOption, 0, len(orgs))
		for _, org := range orgs {
			organizationOptions = append(organizationOptions, pages.ReceiptOrganizationOption{ID: org.ID, Name: org.Name})
		}
	}

	var pickerCustomers []*customers.Customer
	if a.customers != nil {
		var err error
		pickerCustomers, err = a.customers.List(ctx, 0, customers.ListOptions{}, nil)
		if err != nil {
			a.InternalError(w, r, err)
			return
		}
	}

	var pickerProducts []*products.Product
	if a.products != nil {
		var err error
		pickerProducts, err = a.products.List(ctx, 0, products.ListOptions{}, nil)
		if err != nil {
			a.InternalError(w, r, err)
			return
		}
	}

	itemsJSON, customersJSON, productsJSON, err := receiptEditorJSON(doc.Items, pickerCustomers, pickerProducts)
	if err != nil {
		a.InternalError(w, r, err)
		return
	}

	card := pages.ReceiptCopyPage{
		ReceiptCardPage: pages.ReceiptCardPage{
			Header:         pageHeader(r, "Товарные чеки"),
			CanEdit:        true,
			Title:          "Новый товарный чек",
			FormAction:     RouteReceipts,
			Card:           ui.CardData{Title: "Новый товарный чек", CloseURL: RouteReceipts},
			Receipt:        doc.Receipt,
			Items:          doc.Items,
			CustomerID:     doc.Receipt.CustomerID,
			CustomerName:   doc.Receipt.CustomerName,
			OrganizationID: doc.Receipt.OrganizationID,
			ItemsJSON:      itemsJSON,
			CustomersJSON:  customersJSON,
			ProductsJSON:   productsJSON,
			Orgs:           organizationOptions,
			Errors:         make(map[string]string),
			ErrorsJSON:     "{}",
		},
	}
	if sourceNumber != "" {
		card.CopySource = sourceNumber
	}

	pageFS, err := fs.Sub(receipts.Templates(), "card")
	if err != nil {
		a.InternalError(w, r, err)
		return
	}
	if err := ui.RenderPage(w, TemplateFS(), pageFS, card); err != nil {
		a.InternalError(w, r, err)
	}
}

// ReceiptFiles отображает окно «Файлы» чека: полная страница или Фрагмент
// (shell модального окна) в зависимости от ResponseModeFromRequest.
func (a *App) ReceiptFiles(w http.ResponseWriter, r *http.Request) {
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
		a.InternalError(w, r, err)
		return
	}

	rec := doc.Receipt
	header := pages.ReceiptHeader{
		Number:       rec.Number,
		Date:         rec.Date.Format("2006-01-02"),
		Organization: rec.OrganizationName,
	}
	if total, err := rec.DisplayValue("Total"); err == nil {
		header.Total = total
	}

	files, err := a.receiptFiles.ListByReceipt(r.Context(), id)
	if err != nil {
		a.InternalError(w, r, err)
		return
	}

	fileViews := make([]pages.ReceiptFile, 0, len(files))
	for _, f := range files {
		fileViews = append(fileViews, pages.ReceiptFile{
			Name: f.FileName,
			Icon: "file-text",
			URL:  "/receipts/" + idStr + "/files/" + strconv.FormatInt(f.ID, 10),
		})
	}

	filesPage := pages.ReceiptFilesPage{
		Page:    pages.Page{Title: "Файлы чека №" + rec.Number},
		Header:  pageHeader(r, "Товарные чеки"),
		Receipt: header,
		Files:   fileViews,
		BackURL: RouteReceipts,
	}

	filesFS, err := fs.Sub(receipts.Templates(), "files")
	if err != nil {
		a.InternalError(w, r, err)
		return
	}

	if ResponseModeFromRequest(r) == Fragment {
		if err := ui.Render(w, TemplateFS(), filesFS, "receipts_files_modal", filesPage); err != nil {
			a.InternalError(w, r, err)
		}
		return
	}
	if err := ui.RenderPage(w, TemplateFS(), filesFS, filesPage); err != nil {
		a.InternalError(w, r, err)
	}
}

// ReceiptFileContent отдаёт содержимое файла для открытия в браузере.
// Проверяются оба идентификатора: файл обязан принадлежать документу id,
// иначе нельзя открыть файл одного чека через другой. Заголовок
// Content-Disposition строится безопасно через mime.FormatMediaType: имя
// приходит из внешней системы и не должно влиять на заголовок ответа.
func (a *App) ReceiptFileContent(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	id := receiptIDFromURL(r)
	if id == 0 {
		http.NotFound(w, r)
		return
	}

	fileIDStr := chi.URLParam(r, "fileID")
	fileID, err := strconv.ParseInt(fileIDStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	file, err := a.receiptFiles.GetByID(r.Context(), id, fileID)
	if err != nil {
		if errors.Is(err, receipts.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		a.InternalError(w, r, err)
		return
	}

	contentType := file.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(file.Data)))
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": file.FileName}))
	if _, err := w.Write(file.Data); err != nil {
		a.log.Error().Err(err).Msg("receipt file: write failed")
	}
}

func (a *App) ReceiptSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.BadRequest(w, "Invalid request")
		return
	}

	id := receiptIDFromURL(r)

	var existing *receipts.Document
	if id > 0 {
		var err error
		existing, err = a.receipts.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, receipts.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			a.InternalError(w, r, err)
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
			ve := NewValidationError("Ошибка документа").Add("Товар недоступен")
			a.RenderReceiptValidationError(w, r, ve, items)
			return
		}
		product, err := a.products.GetByID(r.Context(), productID)
		if err != nil || product.OrganizationID != orgID {
			ve := NewValidationError("Ошибка документа").Add("Товар не принадлежит выбранной организации")
			a.RenderReceiptValidationError(w, r, ve, items)
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
			ve := NewValidationError("Ошибка документа").AddField("customer_id", "Контрагент не принадлежит выбранной организации")
			a.RenderReceiptValidationError(w, r, ve, items)
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

	sendTo1C := r.FormValue("send_to_1c") == "1"
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
	if existing != nil {
		rec.UUID = existing.Receipt.UUID
		rec.ExchangeID = existing.Receipt.ExchangeID
	}

	if id == 0 {
		if orgID == 0 || customerID == 0 {
			ve := NewValidationError("Ошибка документа")
			if orgID == 0 {
				ve.AddField("organization_id", "Выберите организацию")
			}
			if customerID == 0 {
				ve.AddField("customer_id", "Выберите клиента")
			}
			a.RenderReceiptValidationError(w, r, ve, items)
			return
		}

		uuid, err := common.GenerateUUID()
		if err != nil {
			a.InternalError(w, r, err)
			return
		}
		rec.ExchangeID = uuid
	}

	doc := &receipts.Document{Receipt: rec, Items: items}
	if err := a.receipts.Save(r.Context(), doc); err != nil {
		a.InternalError(w, r, err)
		return
	}

	if sendTo1C {
		sendURL := doc.Receipt.URL() + "?mode=send&from=" + receiptFromEdit
		if ResponseModeFromRequest(r) == Fragment {
			w.Header().Set("HX-Redirect", sendURL)
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, sendURL, http.StatusSeeOther)
		return
	}

	if ResponseModeFromRequest(r) == Fragment {
		w.Header().Set("HX-Redirect", RouteReceipts)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, RouteReceipts, http.StatusSeeOther)
}

// RenderReceiptValidationError доставляет ошибки валидации чека в зависимости
// от режима ответа: Fragment — транспортное представление ValidationError,
// FullPage — форма с ошибками и статусом 422.
func (a *App) RenderReceiptValidationError(w http.ResponseWriter, r *http.Request, ve *ValidationError, items []receipts.ReceiptItem) {
	if ResponseModeFromRequest(r) == Fragment {
		WriteValidationResponse(w, r, NewValidationResponse(*ve))
		return
	}
	a.renderReceiptForm(w, r, ve, items)
}

func (a *App) renderReceiptForm(w http.ResponseWriter, r *http.Request, ve *ValidationError, items []receipts.ReceiptItem) {
	id := receiptIDFromURL(r)
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
		a.InternalError(w, r, err)
		return
	}

	receipt := a.receipts.New()
	receipt.ID = id
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
		CanEdit:        true,
		Title:          "Новый товарный чек",
		FormAction:     RouteReceipts,
		Card:           ui.CardData{Title: "Новый товарный чек", CloseURL: RouteReceipts},
		Receipt:        receipt,
		Errors:         ve.ErrorsMap(),
		OrganizationID: organizationID,
		CustomerID:     customerID,
		CustomerName:   customerName,
		Items:          items,
		ItemsJSON:      itemsJSON,
		CustomersJSON:  customersJSON,
		ProductsJSON:   productsJSON,
		Orgs:           orgOptions,
	}
	if id > 0 {
		page.FormAction = "/receipts/" + strconv.FormatInt(id, 10)
	}
	page.ErrorsJSON, _ = common.ToJSON(ve.ErrorsMap())
	if page.ErrorsJSON == "" {
		page.ErrorsJSON = "{}"
	}
	pageFS, err := fs.Sub(receipts.Templates(), "card")
	if err != nil {
		a.InternalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := ui.RenderPage(w, TemplateFS(), pageFS, page); err != nil {
		a.InternalError(w, r, err)
	}
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
		a.InternalError(w, r, err)
		return
	}
	if existing.Receipt.SentAt != nil {
		a.BadRequest(w, receipts.ErrReceiptReadOnly.Error())
		return
	}

	if err := a.receipts.DeleteByID(r.Context(), id); err != nil {
		a.InternalError(w, r, err)
		return
	}

	http.Redirect(w, r, RouteReceipts, http.StatusSeeOther)
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
		a.InternalError(w, r, err)
		return
	}

	if doc.Receipt.SentAt != nil {
		a.BadRequest(w, receipts.ErrReceiptReadOnly.Error())
		return
	}

	if err := sendReceiptTo1C(r.Context(), doc); err != nil {
		// Остаться на экране подтверждения: SentAt не меняется, кнопки
		// отправки и отмены остаются доступны.
		alert := FlashToAlert(sessions.Flash{Type: sessions.FlashError, Message: err.Error()})
		a.renderReceiptSendConfirmPage(w, r, doc, r.URL.Query().Get("from"), alert)
		return
	}

	now := time.Now()
	doc.Receipt.SentAt = &now
	doc.Receipt.UpdatedAt = now

	if err := a.receipts.Save(r.Context(), doc); err != nil {
		a.InternalError(w, r, err)
		return
	}

	if err := a.SetFlash(r, sessions.FlashSuccess, "Документ успешно отправлен в 1С."); err != nil {
		a.InternalError(w, r, err)
		return
	}

	http.Redirect(w, r, RouteReceipts, http.StatusSeeOther)
}

// sendReceiptTo1C выполняет отправку документа в 1С. Сейчас — заглушка,
// всегда успешна. SentAt устанавливается только после успешной отправки.
func sendReceiptTo1C(ctx context.Context, doc *receipts.Document) error {
	return nil
}

func parseInt64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
