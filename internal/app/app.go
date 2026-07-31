package app

import (
	"context"
	"database/sql"
	"html/template"
	"net/http"
	"sync"

	"Orders/internal/customers"
	"Orders/internal/database"
	"Orders/internal/organizations"
	"Orders/internal/products"
	"Orders/internal/receipts"
	"Orders/internal/sessions"
	"Orders/internal/users"

	"github.com/go-chi/chi/v5"
)

type App struct {
	config *Config

	db            *sql.DB
	users         *users.Store
	identity      *users.IdentityService
	sessions      *sessions.Store
	customers     *customers.Store
	organizations *organizations.Store
	products      *products.Store
	receipts      *receipts.Store

	router *chi.Mux
	server *http.Server

	templates map[string]*template.Template

	orgKeys   map[string]string
	orgKeysMu sync.RWMutex
}

func New(configPath string) (*App, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	db, err := database.OpenPath(config.DatabasePath)
	if err != nil {
		return nil, err
	}

	schema := NewSchema()
	if err := schema.RunMigrations(db); err != nil {
		db.Close()
		return nil, err
	}

	usersStore := users.NewStore(db)

	if err := users.Seed(usersStore); err != nil {
		return nil, err
	}

	sessionStore := sessions.NewStore(db)
	orgStore := organizations.NewStore(db)

	orgKeys, err := orgStore.LoadAPIKeys(context.Background())
	if err != nil {
		db.Close()
		return nil, err
	}

	identity := users.NewIdentityService()
	if err := identity.Load(context.Background(), usersStore); err != nil {
		db.Close()
		return nil, err
	}

	app := &App{
		config:        config,
		db:            db,
		users:         usersStore,
		identity:      identity,
		sessions:      sessionStore,
		customers:     customers.NewStore(db),
		organizations: orgStore,
		products:      products.NewStore(db),
		receipts:      receipts.NewStore(db),
		templates:     make(map[string]*template.Template),
		orgKeys:       orgKeys,
	}

	for _, page := range []string{
		"login",
		"set-password",
		"receipts",
		"receipt_card",
		"organizations",
		"organization_card",
		"customers",
		"customer_card",
		"products",
		"product_card",
		"users",
		"user_card",
	} {
		tmpl, err := LoadTemplates(page)
		if err != nil {
			return nil, err
		}
		app.templates[page] = tmpl
	}

	app.router = app.NewRouter()

	app.server = &http.Server{
		Addr:    config.HTTPAddress,
		Handler: app.router,
	}

	return app, nil
}

func (a *App) Run() error {
	defer a.db.Close()
	return a.server.ListenAndServe()
}
