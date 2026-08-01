package ui

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
