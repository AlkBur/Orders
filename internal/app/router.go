package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (a *App) NewRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr)

	r.Use(middleware.Recoverer)

	// Публичные маршруты
	r.Get("/login", a.LoginPage)
	r.Post("/login", a.Login)

	// Требуют авторизации
	r.Group(func(r chi.Router) {

		r.Use(func(next http.Handler) http.Handler {
			return RequireAuth(a.users, a.config.Secret, next)
		})

		r.Post("/logout", a.Logout)

		r.Get("/", homeHandler)

		// Следующие обработчики появятся позже
		// r.Get("/products", a.Products)
		// r.Get("/customers", a.Customers)
		// r.Get("/orders", a.Orders)
	})

	return r
}

func homeHandler(w http.ResponseWriter, r *http.Request) {

	tmpl, err := LoadTemplates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Title string
	}{
		Title: "Orders Server",
	}

	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
