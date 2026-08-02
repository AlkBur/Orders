package ui

// SearchData — модель блока поиска списка.
//
// URL — базовый путь списка, к которому добавляется параметр ?q=... .
// Query — текущее значение поискового запроса (значение поля q).
// Placeholder — подсказка, показываемая в пустом поле ввода.
// Live — живой поиск по мере ввода после минимальной длины запроса
// (минимальная длина задаётся в static/js/search.js).
type SearchData struct {
	URL         string
	Query       string
	Placeholder string
	Live        bool
}
