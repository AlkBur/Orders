package users

import (
	"database/sql"
	"errors"
)

// Seed гарантирует наличие администратора в системе.
func Seed(store *Store) error {
	_, err := store.FindAdmin()

	switch {
	case errors.Is(err, sql.ErrNoRows):
		admin := &User{
			Login:   "admin",
			IsAdmin: true,
		}

		return store.Create(admin)

	case err != nil:
		return err

	default:
		return nil
	}
}
