package app

import (
	"fmt"
	"net/http"
)

type App struct {
	config *Config
	server *http.Server
}

func New() (*App, error) {

	config, err := LoadConfig("config.json")
	if err != nil {
		return nil, err
	}

	router := NewRouter()

	return &App{
		config: config,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", config.HTTPPort),
			Handler: router,
		},
	}, nil
}

func (a *App) Run() error {
	return a.server.ListenAndServe()
}
