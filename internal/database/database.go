package database

import (
	"database/sql"
	"os"
	"path/filepath"

	"Orders/internal/database/search"

	_ "modernc.org/sqlite"
)

func Open() (*sql.DB, error) {
	return OpenPath(filepath.Join("data", "base.db"))
}

func OpenPath(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	// SQL-функции поиска регистрируются до открытия соединений:
	// новые коннекты драйвера получают их автоматически.
	search.RegisterFunctions()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
