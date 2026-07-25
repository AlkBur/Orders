package databese

import (
	"database/sql"
	"orders/internal/users"
)

func InitSchema(db *sql.DB) error {
	if err := users.InitSchema(db); err != nil {
		return err
	}
	return nil
}
