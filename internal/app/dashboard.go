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
}

type dashboardData struct {
	Title      string
	Header     ui.HeaderData
	Stats      dashboardStats
	RecentList ui.ListData
	Modules    []dashboardModule
}

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

	var rows []ui.ListRow
	for _, o := range recent {
		rows = append(rows, ui.ListRow{
			URL: "/organizations/" + strconv.FormatInt(o.ID, 10),
			Cells: []string{
				o.Name,
				orgStatus(o.Active),
			},
		})
	}

	pageFS, err := fs.Sub(TemplateFS(), "pages/dashboard")
	if err != nil {
		a.InternalError(w, err)
		return
	}

	data := dashboardData{
		Title:  "Панель управления",
		Header: pageHeader(r, "Панель управления"),
		Stats: dashboardStats{
			Total:  total,
			Active: active,
		},
		RecentList: ui.ListData{
			Columns: []ui.ListColumn{
				{Label: "Название"},
				{Label: "Статус"},
			},
			Rows:       rows,
			RenderMode: ui.RenderComfortable,
			Preset:     ui.ListWide,
		},
		Modules: []dashboardModule{
			{Name: "Контрагенты", Note: "Скоро"},
			{Name: "Товары", Note: "Скоро"},
			{Name: "Документы", Note: "Скоро"},
		},
	}

	if err := ui.RenderPage(w, TemplateFS(), pageFS, data); err != nil {
		a.InternalError(w, err)
	}
}
