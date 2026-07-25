package app

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func OpenDatabase() (*sql.DB, error) {

	if err := os.MkdirAll("data", 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", filepath.Join("data", "base.db"))
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
