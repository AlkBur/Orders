package users

import (
	"database/sql"
	"errors"

	"Orders/internal/common"
)

func Seed(store *Store) error {
	_, err := store.FindAdmin()

	switch {
	case errors.Is(err, sql.ErrNoRows):
		uuid, err := common.GenerateUUID()
		if err != nil {
			return err
		}

		admin := &User{
			UUID:    uuid,
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
