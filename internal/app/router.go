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

	// Integration API — обмен между системами
	r.Route("/api/integration/organizations/{oid}", func(r chi.Router) {
		r.Use(a.RequireOrganizationAPIKey)
		r.Put("/customers", a.HandlePutCustomers)
		r.Put("/products", a.HandlePutProducts)
	})

	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return RequireAuth(a.sessions, a.users, next)
		})

		r.Get("/set-password", a.SetPasswordPage)
		r.Post("/set-password", a.SetPasswordSubmit)
		r.Post("/logout", a.Logout)

		r.Group(func(r chi.Router) {
			r.Use(RequirePassword)

			r.Get("/", a.MenuPage)
			r.Get("/orders", a.OrdersPage)

			// Customers — глобальный список (admin)
			r.Get("/customers", a.CustomersPage)

			// Customers — организационный контекст
			r.Route("/organizations/{oid}/customers", func(r chi.Router) {
				r.Get("/", a.CustomersPage)
				r.Get("/{id}", a.CustomerCard)

				r.Group(func(r chi.Router) {
					r.Use(RequireAdmin)
					r.Post("/", a.CustomerSave)
					r.Delete("/{id}", a.CustomerDelete)
				})
			})

			// Products — глобальный список (admin)
			r.Get("/products", a.ProductsPage)

			// Products — организационный контекст
			r.Route("/organizations/{oid}/products", func(r chi.Router) {
				r.Get("/", a.ProductsPage)
				r.Get("/{id}", a.ProductCard)

				r.Group(func(r chi.Router) {
					r.Use(RequireAdmin)
					r.Post("/", a.ProductSave)
					r.Delete("/{id}", a.ProductDelete)
				})
			})

			// Organizations
			r.Get("/organizations", a.OrganizationsPage)
			r.Get("/organizations/{id}", a.OrganizationCard)
			r.Post("/organizations/{id}", a.OrganizationSave)
		})
	})

	return r
}
