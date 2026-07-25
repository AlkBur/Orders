package app

import (
	"Orders/internal/database"
	"Orders/internal/users"
	"database/sql"
	"fmt"
	"net/http"
)

type App struct {
	config *Config
	db     *sql.DB
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

	store := users.NewStore(db)

	if err := users.Seed(store); err != nil {
		return nil, err
	}

	router := NewRouter()

	return &App{
		config: config,
		db:     db,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", config.HTTPPort),
			Handler: router,
		},
	}, nil
}

func (a *App) Run() error {
	defer a.db.Close()
	return a.server.ListenAndServe()
}
