package sessions

import (
	"database/sql"

	"Orders/internal/database"
)

func init() {
	database.RegisterSchema(InitSchema)
}

func InitSchema(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS sessions (
    id              TEXT PRIMARY KEY,
    user_id         INTEGER REFERENCES users(id) ON DELETE SET NULL,
    flash_type      TEXT NOT NULL DEFAULT '',
    flash_message   TEXT NOT NULL DEFAULT '',
    values_json     TEXT NOT NULL DEFAULT '{}',
    user_agent      TEXT NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at      DATETIME NOT NULL
)
`)
	return err
}
