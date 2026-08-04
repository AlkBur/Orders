package app

import (
	"errors"
	"net/http"
)

// Секции файла:
//
//   - Login      — защита /login от переполнения формы.
//   - Upload     — зарезервировано для загрузки файлов (multipart-лимиты).
//   - Integration— зарезервировано для интеграционных эндпоинтов.
//
// Guard'ы возвращают только инфраструктурные статусы (400/413) через
// RenderInfrastructureError и не знают про HTML и HTMX.

const (
	msgInvalidRequest  = "Invalid request"
	msgRequestTooLarge = "Request too large"
)

// loginGuardLimits — пределы защиты формы входа. Вынесены в переменную,
// чтобы числа не размазывались по коду.
var loginGuardLimits = struct {
	BodyBytes    int64
	MaxFields    int
	MaxValues    int
	MaxParamName int
	MaxLogin     int
	MaxPassword  int
}{
	BodyBytes:    4 * 1024,
	MaxFields:    8,
	MaxValues:    16,
	MaxParamName: 64,
	MaxLogin:     64,
	MaxPassword:  128,
}

// loginRequestGuard ограничивает тело, число полей и длину значений формы
// входа. Слишком большое тело → 413; структурно некорректная форма → 400.
// Работает до RateLimiter, поэтому отклоняет переполненные запросы раньше
// счётчика попыток.
func (a *App) loginRequestGuard() func(http.Handler) http.Handler {
	l := loginGuardLimits
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, l.BodyBytes)
			if err := r.ParseForm(); err != nil {
				var maxBytesErr *http.MaxBytesError
				if errors.As(err, &maxBytesErr) {
					a.RenderInfrastructureError(w, r, http.StatusRequestEntityTooLarge, "Запрос слишком большой", []string{msgRequestTooLarge})
					return
				}
				a.RenderInfrastructureError(w, r, http.StatusBadRequest, "Некорректный запрос", []string{msgInvalidRequest})
				return
			}
			if len(r.PostForm) > l.MaxFields {
				a.RenderInfrastructureError(w, r, http.StatusBadRequest, "Некорректный запрос", []string{msgInvalidRequest})
				return
			}
			values := 0
			for name, vv := range r.PostForm {
				if len(name) > l.MaxParamName {
					a.RenderInfrastructureError(w, r, http.StatusBadRequest, "Некорректный запрос", []string{msgInvalidRequest})
					return
				}
				values += len(vv)
			}
			if values > l.MaxValues {
				a.RenderInfrastructureError(w, r, http.StatusBadRequest, "Некорректный запрос", []string{msgInvalidRequest})
				return
			}
			login := r.PostForm.Get("login")
			password := r.PostForm.Get("password")
			if len(login) > l.MaxLogin || len(password) > l.MaxPassword {
				a.RenderInfrastructureError(w, r, http.StatusBadRequest, "Некорректный запрос", []string{msgInvalidRequest})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
