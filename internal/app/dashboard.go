package app

import (
	"io/fs"
	"net/http"

	"Orders/internal/ui"
	"Orders/internal/ui/display"
)

// dashboardModule — навигационная карточка «Рабочего стола».
// Count уже содержит готовое представление (display.FormatNumber);
// Note — необязательное описание, рендерится только если задано.
type dashboardModule struct {
	Name  string
	Icon  string
	URL   string
	Count string
	Hero  bool
	Note  string
}

type dashboardData struct {
	Title   string
	Header  ui.HeaderData
	Modules []dashboardModule
}

// DashboardPage — «Рабочий стол» администратора. Это лаунчер приложения:
// только навигационные карточки модулей (иконка + название + количество
// объектов). Никаких виджетов со свежими данными, Toolbar и FAB
// отсутствуют намеренно.
func (a *App) DashboardPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	docCount, err := a.receipts.Count(r.Context())
	if err != nil {
		a.InternalError(w, err)
		return
	}
	orgCount, err := a.organizations.Count(r.Context())
	if err != nil {
		a.InternalError(w, err)
		return
	}
	customerCount, err := a.customers.Count(r.Context())
	if err != nil {
		a.InternalError(w, err)
		return
	}
	productCount, err := a.products.Count(r.Context())
	if err != nil {
		a.InternalError(w, err)
		return
	}
	userCount, err := a.users.Count(r.Context())
	if err != nil {
		a.InternalError(w, err)
		return
	}

	pageFS, err := fs.Sub(TemplateFS(), "pages/dashboard")
	if err != nil {
		a.InternalError(w, err)
		return
	}

	data := dashboardData{
		Title:  "Рабочий стол",
		Header: pageHeader(r, "Рабочий стол"),
		Modules: []dashboardModule{
			{
				Name:  "Документы",
				Icon:  "file-text",
				URL:   RouteReceipts,
				Count: display.FormatNumber(docCount),
				Hero:  true,
				Note:  "Работа с товарными чеками",
			},
			{Name: "Организации", Icon: "building", URL: "/organizations", Count: display.FormatNumber(orgCount)},
			{Name: "Контрагенты", Icon: "people", URL: "/customers", Count: display.FormatNumber(customerCount)},
			{Name: "Товары", Icon: "package", URL: "/products", Count: display.FormatNumber(productCount)},
			{Name: "Пользователи", Icon: "user", URL: "/users", Count: display.FormatNumber(userCount)},
		},
	}

	if err := ui.RenderPage(w, TemplateFS(), pageFS, data); err != nil {
		a.InternalError(w, err)
	}
}
