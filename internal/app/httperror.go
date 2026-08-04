package app

import (
	"encoding/json"
	"net/http"

	"Orders/internal/ui"
)

// WriteValidationResponse serializes ValidationResponse using the transport
// selected by the platform. Сегодня транспорт — JSON; способ доставки —
// внутренняя деталь платформы.
func WriteValidationResponse(w http.ResponseWriter, r *http.Request, vr ValidationResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(vr)
}

func (a *App) InternalError(w http.ResponseWriter, err error) {
	// TODO: log error

	http.Error(
		w,
		"Internal Server Error",
		http.StatusInternalServerError,
	)
}

func (a *App) BadRequest(w http.ResponseWriter, msg string) {
	http.Error(w, msg, http.StatusBadRequest)
}

func (a *App) Unauthorized(w http.ResponseWriter) {
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

func (a *App) Forbidden(w http.ResponseWriter) {
	http.Error(w, "Forbidden", http.StatusForbidden)
}

// RenderInfrastructureError — единственная точка доставки инфраструктурных
// ошибок (некорректный запрос, превышение лимита). Guard'ы и RateLimiter
// не знают про HTML, HTMX и JSON.
// Для Fragment-запроса отдаёт JSON InfrastructureResponse с реальным
// HTTP-статусом; для обычного запроса перерисовывает форму входа с тем же
// статусом.
func (a *App) RenderInfrastructureError(w http.ResponseWriter, r *http.Request, status int, title string, msgs []string) {
	NoCache(w)
	if ResponseModeFromRequest(r) == Fragment {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(NewInfrastructureResponse(status, title, msgs))
		return
	}
	a.RenderPageStatus(w, r, ResponseModeFromRequest(r), status, a.loginPageData("", &ui.AlertData{
		Type:     ui.AlertError,
		Messages: msgs,
	}))
}

func NoCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}
