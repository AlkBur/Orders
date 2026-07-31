package app

import "net/http"

// ResponseMode определяет формат ответа сервера: полная страница или
// фрагмент. Способ определения режима (HTMX или иной транспорт) является
// инфраструктурной деталью.
type ResponseMode int

const (
	FullPage ResponseMode = iota
	Fragment
)

// ResponseModeFromRequest определяет режим ответа по признакам запроса.
func ResponseModeFromRequest(r *http.Request) ResponseMode {
	if r.Header.Get("HX-Request") == "true" {
		return Fragment
	}
	return FullPage
}
