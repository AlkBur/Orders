package database

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Open() (*sql.DB, error) {
	return OpenPath(filepath.Join("data", "base.db"))
}

func OpenPath(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	db, err := open(path)
	if err != nil {
		return nil, err
	}

	ok, err := checkVersion(db)
	if err != nil {
		db.Close()
		return nil, err
	}

	if !ok {
		db.Close()

		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}

		db, err = open(path)
		if err != nil {
			return nil, err
		}
	}

	if err := InitSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	if err := saveVersion(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func open(path string) (*sql.DB, error) {
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
