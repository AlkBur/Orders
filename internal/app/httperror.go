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
