package app

import (
	"database/sql"
	"fmt"
	"net/http"
	databese "orders/internal/database"
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
	db, err := databese.Open()
	if err != nil {
		return nil, err
	}

	//Подготовка и проверка таблиц базы
	if err := databese.InitSchema(db); err != nil {
		db.Close()
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
