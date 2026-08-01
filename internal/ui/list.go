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
	ListDefault ListPreset = 0
	ListWide    ListPreset = 1
	ListCompact ListPreset = 2
)

func (p ListPreset) String() string {
	switch p {
	case ListWide:
		return "Wide"
	case ListCompact:
		return "Compact"
	default:
		return "Default"
	}
}

func (p ListPreset) Modifier() string {
	switch p {
	case ListWide:
		return "list--wide"
	case ListCompact:
		return "list--compact"
	default:
		return ""
	}
}

type ListColumn struct {
	Label string
	Flex  int
}

// RowAction — действие строки списка. Библиотека не знает бизнес-смысла
// действия: она только рисует иконку-кнопку с URL.
type RowAction struct {
	ID    string
	Icon  string
	Label string
	URL   string
}

type ListRow struct {
	URL     string
	Cells   []string
	Actions []RowAction
}

type ListData struct {
	Columns    []ListColumn
	Rows       []ListRow
	RenderMode RenderMode
	Preset     ListPreset
}
