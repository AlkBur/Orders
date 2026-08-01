package app

import (
	"io/fs"
	"net/http"
	"strconv"

	"Orders/internal/organizations"
	"Orders/internal/ui"
)

type dashboardStats struct {
	Total  int
	Active int
}

type dashboardModule struct {
	Name string
	Note string
	URL  string
	Hero bool
}

type dashboardData struct {
	Title      string
	Header     ui.HeaderData
	Stats      dashboardStats
	RecentDocs ui.ListData
	RecentOrgs ui.ListData
	Modules    []dashboardModule
}

// DashboardPage — «Рабочий стол» администратора. Первично это навигационный
// экран: карточки модулей ведут на их страницы. Виджеты со свежими данными
// («Недавние документы», «Последние организации», «Статистика») вторичны.
// Toolbar и FAB намеренно отсутствуют.
func (a *App) DashboardPage(w http.ResponseWriter, r *http.Request) {
	NoCache(w)

	total, err := a.organizations.Count(r.Context(), "")
	if err != nil {
		a.InternalError(w, err)
		return
	}
	active, err := a.organizations.CountActive(r.Context())
	if err != nil {
		a.InternalError(w, err)
		return
	}
	recent, err := a.organizations.List(r.Context(), organizations.ListOptions{
		Limit:   5,
		OrderBy: "created_at",
	})
	if err != nil {
		a.InternalError(w, err)
		return
	}

	var orgRows []ui.ListRow
	for _, o := range recent {
		orgRows = append(orgRows, ui.ListRow{
			URL: "/organizations/" + strconv.FormatInt(o.ID, 10),
			Cells: []string{
				o.Name,
				orgStatus(o.Active),
			},
		})
	}

	var docRows []ui.ListRow
	docColumns := []ui.ListColumn{
		{Label: "Номер"},
		{Label: "Дата"},
		{Label: "Клиент"},
		{Label: "Организация"},
		{Label: "Сумма"},
		{Label: "Статус"},
	}
	if a.receipts != nil {
		docs, err := a.receipts.List(r.Context())
		if err != nil {
			a.InternalError(w, err)
			return
		}
		if len(docs) > 5 {
			docs = docs[:5]
		}
		for _, d := range docs {
			total, _ := d.DisplayValue("Total")
			status, _ := d.DisplayValue("Status")
			docRows = append(docRows, ui.ListRow{
				URL: d.URL(),
				Cells: []string{
					d.Number,
					d.Date.Format("2006-01-02"),
					d.CustomerName,
					d.OrganizationName,
					total,
					status,
				},
			})
		}
	}

	pageFS, err := fs.Sub(TemplateFS(), "pages/dashboard")
	if err != nil {
		a.InternalError(w, err)
		return
	}

	data := dashboardData{
		Title:  "Рабочий стол",
		Header: pageHeader(r, "Рабочий стол"),
		Stats: dashboardStats{
			Total:  total,
			Active: active,
		},
		RecentDocs: ui.ListData{
			Columns:    docColumns,
			Rows:       docRows,
			RenderMode: ui.RenderComfortable,
		},
		RecentOrgs: ui.ListData{
			Columns: []ui.ListColumn{
				{Label: "Название"},
				{Label: "Статус"},
			},
			Rows:       orgRows,
			RenderMode: ui.RenderComfortable,
			Preset:     ui.ListWide,
		},
		Modules: []dashboardModule{
			{Name: "Документы", Note: "Работа с товарными чеками", URL: RouteReceipts, Hero: true},
			{Name: "Организации", Note: "Справочник организаций", URL: "/organizations"},
			{Name: "Контрагенты", Note: "Справочник контрагентов", URL: "/customers"},
			{Name: "Товары", Note: "Справочник товаров", URL: "/products"},
			{Name: "Пользователи", Note: "Пользователи и права", URL: "/users"},
		},
	}

	if err := ui.RenderPage(w, TemplateFS(), pageFS, data); err != nil {
		a.InternalError(w, err)
	}
}
