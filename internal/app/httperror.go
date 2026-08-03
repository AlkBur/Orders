package app

import (
	"encoding/json"
	"net/http"
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

func NoCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}
