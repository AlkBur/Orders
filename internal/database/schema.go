package databese

import "database/sql"

func InitSchema(db *sql.DB) error {

	const schema = `
CREATE TABLE IF NOT EXISTS app (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    version INTEGER NOT NULL
);

INSERT OR IGNORE INTO app(id, version)
VALUES (1, 1);
`

	_, err := db.Exec(schema)
	return err
}
