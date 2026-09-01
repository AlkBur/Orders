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
// TargetID — CSS-селектор элемента, который подменяет htmx при отправке
// формы. Пустое значение эквивалентно "#list" (живой поиск и так
// заменяет только список). Для страниц с панелью фильтра задаётся
// идентификатор обёртки, включающей форму и список, чтобы сервер мог
// перерисовать состояние кнопки «Фильтр».
// Filter — данные панели расширенного отбора; nil — панель отсутствует.
type SearchData struct {
	URL         string
	Query       string
	Placeholder string
	Mode        SearchMode
	MinLength   int
	TargetID    string
	Filter      *FilterData
}

// StatusOption — пункт фиксированного списка статусов панели фильтра.
// Value — значение параметра status в URL и <select>, Label — отображаемое
// имя. Список статусов определяет сервер, а не шаблон.
type StatusOption struct {
	Value string
	Label string
}

// FilterData — модель панели расширенного отбора списка.
//
// Заполняется сервером из query parameters: значения полей — единственный
// источник состояния (обновление страницы и копирование URL сохраняют
// условия). Не хранить состояние фильтра только в Alpine/JavaScript.
//
// HasFilter — установлен ли хотя бы один параметр расширенного фильтра
// (текстовый поиск q не учитывается). Определяет активное состояние
// кнопки «Фильтр».
// Open — раскрыта ли панель при первичной отрисовке.
// Statuses — фиксированный список доступных статусов.
// OrgsJSON / CustomersJSON — JSON-справочники для пикеров организации
// и контрагента (готовые к встраиванию в Alpine x-data).
// PayloadJSON — сводный JSON панели для атрибута data-filter компонента
// receiptsFilter (open, orgId, customerId и справочники): собирается
// на сервере один раз, JS только разбирает его.
type FilterData struct {
	HasFilter        bool
	Open             bool
	DateFrom         string
	DateTo           string
	AmountFrom       string
	AmountTo         string
	OrganizationID   int64
	OrganizationName string
	CustomerID       int64
	CustomerName     string
	Status           string
	Statuses         []StatusOption
	ActionSetFrom    string
	ActionSetTo      string
	OrgsJSON         string
	CustomersJSON    string
	PayloadJSON      string
}
