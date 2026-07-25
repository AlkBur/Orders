package users

import "database/sql"

func InitSchema(db *sql.DB) error {

	const query = `
CREATE TABLE IF NOT EXISTS users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    login           TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    display_name    TEXT NOT NULL,
    disabled        INTEGER NOT NULL DEFAULT 0
);
`

	_, err := db.Exec(query)
	return err
}
