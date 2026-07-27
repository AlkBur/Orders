package database

import (
	"context"
	"database/sql"
)

type Migration struct {
	Version int
	Name    string
	Up      func(ctx context.Context, tx *sql.Tx) error
}
