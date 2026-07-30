package pages

import (
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
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
// Отдельная модель от Button: может эволюционировать независимо.
type FAB struct {
	Icon string
	Text string
	URL  string
}

// Temporary solution.
// FAB will move to common LayoutModel when it appears.
type FABProvider interface {
	FAB() *FAB
}

type HeaderData struct {
	AppName  string
	Username string
}

type ToolbarData struct {
	Title   string
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

func (OrganizationsPage) FAB() *FAB {
	return &FAB{Icon: "plus", URL: "#", Text: "Добавить"}
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

func icon(name string) template.HTML {
	switch name {
	case "house":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>`
	case "plus":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>`
	case "search":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>`
	case "save":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg>`
	case "edit":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>`
	case "trash":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>`
	case "arrow_left":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="19" y1="12" x2="5" y2="12"/><polyline points="12 19 5 12 12 5"/></svg>`
	case "arrow_right":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>`
	case "user":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>`
	case "logout":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>`
	case "building":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="2" width="16" height="20" rx="2" ry="2"/><path d="M9 22v-4h6v4"/><line x1="8" y1="6" x2="10" y2="6"/><line x1="14" y1="6" x2="16" y2="6"/><line x1="8" y1="10" x2="10" y2="10"/><line x1="14" y1="10" x2="16" y2="10"/><line x1="8" y1="14" x2="10" y2="14"/><line x1="14" y1="14" x2="16" y2="14"/></svg>`
	case "package":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M16 16v4H8v-4"/><rect x="3" y="4" width="18" height="12" rx="2"/><path d="M12 10V4"/><path d="M8 8h8"/></svg>`
	case "chart":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>`
	case "settings":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`
	case "people":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`
	default:
		return ``
	}
}

func getFAB(data any) *FAB {
	if p, ok := data.(FABProvider); ok {
		return p.FAB()
	}
	return nil
}

func showcaseFuncs() template.FuncMap {
	return template.FuncMap{
		"icon":   icon,
		"getFAB": getFAB,
	}
}

func loadShowcaseTemplates(page string, tmplFS fs.FS) (*template.Template, error) {
	return template.New("").Funcs(showcaseFuncs()).ParseFS(tmplFS,
		"layout/*.html",
		"components/*.html",
		"pages/"+page+".html",
	)
}

func ShowcaseTemplatesDir() string {
	return filepath.Join("internal", "app", "templates")
}

func renderShowcase(w http.ResponseWriter, page string, data any, tmplFS fs.FS) error {
	tmpl, err := loadShowcaseTemplates(page, tmplFS)
	if err != nil {
		return err
	}
	return tmpl.ExecuteTemplate(w, "base", data)
}

func HandleDashboard(tmplFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := DashboardPage{
			Title:  "Панель управления",
			Header: HeaderData{AppName: "Orders", Username: "Администратор"},
			Toolbar: ToolbarData{
				Title: "Панель управления",
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
			Header: HeaderData{AppName: "Orders", Username: "Администратор"},
			Toolbar: ToolbarData{
				Title: "Организации",
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
			Header: HeaderData{AppName: "Orders", Username: "Администратор"},
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
