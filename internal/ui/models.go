package ui

import "html/template"

// ButtonStyle — стиль кнопки.
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
