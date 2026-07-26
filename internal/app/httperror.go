package app

import "net/http"

func (a *App) InternalError(w http.ResponseWriter, err error) {
	// TODO: log error

	http.Error(
		w,
		"Internal Server Error",
		http.StatusInternalServerError,
	)
}

func NoCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}
