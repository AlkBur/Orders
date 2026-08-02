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
	ListView    ui.ListView
	Dialog      ui.DialogData
	CodeSamples map[string]string
}

func (CatalogPage) FAB() *ui.FAB {
	return &ui.FAB{Icon: "plus", URL: "#", Text: "Добавить"}
}

func HandleCatalog(tmplFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := CatalogPage{
			Title: "Компоненты",
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
			Search: ui.SearchData{URL: "/ui", Placeholder: "Поиск компонентов...", Query: "", Live: true},
			Buttons: []ui.Button{
				{Style: ui.ButtonDefault, Text: "Обычная", URL: "#"},
				{Style: ui.ButtonPrimary, Text: "Основная", URL: "#", Icon: "plus"},
				{Style: ui.ButtonOutline, Text: "Контурная", URL: "#", Icon: "edit"},
				{Style: ui.ButtonDanger, Text: "Опасная", URL: "#", Icon: "trash"},
			},
			Icons: []string{
				"house", "plus", "search", "save", "edit", "trash",
				"arrow_left", "arrow_right", "user", "lock", "logout",
				"building", "package", "chart", "settings", "people",
				"more_vertical",
			},
			Fields: []ui.Field{
				{Name: "name", Label: "Название", Type: ui.FieldText, Value: "ООО Ромашка", Icon: "user"},
				{Name: "inn", Label: "ИНН", Type: ui.FieldText},
				{Name: "employees", Label: "Сотрудники", Type: ui.FieldNumber, Value: "24"},
				{Name: "secret", Label: "Пароль", Type: ui.FieldPassword, Autocomplete: "current-password", Icon: "lock"},
				{Name: "org", Label: "Организация", Type: ui.FieldSelect, Placeholder: "Выберите организацию",
					Options: []ui.SelectOption{
						{Value: "1", Label: "ООО Ромашка"},
						{Value: "2", Label: "ИП Иванов"},
						{Value: "3", Label: "ЗАО ТехноСервис", Disabled: true},
					}},
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
					{Cells: []string{"ООО Ромашка", "7701123456", "Активна"},
						Actions: []ui.RowAction{{ID: "edit", Icon: "edit", Label: "Открыть", URL: "#"}}},
					{Cells: []string{"ИП Иванов", "7801234567", "Активна"},
						Actions: []ui.RowAction{{ID: "select", Icon: "check", Label: "Выбрать", URL: "#"}}},
					{Cells: []string{"ЗАО ТехноСервис", "1601234567", "Неактивна"}},
				},
				RenderMode: ui.RenderComfortable,
				Preset:     ui.ListWide,
			},
			Dialog: ui.DialogData{
				ID:      "demo-dialog",
				Title:   "Подтверждение",
				Content: template.HTML("<p>Удалить запись?</p>"),
			},
			CodeSamples: map[string]string{
				"toolbar": `{{template "list_view" .ListView}}`,
				"button":  `{{template "toolbar_button" .}}`,
				"card": `{{template "card_open" "Заголовок"}}
  ...содержимое...
{{template "card_close"}}`,
				"list":   `{{template "list" .List}}`,
				"search": `{{template "search" .Search}}`,
				"form":   `{{template "form_group" .Fields}}`,
				"menu":   `{{template "app_menu" .Header}}`,
				"alert":  `{{template "alert" .Alert}}`,
				"dialog": `{{template "dialog" .Dialog}}`,
			},
		}
		data.ListView = ui.ListView{
			Toolbar: &data.Toolbar,
			Search:  &data.Search,
			List:    data.List,
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
