package pages

import (
	"html/template"
	"io/fs"
	"net/http"

	"Orders/internal/ui"
)

// CatalogPage — Component Catalog: единая страница /ui с демонстрацией
// компонентов платформы и примерами использования.
type CatalogPage struct {
	Title       string
	Header      ui.HeaderData
	Toolbar     ui.ToolbarData
	Search      ui.SearchData
	Buttons     []ui.Button
	Icons       []string
	Fields      []ui.Field
	Alerts      []ui.AlertData
	List        ui.ListData
	Dialog      ui.DialogData
	CodeSamples map[string]string
}

func (CatalogPage) FAB() *ui.FAB {
	return &ui.FAB{Icon: "plus", URL: "#", Text: "Добавить"}
}

func HandleCatalog(tmplFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := CatalogPage{
			Title:  "Компоненты",
			Header: ui.HeaderData{
				Section:  "Компоненты",
				Username: "Администратор",
				Menu: []ui.MenuItem{
					{ID: "logout", Label: "Выход", Icon: "logout", URL: "/logout"},
				},
			},
			Toolbar: ui.ToolbarData{
				Buttons: []ui.Button{
					{Style: ui.ButtonPrimary, Text: "Создать", URL: "#", Icon: "plus"},
					{Style: ui.ButtonOutline, Text: "Экспорт", URL: "#", Icon: "save"},
				},
			},
			Search: ui.SearchData{Placeholder: "Поиск компонентов...", Value: ""},
			Buttons: []ui.Button{
				{Style: ui.ButtonDefault, Text: "Обычная", URL: "#"},
				{Style: ui.ButtonPrimary, Text: "Основная", URL: "#", Icon: "plus"},
				{Style: ui.ButtonOutline, Text: "Контурная", URL: "#", Icon: "edit"},
				{Style: ui.ButtonDanger, Text: "Опасная", URL: "#", Icon: "trash"},
			},
			Icons: []string{
				"house", "plus", "search", "save", "edit", "trash",
				"arrow_left", "arrow_right", "user", "logout",
				"building", "package", "chart", "settings", "people",
				"more_vertical",
			},
			Fields: []ui.Field{
				{Name: "name", Label: "Название", Type: ui.FieldText, Value: "ООО Ромашка"},
				{Name: "inn", Label: "ИНН", Type: ui.FieldText},
				{Name: "employees", Label: "Сотрудники", Type: ui.FieldNumber, Value: "24"},
				{Name: "secret", Label: "Пароль", Type: ui.FieldPassword, Autocomplete: "current-password"},
				{Name: "active", Label: "Активна", Type: ui.FieldCheckbox, Value: "true"},
			},
			Alerts: []ui.AlertData{
				{Type: ui.AlertInfo, Messages: []string{"Информационное сообщение"}},
				{Type: ui.AlertError, Messages: []string{"Пароль не может быть пустым.", "Пароли не совпадают."}},
			},
			List: ui.ListData{
				Columns: []ui.ListColumn{
					{Label: "Название"},
					{Label: "ИНН"},
					{Label: "Статус"},
				},
				Rows: []ui.ListRow{
					{URL: "#", Cells: []string{"ООО Ромашка", "7701123456", "Активна"}},
					{URL: "#", Cells: []string{"ИП Иванов", "7801234567", "Активна"}},
					{URL: "#", Cells: []string{"ЗАО ТехноСервис", "1601234567", "Неактивна"}},
				},
				RenderMode: ui.RenderComfortable,
				Preset:     ui.ListOrganizations,
			},
			Dialog: ui.DialogData{
				ID:      "demo-dialog",
				Title:   "Подтверждение",
				Content: template.HTML("<p>Удалить запись?</p>"),
			},
			CodeSamples: map[string]string{
				"toolbar": `{{template "toolbar_list" .}}`,
				"button":  `{{template "toolbar_button" .}}`,
				"card": `{{template "card_open" "Заголовок"}}
  ...содержимое...
{{template "card_close"}}`,
				"list":  `{{template "list" .List}}`,
				"form":  `{{template "form_group" .Fields}}`,
				"menu":  `{{template "app_menu" .Header}}`,
				"alert": `{{template "alert" .Alert}}`,
				"dialog": `{{template "dialog" .Dialog}}`,
			},
		}
		pageFS, err := fs.Sub(tmplFS, "pages/catalog")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := ui.RenderPage(w, tmplFS, pageFS, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
