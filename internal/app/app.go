package app

import (
	"Orders/internal/database"
	"Orders/internal/users"
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type App struct {
	config *Config

	db    *sql.DB
	users *users.Store

	router *chi.Mux
	server *http.Server
}

func New() (*App, error) {
	config, err := LoadConfig("config.json")
	if err != nil {
		return nil, err
	}

	//Подготовка работы с базой
	db, err := database.Open()
	if err != nil {
		return nil, err
	}

	usersStore := users.NewStore(db)

	if err := users.Seed(usersStore); err != nil {
		return nil, err
	}

	app := &App{
		config: config,
		db:     db,
		users:  usersStore,
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
