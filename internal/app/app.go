package app

import (
	"Orders/internal/database"
	"Orders/internal/sessions"
	"Orders/internal/users"
	"database/sql"
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type App struct {
	config *Config

	db        *sql.DB
	users     *users.Store
	sessions  *sessions.Store

	router    *chi.Mux
	server    *http.Server

	templates map[string]*template.Template
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

	usersStore := users.NewStore(db)

	if err := users.Seed(usersStore); err != nil {
		return nil, err
	}

	sessionStore := sessions.NewStore(db)

	app := &App{
		config:    config,
		db:        db,
		users:     usersStore,
		sessions:  sessionStore,
		templates: make(map[string]*template.Template),
	}

	for _, page := range []string{
		"login",
		"orders",
		"set-password",
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
