package app

import (
	"database/sql"
	"html/template"
	"net/http"

	"Orders/internal/customers"
	"Orders/internal/database"
	"Orders/internal/organizations"
	"Orders/internal/sessions"
	"Orders/internal/users"

	"github.com/go-chi/chi/v5"
)

type Integration struct {
	Name string
}

type App struct {
	config *Config

	db            *sql.DB
	users         *users.Store
	sessions      *sessions.Store
	customers     *customers.Store
	organizations *organizations.Store

	router *chi.Mux
	server *http.Server

	templates    map[string]*template.Template
	integrations map[string]*Integration
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

	integrations := make(map[string]*Integration, len(config.API.Keys))
	for _, k := range config.API.Keys {
		integrations[k.Key] = &Integration{Name: k.Name}
	}

	app := &App{
		config:        config,
		db:            db,
		users:         usersStore,
		sessions:      sessionStore,
		customers:     customers.NewStore(db),
		organizations: organizations.NewStore(db),
		templates:     make(map[string]*template.Template),
		integrations:  integrations,
	}

	for _, page := range []string{
		"login",
		"orders",
		"set-password",
		"menu",
		"organizations",
		"organization_card",
		"customers",
		"customer_card",
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
