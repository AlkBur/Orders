package users

import (
	"database/sql"

	"Orders/internal/database"
)

func init() {
	database.RegisterSchema(InitSchema)
}

func InitSchema(db *sql.DB) error {
	const query = `
CREATE TABLE IF NOT EXISTS users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,

    login           TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL DEFAULT '',

    is_admin        INTEGER NOT NULL DEFAULT 0,

    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

	_, err := db.Exec(query)
	return err
}
