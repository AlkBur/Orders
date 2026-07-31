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
	ListDefault      ListPreset = 0
	ListOrganizations ListPreset = 1
	ListEmployees    ListPreset = 2
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
)

func (t FieldType) String() string {
	switch t {
	case FieldNumber:
		return "number"
	case FieldSelect:
		return "select"
	case FieldCheckbox:
		return "checkbox"
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
	Name  string
	Label string
	Type  FieldType
	Value string
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

type DashboardStats struct {
	Total  int
	Active int
}

type DashboardPage struct {
	Title      string
	Header     HeaderData
	Toolbar    ToolbarData
	Stats      DashboardStats
	RecentList ListData
}

type OrganizationsPage struct {
	Title   string
	Header  HeaderData
	Toolbar ToolbarData
	Search  SearchData
	List    ListData
}

func (OrganizationsPage) FAB() *ui.FAB {
	return &ui.FAB{Icon: "plus", URL: "#", Text: "Добавить"}
}

type OrganizationPage struct {
	Title         string
	Header        HeaderData
	Name          string
	Description   string
	Active        bool
	EmployeeCount int
	CreatedAt     string
	Employees     ListData
}

func renderShowcase(w http.ResponseWriter, page string, data any, tmplFS fs.FS) error {
	pageFS, err := fs.Sub(tmplFS, "pages/"+page)
	if err != nil {
		return err
	}
	return ui.RenderPage(w, tmplFS, pageFS, data)
}

func demoHeader(section string) HeaderData {
	return HeaderData{
		Section:  section,
		Username: "Администратор",
		Menu: []MenuItem{
			{ID: "logout", Label: "Выход", Icon: "logout", URL: "/logout"},
		},
	}
}

func HandleDashboard(tmplFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := DashboardPage{
			Title:  "Панель управления",
			Header: demoHeader("Панель управления"),
			Toolbar: ToolbarData{
				Buttons: []Button{
					{Style: ButtonPrimary, Text: "Создать", URL: "#", Icon: "plus"},
				},
			},
			Stats: DashboardStats{Total: 12, Active: 8},
			RecentList: ListData{
				Columns: []ListColumn{
					{Label: "Название"},
					{Label: "Статус"},
					{Label: "Город"},
				},
				Rows: []ListRow{
					{URL: "/ui/organizations/1", Cells: []string{"ООО Ромашка", "Активна", "Москва"}},
					{URL: "/ui/organizations/2", Cells: []string{"ИП Иванов", "Активна", "СПб"}},
					{URL: "/ui/organizations/3", Cells: []string{"ЗАО ТехноСервис", "Неактивна", "Казань"}},
				},
				RenderMode: RenderComfortable,
				Preset:     ListOrganizations,
			},
		}
		if err := renderShowcase(w, "dashboard", data, tmplFS); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func HandleOrganizations(tmplFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := OrganizationsPage{
			Title:  "Организации",
			Header: demoHeader("Организации"),
			Toolbar: ToolbarData{
				Buttons: []Button{
					{Style: ButtonPrimary, Text: "Добавить", URL: "#", Icon: "plus"},
				},
			},
			Search: SearchData{Placeholder: "Поиск организаций...", Value: ""},
			List: ListData{
				Columns: []ListColumn{
					{Label: "Название"},
					{Label: "ИНН"},
					{Label: "Статус"},
					{Label: "Город"},
				},
				Rows: []ListRow{
					{URL: "/ui/organizations/1", Cells: []string{"ООО Ромашка", "7701123456", "Активна", "Москва"}},
					{URL: "/ui/organizations/2", Cells: []string{"ИП Иванов", "7801234567", "Активна", "Санкт-Петербург"}},
					{URL: "/ui/organizations/3", Cells: []string{"ЗАО ТехноСервис", "1601234567", "Неактивна", "Казань"}},
					{URL: "/ui/organizations/4", Cells: []string{"ООО Альфа", "7702123456", "Активна", "Москва"}},
					{URL: "/ui/organizations/5", Cells: []string{"ООО Бета", "5401123456", "Активна", "Новосибирск"}},
				},
				RenderMode: RenderComfortable,
				Preset:     ListOrganizations,
			},
		}
		if err := renderShowcase(w, "organizations", data, tmplFS); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func HandleOrganization(tmplFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := OrganizationPage{
			Title:  "ООО Ромашка",
			Header: demoHeader("Организации"),
			Name:   "ООО Ромашка",
			Description: "Оптовая торговля цветами и растениями. " +
				"Компания основана в 2010 году, имеет филиалы в 5 городах.",
			Active:        true,
			EmployeeCount: 24,
			CreatedAt:     "15 марта 2024",
			Employees: ListData{
				Columns: []ListColumn{
					{Label: "Имя"},
					{Label: "Должность"},
					{Label: "Телефон"},
				},
				Rows: []ListRow{
					{URL: "#", Cells: []string{"Иванов Иван", "Директор", "+7 (495) 123-45-67"}},
					{URL: "#", Cells: []string{"Петрова Анна", "Бухгалтер", "+7 (495) 123-45-68"}},
					{URL: "#", Cells: []string{"Сидоров Алексей", "Менеджер", "+7 (495) 123-45-69"}},
				},
				RenderMode: RenderComfortable,
				Preset:     ListEmployees,
			},
		}
		if err := renderShowcase(w, "organization", data, tmplFS); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
