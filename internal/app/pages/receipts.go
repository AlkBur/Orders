package pages

import (
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
	Number       string
	Date         string
	Organization string
	Customer     string
	Total        string
	Status       string

	CanEdit bool
	CanSend bool

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

type ReceiptCardPage struct {
	Header         ui.HeaderData
	Alert          *ui.AlertData
	CanEdit        bool
	CanSend        bool
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
