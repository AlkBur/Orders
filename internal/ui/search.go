package ui

// SearchMode задаёт поведение поиска списка.
//
// В будущем могут появиться новые режимы (SearchDisabled, SearchServer)
// без изменения API.
type SearchMode uint8

const (
	SearchManual SearchMode = iota
	SearchLive
)

// IsLive сообщает, работает ли поиск по мере ввода.
func (m SearchMode) IsLive() bool {
	return m == SearchLive
}

// String возвращает читаемое значение режима для логов и диагностики.
func (m SearchMode) String() string {
	switch m {
	case SearchLive:
		return "live"
	case SearchManual:
		return "manual"
	default:
		return "unknown"
	}
}

// SearchData — модель блока поиска списка.
//
// URL — базовый путь списка, к которому добавляется параметр ?q=... .
// Query — текущее значение поискового запроса (значение поля q).
// Placeholder — подсказка, показываемая в пустом поле ввода.
// Mode — режим поиска: SearchLive включает поиск по мере ввода.
// MinLength — минимальная длина запроса для автоматического поиска;
// 0 — платформенный дефолт (задаётся в static/js/search.js).
type SearchData struct {
	URL         string
	Query       string
	Placeholder string
	Mode        SearchMode
	MinLength   int
}
