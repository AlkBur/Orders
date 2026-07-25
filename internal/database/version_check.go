package database

import "database/sql"

func checkVersion(db *sql.DB) (bool, error) {

	exists, err := hasSystemInfo(db)
	if err != nil {
		return false, err
	}

	// первая база или старая база без system_info
	if !exists {
		return false, nil
	}

	version, err := loadVersion(db)
	if err != nil {
		return false, err
	}

	return version == SchemaVersion, nil
}

func hasSystemInfo(db *sql.DB) (bool, error) {

	var count int

	err := db.QueryRow(`
SELECT COUNT(*)
FROM sqlite_master
WHERE type='table'
  AND name='system_info'
`).Scan(&count)

	if err != nil {
		return false, err
	}

	return count == 1, nil
}

func loadVersion(db *sql.DB) (int, error) {

	var version int

	err := db.QueryRow(`
SELECT value
FROM system_info
WHERE key='schema_version'
`).Scan(&version)

	return version, err
}

func saveVersion(db *sql.DB) error {

	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS system_info(
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
)
`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
INSERT OR REPLACE INTO system_info(key, value)
VALUES('schema_version', ?)
`, SchemaVersion)

	return err
}
