package pages

import (
	"html/template"
	"io/fs"
	"net/http"

	"Orders/internal/ui"
)

type ButtonStyle int

const (
	ButtonDefault  ButtonStyle = 0
	ButtonPrimary  ButtonStyle = 1
	ButtonOutline  ButtonStyle = 2
	ButtonDanger   ButtonStyle = 3
)

func (s ButtonStyle) String() string {
	switch s {
	case ButtonPrimary:
		return "primary"
	case ButtonOutline:
		return "outline"
	case ButtonDanger:
		return "danger"
	default:
		return "default"
	}
}

type RenderMode int

const (
	RenderComfortable RenderMode = 0
	RenderCompact     RenderMode = 1
	RenderCards       RenderMode = 2
)

func (m RenderMode) String() string {
	switch m {
	case RenderCompact:
		return "Compact"
	case RenderCards:
		return "Cards"
	default:
		return "Comfortable"
	}
}

type ListPreset int

const (
	ListDefault       ListPreset = 0
	ListOrganizations ListPreset = 1
	ListEmployees     ListPreset = 2
)

func (p ListPreset) String() string {
	switch p {
	case ListOrganizations:
		return "Organizations"
	case ListEmployees:
		return "Employees"
	default:
		return "Default"
	}
}

func (p ListPreset) Modifier() string {
	switch p {
	case ListOrganizations:
		return "list--organizations"
	case ListEmployees:
		return "list--employees"
	default:
		return ""
	}
}

type FieldType int

const (
	FieldText     FieldType = 0
	FieldNumber   FieldType = 1
	FieldSelect   FieldType = 2
	FieldCheckbox FieldType = 3
	FieldPassword FieldType = 4
)

func (t FieldType) String() string {
	switch t {
	case FieldNumber:
		return "number"
	case FieldSelect:
		return "select"
	case FieldCheckbox:
		return "checkbox"
	case FieldPassword:
		return "password"
	default:
		return "text"
	}
}

type Button struct {
	Style ButtonStyle
	Text  string
	URL   string
	Icon  string
}

func (b Button) Class() string {
	switch b.Style {
	case ButtonPrimary:
		return "button is-primary"
	case ButtonOutline:
		return "button is-outlined"
	case ButtonDanger:
		return "button is-danger"
	default:
		return "button"
	}
}

// FAB — платформенный компонент для главного действия на мобильных.
// Модель и функции доступны через пакет ui.

// MenuItem — пункт меню приложения (AppMenu) в шапке.
type MenuItem struct {
	ID        string
	Label     string
	Icon      string
	URL       string
	Danger    bool
	Separator bool
}

type HeaderData struct {
	Section  string
	Username string
	Menu     []MenuItem
}

type ToolbarData struct {
	Buttons []Button
}

type SearchData struct {
	Placeholder string
	Value       string
}

type Field struct {
	Name        string
	Label       string
	Type        FieldType
	Value       string
	Readonly    bool
	Required    bool
	Autofocus   bool
	Autocomplete string
}

type AlertType int

const (
	AlertInfo AlertType = iota
	AlertSuccess
	AlertWarning
	AlertError
)

func (t AlertType) String() string {
	switch t {
	case AlertSuccess:
		return "success"
	case AlertWarning:
		return "warning"
	case AlertError:
		return "error"
	default:
		return "info"
	}
}

type AlertData struct {
	Type     AlertType
	Messages []string
}

type ListColumn struct {
	Label string
	Flex  int
}

type ListRow struct {
	URL   string
	Cells []string
}

type ListData struct {
	Columns    []ListColumn
	Rows       []ListRow
	RenderMode RenderMode
	Preset     ListPreset
}

type DialogData struct {
	ID      string
	Title   string
	Content template.HTML
}

// CatalogPage — Component Catalog: единая страница /ui с демонстрацией
// компонентов платформы и примерами использования.
type CatalogPage struct {
	Title       string
	Header      HeaderData
	Toolbar     ToolbarData
	Search      SearchData
	Buttons     []Button
	Icons       []string
	Fields      []Field
	Alerts      []AlertData
	List        ListData
	Dialog      DialogData
	CodeSamples map[string]string
}

func (CatalogPage) FAB() *ui.FAB {
	return &ui.FAB{Icon: "plus", URL: "#", Text: "Добавить"}
}

func HandleCatalog(tmplFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := CatalogPage{
			Title:  "Компоненты",
			Header: HeaderData{
				Section:  "Компоненты",
				Username: "Администратор",
				Menu: []MenuItem{
					{ID: "logout", Label: "Выход", Icon: "logout", URL: "/logout"},
				},
			},
			Toolbar: ToolbarData{
				Buttons: []Button{
					{Style: ButtonPrimary, Text: "Создать", URL: "#", Icon: "plus"},
					{Style: ButtonOutline, Text: "Экспорт", URL: "#", Icon: "save"},
				},
			},
			Search: SearchData{Placeholder: "Поиск компонентов...", Value: ""},
			Buttons: []Button{
				{Style: ButtonDefault, Text: "Обычная", URL: "#"},
				{Style: ButtonPrimary, Text: "Основная", URL: "#", Icon: "plus"},
				{Style: ButtonOutline, Text: "Контурная", URL: "#", Icon: "edit"},
				{Style: ButtonDanger, Text: "Опасная", URL: "#", Icon: "trash"},
			},
			Icons: []string{
				"house", "plus", "search", "save", "edit", "trash",
				"arrow_left", "arrow_right", "user", "logout",
				"building", "package", "chart", "settings", "people",
				"more_vertical",
			},
			Fields: []Field{
				{Name: "name", Label: "Название", Type: FieldText, Value: "ООО Ромашка"},
				{Name: "inn", Label: "ИНН", Type: FieldText},
				{Name: "employees", Label: "Сотрудники", Type: FieldNumber, Value: "24"},
				{Name: "secret", Label: "Пароль", Type: FieldPassword, Autocomplete: "current-password"},
				{Name: "active", Label: "Активна", Type: FieldCheckbox, Value: "true"},
			},
			Alerts: []AlertData{
				{Type: AlertInfo, Messages: []string{"Информационное сообщение"}},
				{Type: AlertError, Messages: []string{"Пароль не может быть пустым.", "Пароли не совпадают."}},
			},
			List: ListData{
				Columns: []ListColumn{
					{Label: "Название"},
					{Label: "ИНН"},
					{Label: "Статус"},
				},
				Rows: []ListRow{
					{URL: "#", Cells: []string{"ООО Ромашка", "7701123456", "Активна"}},
					{URL: "#", Cells: []string{"ИП Иванов", "7801234567", "Активна"}},
					{URL: "#", Cells: []string{"ЗАО ТехноСервис", "1601234567", "Неактивна"}},
				},
				RenderMode: RenderComfortable,
				Preset:     ListOrganizations,
			},
			Dialog: DialogData{
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
