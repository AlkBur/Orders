package app

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"Orders/internal/common"
	"Orders/internal/customers"
	"Orders/internal/organizations"
	"Orders/internal/receipts"
	"Orders/internal/ui"
)

// receiptFilterStatuses — фиксированный список статусов панели фильтра
// в порядке отображения. «Просрочен» — производное состояние: в базе
// не существует и определяется тем же механизмом, что и отображение
// статуса в списке (statusPresentation).
var receiptFilterStatuses = []string{
	receipts.StatusCreated,
	receipts.OverdueStatus,
	receipts.StatusSent,
	receipts.StatusAccepted,
	receipts.StatusCancelled,
	receipts.StatusProcessed,
	receipts.StatusFinished,
}

// receiptFilter — результат разбора query parameters панели расширенного
// отбора списка чеков. Пустые значения полей означают отсутствие отбора.
type receiptFilter struct {
	dateFrom      string
	dateTo        string
	amountFrom    *float64
	amountTo      *float64
	orgID         int64
	custID        int64
	status        string
	actionSetFrom string
	actionSetTo   string
}

// active сообщает, установлен ли хотя бы один параметр расширенного
// фильтра. Текстовый поиск (query параметр q) сюда не входит.
func (f receiptFilter) active() bool {
	return f.dateFrom != "" || f.dateTo != "" ||
		f.amountFrom != nil || f.amountTo != nil ||
		f.orgID > 0 || f.custID > 0 || f.status != "" ||
		f.actionSetFrom != "" || f.actionSetTo != ""
}

// storeFilter преобразует разобранный фильтр в условия хранилища.
func (f receiptFilter) storeFilter() receipts.Filter {
	return receipts.Filter{
		DateFrom:       f.dateFrom,
		DateTo:         f.dateTo,
		AmountFrom:     f.amountFrom,
		AmountTo:       f.amountTo,
		OrganizationID: f.orgID,
		CustomerID:     f.custID,
		Status:         f.status,
		ActionSetFrom:  f.actionSetFrom,
		ActionSetTo:    f.actionSetTo,
	}
}

// parseReceiptFilter читает условия расширенного отбора из query
// parameters. Невалидные значения игнорируются (параметр отбрасывается),
// чтобы скопированный URL не «ломался» частично. Значение status
// проверяется по фиксированному списку статусов проекта.
func parseReceiptFilter(r *http.Request) receiptFilter {
	q := r.URL.Query()

	var f receiptFilter
	f.dateFrom = parseDateParam(q.Get("date_from"))
	f.dateTo = parseDateParam(q.Get("date_to"))
	f.amountFrom = parseAmountParam(q.Get("amount_from"))
	f.amountTo = parseAmountParam(q.Get("amount_to"))
	f.orgID = parseIDParam(q.Get("organization_id"))
	f.custID = parseIDParam(q.Get("customer_id"))
	f.actionSetFrom = parseDateParam(q.Get("action_set_from"))
	f.actionSetTo = parseDateParam(q.Get("action_set_to"))

	if status := q.Get("status"); status != "" && validReceiptFilterStatus(status) {
		f.status = status
	}
	return f
}

// validateReceiptFilterPair проверяет связь организация → контрагент
// на сервере (не только в UI). Контрагент обязан принадлежать выбранной
// организации; иначе параметр customer_id отбрасывается. Несуществующий
// контрагент тоже отбрасывается.
func validateReceiptFilterPair(f receiptFilter, custs []*customers.Customer) receiptFilter {
	if f.custID == 0 {
		return f
	}

	var found *customers.Customer
	for _, c := range custs {
		if c.ID == f.custID {
			found = c
			break
		}
	}
	if found == nil {
		f.custID = 0
		return f
	}
	if f.orgID > 0 && found.OrganizationID != f.orgID {
		f.custID = 0
	}
	return f
}

