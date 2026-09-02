package app

import (
	"context"
	"errors"
	"io/fs"
	"math"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"Orders/internal/app/pages"
	"Orders/internal/common"
	"Orders/internal/customers"
	"Orders/internal/database/search"
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
	returnURL := a.URL(receiptReturnURL(doc, from))
	formAction := a.URL(doc.Receipt.URL())
	page := buildReceiptSendConfirmPage(a.pageHeader(r, "Товарные чеки"), doc, formAction, returnURL, alert)

	pageFS, err := fs.Sub(receipts.Templates(), "card")
	if err != nil {
		a.InternalError(w, r, err)
		return
	}
	if err := ui.RenderPage(w, TemplateFS(), pageFS, a.basePath(), page); err != nil {
		a.InternalError(w, r, err)
	}
}

// buildReceiptSendConfirmPage — единственный путь построения экрана
// подтверждения отправки. Используется и GET (mode=send), и POST /send
// при ошибке. Alert может быть nil — ошибки нет.
func buildReceiptSendConfirmPage(header ui.HeaderData, doc *receipts.Document, formAction, returnURL string, alert *ui.AlertData) pages.ReceiptSendConfirmPage {
	title := "Товарный чек №" + doc.Receipt.Number
	confirmable := doc.Receipt.ID > 0 && doc.Receipt.SentAt == nil
	return pages.ReceiptSendConfirmPage{
		ReceiptCardPage: pages.ReceiptCardPage{
			Header:     header,
			Alert:      alert,
			CanSend:    confirmable,
			Title:      title,
			FormAction: formAction,
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

	limit, err := receiptListLimit(r)
	if err != nil {
		a.BadRequest(w, err.Error())
		return
	}
	after, err := parseReceiptAfter(r.URL.Query().Get("after"))
	if err != nil {
		a.BadRequest(w, err.Error())
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	filter := parseReceiptFilter(r)

	fields := receipts.Descriptor.ListFields()
	visibleFields := entity.Names(fields)

	// Справочники для пикеров панели фильтра: один запрос на справочник
	// (без N+1). Те же данные используются для валидации связи
	// организация → контрагент и для имён выбранных значений.
	var orgs []*organizations.Organization
	if a.organizations != nil {
		var err error
		orgs, err = a.organizations.List(r.Context(), organizations.ListOptions{}, nil)
		if err != nil {
			a.InternalError(w, r, err)
			return
		}
	}
	var custs []*customers.Customer
	if a.customers != nil {
		var err error
		custs, err = a.customers.List(r.Context(), 0, customers.ListOptions{}, nil)
		if err != nil {
			a.InternalError(w, r, err)
			return
		}
	}
	filter = validateReceiptFilterPair(filter, custs)

	// Только одна порция (keyset-пагинация). Поиск и фильтры применяются
	// в SQL до LIMIT; связанные данные считаются только для этой порции.
	listPage, err := a.receipts.ListPage(r.Context(), receipts.ListOptions{
		Query:  query,
		Limit:  limit,
		After:  after,
		Filter: filter.storeFilter(),
	}, visibleFields)
	if err != nil {
		a.InternalError(w, r, err)
		return
	}

	words := search.NormalizeQuery(query).Words

	rows, err := a.buildReceiptListRows(r.Context(), listPage.Items, words, CurrentUser(r).IsAdmin)
	if err != nil {
		a.InternalError(w, r, err)
		return
	}

	searchData := &ui.SearchData{
		URL:         a.URL(RouteReceipts),
		Placeholder: "Поиск чеков...",
		Query:       query,
		Mode:        ui.SearchLive,
		TargetID:    "#receipts-browser",
	}
	if a.organizations != nil || a.customers != nil {
		searchData.Filter = buildFilterData(filter, orgs, custs)
	}

	page := pages.ReceiptsListPage{
		Page:   pages.Page{Title: "Товарные чеки"},
		Header: a.pageHeader(r, "Товарные чеки"),
		Toolbar: &ui.ToolbarData{
			Buttons: []ui.Button{
				{Style: ui.ButtonPrimary, Text: "Добавить", URL: a.URL("/receipts/new"), Icon: "plus"},
			},
		},
		Search: searchData,
		Rows:   rows,
		NewURL: a.URL("/receipts/new"),
	}
	if listPage.HasMore {
		page.HasMore = true
		page.LoadMoreURL = a.receiptListLoadMoreURL(query, filter, limit, *listPage.Next)
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

	if ResponseModeFromRequest(r) == Fragment && r.URL.Query().Get("part") == "rows" {
		if err := ui.Render(w, TemplateFS(), pageFS, a.basePath(), "receipts_rows", page); err != nil {
			a.InternalError(w, r, err)
		}
		return
	}
	if ResponseModeFromRequest(r) == Fragment {
		if err := ui.Render(w, TemplateFS(), pageFS, a.basePath(), "receipts_browser", page); err != nil {
			a.InternalError(w, r, err)
		}
		return
	}
	if err := ui.RenderPage(w, TemplateFS(), pageFS, a.basePath(), page); err != nil {
		a.InternalError(w, r, err)
	}
}

// buildReceiptListRows собирает готовые строки списка чеков. Связанные
// данные (fileCounts из отдельной базы files.db) вычисляются одним запросом
// строго для переданной порции документов, без N+1 и без выборок для
// всей таблицы.
func (a *App) buildReceiptListRows(ctx context.Context, list []*receipts.Receipt, words []string, isAdmin bool) ([]pages.ReceiptListRow, error) {
	rows := make([]pages.ReceiptListRow, 0, len(list))
	if len(list) == 0 {
		return rows, nil
	}

	ids := make([]int64, len(list))
	for i, rec := range list {
		ids[i] = rec.ID
	}

	var fileCounts map[int64]int
	if a.receiptFiles != nil {
		var err error
		fileCounts, err = a.receiptFiles.CountByReceipts(ctx, ids)
		if err != nil {
			return nil, err
		}
	}

	for _, rec := range list {
		total, err := rec.DisplayValue("Total")
		if err != nil {
			return nil, err
		}

		presentation := statusPresentation(rec.Status, rec.CreatedAt, rec.SentAt)
		sent := rec.SentAt != nil
		base := a.URL(rec.URL())
		idStr := strconv.FormatInt(rec.ID, 10)

		info := ""
		if rec.Action != "" && rec.ActionReceivedAt == nil {
			info = rec.Action
		}

		rows = append(rows, pages.ReceiptListRow{
			Number:       ui.MarkMatches(rec.Number, words),
			Date:         rec.Date.Format("02.01.2006"),
			Organization: ui.MarkMatches(rec.OrganizationName, words),
			Customer:     ui.MarkMatches(rec.CustomerName, words),
			Total:        total,
			Status:       presentation.Display,
			StatusKey:    string(presentation.StatusKey),

			Info: info,

			CanEdit: !sent,
			CanSend: !sent,

			HasFiles: fileCounts[rec.ID] > 0,

			FilesURL: a.URL("/receipts/" + idStr + "/files"),
			CopyURL:  a.URL("/receipts/" + idStr + "/copy"),
			SendURL:  base + "?mode=send",
			ViewURL:  base + "?mode=view",
			EditURL:  base,

			CanSendAction: rec.UUID != "",
			ActionURL:     a.URL("/receipts/" + idStr + "/action"),

			CanMarkDeleted: isAdmin && receipts.ReceiptDeletable(rec.Status),
			DeleteURL:      a.URL("/receipts/" + idStr + "/delete"),
			DeleteConfirm:  "Пометить товарный чек №" + rec.Number + " на удаление?",
		})
	}
	return rows, nil
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

	formAction := a.URL(RouteReceipts)
	if doc.Receipt.ID > 0 {
		formAction = a.URL("/receipts/" + strconv.FormatInt(doc.Receipt.ID, 10))
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

	// Файлы документа нужны только в режиме просмотра (read-only):
	// один запрос к files.db, без N+1. В режиме правки блок файлов
	// не показывается.
	var fileViews []pages.ReceiptFile
	if !canEdit && doc.Receipt.ID > 0 && a.receiptFiles != nil {
		files, err := a.receiptFiles.ListByReceipt(r.Context(), doc.Receipt.ID)
		if err != nil {
			a.InternalError(w, r, err)
			return
		}
		if len(files) > 0 {
			idStr := strconv.FormatInt(doc.Receipt.ID, 10)
			fileViews = make([]pages.ReceiptFile, 0, len(files))
			for _, f := range files {
				fileViews = append(fileViews, pages.ReceiptFile{
					Name: f.FileName,
					Icon: "file-text",
					URL:  a.URL("/receipts/" + idStr + "/files/" + strconv.FormatInt(f.ID, 10)),
				})
			}
		}
	}

	page := pages.ReceiptCardPage{
		Header:         a.pageHeader(r, "Товарные чеки"),
		CanEdit:        canEdit,
		CanSend:        canSend,
		Title:          title,
		FormAction:     formAction,
		Card:           ui.CardData{Title: title, CloseURL: a.URL(RouteReceipts)},
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
		Files:          fileViews,
	}

	pageFS, err := fs.Sub(receipts.Templates(), "card")
	if err != nil {
		a.InternalError(w, r, err)
		return
	}
	if err := ui.RenderPage(w, TemplateFS(), pageFS, a.basePath(), page); err != nil {
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
			Header:         a.pageHeader(r, "Товарные чеки"),
			CanEdit:        true,
			Title:          "Новый товарный чек",
			FormAction:     a.URL(RouteReceipts),
			Card:           ui.CardData{Title: "Новый товарный чек", CloseURL: a.URL(RouteReceipts)},
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
	if err := ui.RenderPage(w, TemplateFS(), pageFS, a.basePath(), card); err != nil {
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
			URL:  a.URL("/receipts/" + idStr + "/files/" + strconv.FormatInt(f.ID, 10)),
		})
	}

	filesPage := pages.ReceiptFilesPage{
		Page:    pages.Page{Title: "Файлы чека №" + rec.Number},
		Header:  a.pageHeader(r, "Товарные чеки"),
		Receipt: header,
		Files:   fileViews,
		BackURL: a.URL(RouteReceipts),
	}

	filesFS, err := fs.Sub(receipts.Templates(), "files")
	if err != nil {
		a.InternalError(w, r, err)
		return
	}

	if ResponseModeFromRequest(r) == Fragment {
		if err := ui.Render(w, TemplateFS(), filesFS, a.basePath(), "receipts_files_modal", filesPage); err != nil {
			a.InternalError(w, r, err)
		}
		return
	}
	if err := ui.RenderPage(w, TemplateFS(), filesFS, a.basePath(), filesPage); err != nil {
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
		quantity := round3(parseFloat(r.FormValue("items[" + strconv.Itoa(i) + "][quantity]")))
		price := round2(parseFloat(r.FormValue("items[" + strconv.Itoa(i) + "][price]")))
		// amount — самостоятельное вводимое значение: переданная сумма
		// сохраняется (после округления), а не заменяется расчётным
		// quantity × price. Фолбэк round2(quantity * price) применяется
		// только для legacy-запросов без поля amount.
		amountStr := r.FormValue("items[" + strconv.Itoa(i) + "][amount]")
		amount := round2(parseFloat(amountStr))
		if amountStr == "" {
			amount = round2(quantity * price)
		}
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
		sendURL := a.URL(doc.Receipt.URL()) + "?mode=send&from=" + receiptFromEdit
		if ResponseModeFromRequest(r) == Fragment {
			w.Header().Set("HX-Redirect", sendURL)
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, sendURL, http.StatusSeeOther)
		return
	}

	if ResponseModeFromRequest(r) == Fragment {
		w.Header().Set("HX-Redirect", a.URL(RouteReceipts))
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, a.URL(RouteReceipts), http.StatusSeeOther)
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
		Header:         a.pageHeader(r, "Товарные чеки"),
		CanEdit:        true,
		Title:          "Новый товарный чек",
		FormAction:     a.URL(RouteReceipts),
		Card:           ui.CardData{Title: "Новый товарный чек", CloseURL: a.URL(RouteReceipts)},
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
		page.FormAction = a.URL("/receipts/" + strconv.FormatInt(id, 10))
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
	if err := ui.RenderPage(w, TemplateFS(), pageFS, a.basePath(), page); err != nil {
		a.InternalError(w, r, err)
	}
}

// ReceiptMarkDeleted помечает документ на удаление (этап 1 жизненного
// цикла удаления). Маршрут покрыт RequireAdmin: обработчик доступен только
// администратору. Пометка атомарно приводит статус к StatusCancelled,
// снимает отправку (sent_at = NULL) и сохраняет признак deleted_at.
// Физическое удаление документа и его файлов выполняется отдельными
// этапами позже.
func (a *App) ReceiptMarkDeleted(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	err = a.receipts.MarkDeleted(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, receipts.ErrNotFound):
			http.NotFound(w, r)
		case errors.Is(err, receipts.ErrReceiptNotDeletable):
			http.Error(w, err.Error(), http.StatusForbidden)
		default:
			a.InternalError(w, r, err)
		}
		return
	}

	if err := a.SetFlash(r, sessions.FlashSuccess, "Документ помечен на удаление."); err != nil {
		a.InternalError(w, r, err)
		return
	}

	http.Redirect(w, r, a.URL(RouteReceipts), http.StatusSeeOther)
}

// ReceiptActionDialog отображает модальное окно «Действие» документа
// (фрагмент для data-dialog-url). Доступно только для синхронизированных
// в 1С документов (uuid != "").
func (a *App) ReceiptActionDialog(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	id := receiptIDFromURL(r)
	if id == 0 {
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
	if doc.Receipt.UUID == "" {
		http.NotFound(w, r)
		return
	}

	currentAction, _, err := a.receipts.GetAction(r.Context(), id)
	if err != nil {
		a.InternalError(w, r, err)
		return
	}

	page := pages.ReceiptActionPage{
		Number:        doc.Receipt.Number,
		CurrentAction: currentAction,
		FormAction:    a.URL("/receipts/" + strconv.FormatInt(id, 10) + "/action"),
	}

	pageFS, err := fs.Sub(receipts.Templates(), "action")
	if err != nil {
		a.InternalError(w, r, err)
		return
	}
	if err := ui.Render(w, TemplateFS(), pageFS, a.basePath(), "receipts_action_modal", page); err != nil {
		a.InternalError(w, r, err)
	}
}

// ReceiptActionSave устанавливает или очищает действие документа.
// Доступно только для синхронизированных документов. Пустое значение
// (кнопка «Отмена») очищает ранее установленное действие (если оно было).
func (a *App) ReceiptActionSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.BadRequest(w, "Invalid request")
		return
	}

	id := receiptIDFromURL(r)
	if id == 0 {
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
	if doc.Receipt.UUID == "" {
		http.NotFound(w, r)
		return
	}

	action := r.FormValue("action")
	if action != "" && !receipts.ValidAction(action) {
		a.BadRequest(w, "invalid action")
		return
	}

	if err := a.receipts.SetAction(r.Context(), id, action); err != nil {
		a.InternalError(w, r, err)
		return
	}

	message := "Действие отменено."
	if action != "" {
		message = "Действие установлено: " + action + "."
	}
	if err := a.SetFlash(r, sessions.FlashSuccess, message); err != nil {
		a.InternalError(w, r, err)
		return
	}

	http.Redirect(w, r, a.URL(RouteReceipts), http.StatusSeeOther)
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

	http.Redirect(w, r, a.URL(RouteReceipts), http.StatusSeeOther)
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

// round2 округляет значение до двух знаков после запятой.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// round3 округляет значение до трёх знаков после запятой (количество).
func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}
