package pages

import (
	"html/template"

	"Orders/internal/receipts"
	"Orders/internal/ui"
)

type ReceiptOrganizationOption struct {
	ID   int64
	Name string
}

// ReceiptListRow — готовая к выводу строка списка чеков.
// Верхняя часть описывает отображаемые данные документа,
// нижняя — доступные действия (URL и флаги готовности шаблона).
// Шаблон только выводит эти значения и не строит URL.
type ReceiptListRow struct {
	// Number, Organization, Customer — поля текстового поиска: сервер
	// подсвечивает совпадения (ui.MarkMatches) и передаёт готовый HTML.
	// Шаблон выводит их без дополнительного экранирования.
	Number       template.HTML
	Date         string
	Organization template.HTML
	Customer     template.HTML
	Total        string
	Status       string

	// StatusKey — семантический UI-ключ статуса (created, overdue, sent,
	// accepted, cancelled, processed, finished). Не является цветом:
	// фактический цвет задаёт CSS темой. Используется как класс
	// receipts-status is-<StatusKey> и одинаков для обеих тем.
	StatusKey string

	// Info — текст колонки «Инфо»: текущее действие (Удалить/Изменить),
	// когда оно установлено и ещё не получено 1С; иначе пустая строка.
	Info string

	CanEdit bool
	CanSend bool

	// CanSendAction — доступна ли кнопка «Действие» (только для
	// синхронизированных в 1С документов, uuid != "").
	// ActionURL — адрес модалки действия (data-dialog-url).
	CanSendAction bool
	ActionURL     string

	// CanMarkDeleted — доступна ли пометка документа на удаление
	// (только администратор и только для разрешённых статусов).
	// DeleteURL — целевой адрес POST-формы пометки; DeleteConfirm —
	// текст подтверждения (data-confirm) с номером документа.
	CanMarkDeleted bool
	DeleteURL      string
	DeleteConfirm  string

	// HasFiles — булев признак наличия прикреплённых файлов у документа.
	// Журналу требуется только факт существования файлов (показать кнопку
	// «Файлы»), а не их количество.
	HasFiles bool

	FilesURL string
	CopyURL  string
	SendURL  string
	ViewURL  string
	EditURL  string
}

// ReceiptsListPage — модель специализированного списка чеков.
type ReceiptsListPage struct {
	Page
	Header  ui.HeaderData
	Alert   *ui.AlertData
	Toolbar *ui.ToolbarData
	Search  *ui.SearchData
	Rows    []ReceiptListRow
	NewURL  string

	// HasMore / LoadMoreURL — keyset-пагинация списка. HasMore сообщает,
	// есть ли документы после текущей порции; LoadMoreURL — URL следующей
	// порции, который сервер встраивает в sentinel lazy loading.
	HasMore     bool
	LoadMoreURL string
}

func (p ReceiptsListPage) FAB() *ui.FAB {
	if p.NewURL == "" {
		return nil
	}
	return &ui.FAB{Icon: "plus", URL: p.NewURL, Text: "Добавить"}
}

type ReceiptCopyPage struct {
	ReceiptCardPage
}

// CanConfirmSend и ReturnURL — нулевые по умолчанию: обычный просмотр
// (view = read-only) не показывает действий отправки. Заполняются только
// на экране confirm (ReceiptSendConfirmPage).
type ReceiptCardPage struct {
	Header         ui.HeaderData
	Alert          *ui.AlertData
	CanEdit        bool
	CanSend        bool
	CanConfirmSend bool
	ReturnURL      string
	Title          string
	FormAction     string
	Card           ui.CardData
	Receipt        *receipts.Receipt
	Items          []receipts.ReceiptItem
	Orgs           []ReceiptOrganizationOption
	CustomersJSON  string
	ProductsJSON   string
	CustomerID     int64
	CustomerName   string
	OrganizationID int64
	Errors         map[string]string
	ErrorsJSON     string
	ItemsJSON      string
	CopySource     string

	// Files — файлы документа для просмотра (режим только для чтения).
	// Пустое значение = файлов нет, блок «Файлы» не выводится.
	Files []ReceiptFile
}

// ReceiptSendConfirmPage — экран подтверждения отправки документа в 1С
// (mode=send). Расширяет карточку документа: добавляет флаг готовности
// действия и точку возврата после отмены. Сама отправка происходит в POST
// /receipts/{id}/send, эта страница только показывает документ и просит
// подтвердить действие.
type ReceiptSendConfirmPage struct {
	ReceiptCardPage
	CanConfirmSend bool
	ReturnURL      string
}

// ReceiptFile — готовый к выводу файл документа.
type ReceiptFile struct {
	Name  string
	URL   string
	Icon  string
	Blank bool
}

// ReceiptFilesPage — модель окна «Файлы» чека (просмотр только для чтения:
// загрузка и удаление файлов происходят через Integration API).
type ReceiptFilesPage struct {
	Page
	Header  ui.HeaderData
	Receipt ReceiptHeader
	Files   []ReceiptFile
	BackURL string
}

// ReceiptHeader — сводка документа в окне «Файлы».
type ReceiptHeader struct {
	Number       string
	Date         string
	Organization string
	Total        string
}

// ReceiptActionPage — модель модального окна «Действие» для документа.
// Три варианта (Удалить / Изменить / Отмена) отправляются формой
// POST /receipts/{id}/action; CurrentAction показывает уже установленное
// действие (пустая строка — действия нет).
type ReceiptActionPage struct {
	Number        string
	CurrentAction string
	FormAction    string
}