// buildFilterData собирает модель панели фильтра для шаблона: текущие
// значения, список статусов и JSON-справочники для пикеров.
func buildFilterData(f receiptFilter, orgs []*organizations.Organization, custs []*customers.Customer) *ui.FilterData {
	fd := &ui.FilterData{
		HasFilter:      f.active(),
		Open:           f.active(),
		DateFrom:       f.dateFrom,
		DateTo:         f.dateTo,
		Status:         f.status,
		Statuses:       filterStatusOptions(),
		OrganizationID: f.orgID,
		CustomerID:     f.custID,
		ActionSetFrom:  f.actionSetFrom,
		ActionSetTo:    f.actionSetTo,
	}
	if f.amountFrom != nil {
		fd.AmountFrom = strconv.FormatFloat(*f.amountFrom, 'f', -1, 64)
	}
	if f.amountTo != nil {
		fd.AmountTo = strconv.FormatFloat(*f.amountTo, 'f', -1, 64)
	}

	for _, o := range orgs {
		if o.ID == f.orgID {
			fd.OrganizationName = o.Name
			break
		}
	}
	for _, c := range custs {
		if c.ID == f.custID {
			fd.CustomerName = c.Name
			break
		}
	}

	orgOptions := filterOrgOptions(orgs)
	customerOptions := filterCustomerOptions(custs)

	fd.OrgsJSON, _ = common.ToJSON(orgOptions)
	fd.CustomersJSON, _ = common.ToJSON(customerOptions)
	fd.PayloadJSON, _ = common.ToJSON(receiptFilterPayload{
		Open:      f.active(),
		OrgID:     f.orgID,
		CustID:    f.custID,
		Orgs:      orgOptions,
		Customers: customerOptions,
	})
	if fd.PayloadJSON == "" {
		fd.PayloadJSON = "{}"
	}
	return fd
}

// receiptFilterPayload — данные компонента receiptsFilter (Alpine),
// сериализованные на сервере. JS только разбирает их из data-filter.
type receiptFilterPayload struct {
	Open      bool                   `json:"open"`
	OrgID     int64                  `json:"orgId"`
	CustID    int64                  `json:"customerId"`
	Orgs      []filterOrgOption      `json:"orgs"`
	Customers []filterCustomerOption `json:"customers"`
}

type filterOrgOption struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type filterCustomerOption struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	OrganizationID int64  `json:"organization_id"`
}

func filterOrgOptions(orgs []*organizations.Organization) []filterOrgOption {
	if orgs == nil {
		orgs = []*organizations.Organization{}
	}
	opts := make([]filterOrgOption, 0, len(orgs))
	for _, o := range orgs {
		opts = append(opts, filterOrgOption{ID: o.ID, Name: o.Name})
	}
	return opts
}

func filterCustomerOptions(custs []*customers.Customer) []filterCustomerOption {
	if custs == nil {
		custs = []*customers.Customer{}
	}
	opts := make([]filterCustomerOption, 0, len(custs))
	for _, c := range custs {
		opts = append(opts, filterCustomerOption{ID: c.ID, Name: c.Name, OrganizationID: c.OrganizationID})
	}
	return opts
}

// filterStatusOptions строит список статусов для <select> панели фильтра:
// value и label совпадают (это и есть отображаемое значение статуса).
// Пустая опция «Все статусы» добавляется в шаблоне.
func filterStatusOptions() []ui.StatusOption {
	opts := make([]ui.StatusOption, 0, len(receiptFilterStatuses))
	for _, s := range receiptFilterStatuses {
		opts = append(opts, ui.StatusOption{Value: s, Label: s})
	}
	return opts
}

func parseDateParam(s string) string {
	if s == "" {
		return ""
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return ""
	}
	return s
}

func parseAmountParam(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return nil
	}
	return &v
}

func parseIDParam(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil || v <= 0 {
		return 0
	}
	return v
}

func validReceiptFilterStatus(status string) bool {
	for _, s := range receiptFilterStatuses {
		if s == status {
			return true
		}
	}
	return false
}
