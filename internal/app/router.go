package app

import (
	"net/http"

	"Orders/internal/app/pages"
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

	r.Get("/login", a.LoginPage)
	r.Post("/login", a.Login)

	r.Handle(
		"/static/*",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.FS(StaticFS())),
		),
	)

	// UI Catalog — компоненты платформы
	r.Route("/ui", func(r chi.Router) {
		showcaseFS := TemplateFS()
		r.Get("/", pages.HandleCatalog(showcaseFS))
	})

	// Integration API — обмен между системами (UUID-based)
	r.Route("/api/integration/organizations/{oid}", func(r chi.Router) {
		r.Use(a.RequireOrganizationAPIKey)
		r.Put("/customers", a.HandlePutCustomers)
		r.Put("/products", a.HandlePutProducts)
	})

	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return RequireAuth(a.sessions, a.identity, next)
		})

		r.Get(RouteSetPassword, a.SetPasswordPage)
		r.Post(RouteSetPassword, a.SetPasswordSubmit)
		r.Post("/logout", a.Logout)

		r.Group(func(r chi.Router) {
			r.Use(RequirePassword)

			r.Group(func(r chi.Router) {
				r.Use(RequireAdmin)
				r.Get(RouteDashboard, a.DashboardPage)
			})

			// Receipts
			r.Get(RouteReceipts, a.ReceiptsPage)
			r.Get("/receipts/new", a.ReceiptCard)
			r.Post("/receipts", a.ReceiptSave)
			r.Get("/receipts/{id}", a.ReceiptCard)
			r.Post("/receipts/{id}", a.ReceiptSave)
			r.Post("/receipts/{id}/send", a.ReceiptSubmit)
			r.Post("/receipts/{id}/delete", a.ReceiptDelete)

			// Customers
			r.Get("/customers", a.CustomersPage)
			r.Get("/customers/new", a.CustomerCard)
			r.Post("/customers", a.CustomerSave)
			r.Route("/organizations/{oid}/customers", func(r chi.Router) {
				r.Get("/", a.CustomersPage)
				r.Get("/new", a.CustomerCard)
				r.Post("/", a.CustomerSave)
				r.Get("/{id}", a.CustomerCard)

				r.Group(func(r chi.Router) {
					r.Use(RequireAdmin)
					r.Post("/{id}", a.CustomerSave)
				})
			})

			// Products
			r.Get("/products", a.ProductsPage)
			r.Get("/products/new", a.ProductCard)
			r.Post("/products", a.ProductSave)
			r.Route("/organizations/{oid}/products", func(r chi.Router) {
				r.Get("/", a.ProductsPage)
				r.Get("/new", a.ProductCard)
				r.Post("/", a.ProductSave)
				r.Get("/{id}", a.ProductCard)

				r.Group(func(r chi.Router) {
					r.Use(RequireAdmin)
					r.Post("/{id}", a.ProductSave)
					r.Delete("/{id}", a.ProductDelete)
				})
			})

			// Organizations
			r.Get("/organizations", a.OrganizationsPage)
			r.Get("/organizations/new", a.OrganizationCard)
			r.Post("/organizations", a.OrganizationSave)
			r.Get("/organizations/{id}", a.OrganizationCard)
			r.Post("/organizations/{id}", a.OrganizationSave)

			// Users
			r.Route("/users", func(r chi.Router) {
				r.Use(RequireAdmin)
				r.Get("/", a.UsersPage)
				r.Get("/new", a.UserCard)
				r.Post("/", a.UserSave)
				r.Get("/{id}", a.UserCard)
				r.Post("/{id}", a.UserSave)
				r.Delete("/{id}", a.UserDelete)
			})
		})
	})

	return r
}
