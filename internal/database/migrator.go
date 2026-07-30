package database

import (
	"database/sql"
	"fmt"
)

var knownOldVersions = map[int]bool{}

func ensureSystemInfoTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS system_info(
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`)
	return err
}

func getVersion(db *sql.DB) (int, error) {
	var value int
	err := db.QueryRow(`
		SELECT value
		FROM system_info
		WHERE key = 'schema_version'
	`).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return value, err
}

func setVersion(db *sql.DB, version int) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO system_info(key, value)
		VALUES('schema_version', ?)
	`, version)
	if err != nil {
		return fmt.Errorf("set schema version %d: %w", version, err)
	}
	return nil
}

func CurrentVersion(migrations []Migration) int {
	v := InitialSchemaVersion
	for _, m := range migrations {
		if m.Version > v {
			v = m.Version
		}
	}
	return v
}
