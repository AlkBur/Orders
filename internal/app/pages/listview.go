package pages

import "Orders/internal/ui"

// ListViewPage — модель страницы-списка: тулбар, поиск и список.
// NewURL — адрес главного действия (кнопка «Добавить» и FAB).
// Пустой NewURL скрывает оба.
type ListViewPage struct {
	Title  string
	Header ui.HeaderData
	List   ui.ListView
	Alert  *ui.AlertData
	NewURL string
}

func (p ListViewPage) FAB() *ui.FAB {
	if p.NewURL == "" {
		return nil
	}
	return &ui.FAB{Icon: "plus", URL: p.NewURL, Text: "Добавить"}
}
