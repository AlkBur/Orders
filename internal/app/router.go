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

	// Публичные маршруты
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

	// Требуют авторизации
	r.Group(func(r chi.Router) {

		r.Use(func(next http.Handler) http.Handler {
			return RequireAuth(a.users, a.config.Secret, next)
		})

		r.Post("/logout", a.Logout)

		r.Get("/orders", a.OrdersPage)

		// Следующие обработчики появятся позже
		// r.Get("/products", a.Products)
		// r.Get("/customers", a.Customers)
		// r.Get("/orders", a.Orders)
	})

	return r
}
