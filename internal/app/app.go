package app

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"sync"
	"time"

	"Orders/internal/customers"
	"Orders/internal/database"
	"Orders/internal/gotify"
	"Orders/internal/organizations"
	"Orders/internal/products"
	"Orders/internal/receipts"
	"Orders/internal/sessions"
	"Orders/internal/users"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type App struct {
	config *Config
	log    zerolog.Logger

	db            *sql.DB
	filesDB       *sql.DB
	users         *users.Store
	identity      *users.IdentityService
	sessions      *sessions.Store
	customers     *customers.Store
	organizations *organizations.Store
	products      *products.Store
	receipts      *receipts.Store
	receiptFiles  *receipts.FileStore

	// notifier — отправитель уведомлений Gotify. nil, если рассылка не
	// настроена: вызовы notify становятся no-op.
	notifier gotifySender

	router *chi.Mux
	server *http.Server

	orgKeys   map[string]string
	orgKeysMu sync.RWMutex

	// usersMu сериализует операции, способные уменьшить число
	// администраторов (удаление пользователя, снятие прав), чтобы проверка
	// инварианта «последний администратор» и запись были атомарны.
	usersMu sync.Mutex
}

// Server hardening: таймауты защищают от Slowloris и зависших операций ввода-вывода.
const (
	serverReadHeaderTimeout = 5 * time.Second
	serverReadTimeout       = 120 * time.Second
	serverWriteTimeout      = 120 * time.Second
	serverIdleTimeout       = 60 * time.Second
	serverMaxHeaderBytes    = 1 << 20
)

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

	filesDB, err := database.OpenPath(config.FilesDatabasePath)
	if err != nil {
		db.Close()
		return nil, err
	}
	filesSchema := NewFilesSchema()
	if err := filesSchema.RunMigrations(filesDB); err != nil {
		filesDB.Close()
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

	logger := NewLogger(false)

	var notifier gotifySender
	if config.Gotify.Enabled() {
		notifier = gotify.New(gotify.Config{
			URL:      config.Gotify.URL,
			Priority: config.Gotify.Priority,
			Tokens:   config.Gotify.Tokens,
		}, &http.Client{Timeout: gotifyHTTPTimeout})
	}

	app := &App{
		config:        config,
		log:           logger,
		db:            db,
		filesDB:       filesDB,
		users:         usersStore,
		identity:      identity,
		sessions:      sessionStore,
		customers:     customers.NewStore(db),
		organizations: orgStore,
		products:      products.NewStore(db),
		receipts:      receipts.NewStore(db),
		receiptFiles:  receipts.NewFileStore(filesDB),
		notifier:      notifier,
		orgKeys:       orgKeys,
	}

	app.router = app.NewRouter()

	app.server = &http.Server{
		Addr:              config.HTTPAddress,
		Handler:           app.Handler(),
		ReadHeaderTimeout: serverReadHeaderTimeout,
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
		MaxHeaderBytes:    serverMaxHeaderBytes,
	}

	return app, nil
}

// Handler возвращает HTTP-приложение целиком.
//
// При непустом BasePath роутер монтируется под префиксом:
// публикация меняет внешний путь, но сам chi-router продолжает
// работать с внутренними маршрутами приложения (/receipts, /api/...).
// Без префикса возвращается роутер без изменений.
func (a *App) Handler() http.Handler {
	if a.basePath() == "" {
		return a.router
	}
	return http.StripPrefix(a.basePath(), a.router)
}

// basePath возвращает нормализованный префикс приложения.
func (a *App) basePath() string {
	if a.config == nil || a.config.BasePath == "" {
		return ""
	}
	return a.config.BasePath
}

// URL добавляет базовый префикс к внутреннему абсолютному пути.
//
// Контракт: path — это путь приложения без BasePath (/receipts/1,
// /static/js/app.js). Код, вызывающий URL, обязан передавать чистый
// внутренний путь, а не результат другого вызова a.URL().
//
// При пустом префиксе путь возвращается без изменений.
func (a *App) URL(path string) string {
	base := a.basePath()
	if base == "" || path == "" || !strings.HasPrefix(path, "/") {
		return path
	}
	return base + path
}

func (a *App) Run() error {
	defer a.db.Close()
	defer a.filesDB.Close()
	return a.server.ListenAndServe()
}
