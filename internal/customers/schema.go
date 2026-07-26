package customers

import (
	"database/sql"

	"Orders/internal/database"
)

func init() {
	database.RegisterSchema(InitSchema)
}

func InitSchema(db *sql.DB) error {
	const query = `
CREATE TABLE IF NOT EXISTS customers (
    uuid        TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    active      BOOLEAN NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`
	_, err := db.Exec(query)
	return err
}
