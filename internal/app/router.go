package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (a *App) NewRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(SessionMiddleware(a.sessions))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusFound)
	})

	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	r.Get("/login", a.LoginPage)
	r.Post("/login", a.Login)

	r.Handle(
		"/static/*",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.FS(StaticFS())),
		),
	)

	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return RequireAuth(a.sessions, a.users, next)
		})

		r.Get("/set-password", a.SetPasswordPage)
		r.Post("/set-password", a.SetPasswordSubmit)
		r.Post("/logout", a.Logout)

		r.Group(func(r chi.Router) {
			r.Use(RequirePassword)

			r.Get("/orders", a.OrdersPage)
		})
	})

	return r
}
